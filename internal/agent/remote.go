package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"log"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"
)

const remoteFrameTick = 25 * time.Millisecond

type remoteCommand struct {
	Enabled bool     `json:"enabled"`
	Name    string   `json:"name"`
	Size    int64    `json:"size"`
	Offset  int64    `json:"offset"`
	Data    string   `json:"data"`
	Digest  string   `json:"digest"`
	Type    string   `json:"type"`
	X       float64  `json:"x"`
	Y       float64  `json:"y"`
	DeltaX  int      `json:"delta_x"`
	DeltaY  int      `json:"delta_y"`
	Button  int      `json:"button"`
	Key     string   `json:"key"`
	Code    string   `json:"code"`
	Ctrl    bool     `json:"ctrl"`
	Alt     bool     `json:"alt"`
	Shift   bool     `json:"shift"`
	Meta    bool     `json:"meta"`
	Monitor int      `json:"monitor"`
	Text    string   `json:"text"`
	Profile string   `json:"profile"`
	Keys    []string `json:"keys"`
}

type remoteScreenState struct {
	sync.RWMutex
	monitor       int
	bounds        image.Rectangle
	quality       int
	maxWidth      int
	frameInterval time.Duration
}

func validRemoteSessionID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, ch := range value {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

func (a *Agent) startRemoteRelay(sessionID string) {
	desktopName := activeInputDesktopName()
	if a.secureRelayStarter != nil && strings.EqualFold(desktopName, "Winlogon") {
		if err := a.secureRelayStarter(sessionID); err == nil {
			log.Printf("[remote] delegated locked-screen relay to Winlogon worker")
			return
		} else {
			log.Printf("[remote] cannot start Winlogon relay worker: %v", err)
		}
	}
	// A Windows desktop is bound to an OS thread, not a Go goroutine.  Keep the
	// capture path on one thread for the complete relay lifetime so it remains
	// attached when Windows switches Default <-> Winlogon on lock/unlock.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	desktop := newRemoteDesktop()
	defer desktop.close()
	var protection remoteProtection
	defer protection.close()

	monitorCount := screenshot.NumActiveDisplays()
	if monitorCount == 0 {
		log.Printf("[remote] no active display is available")
		return
	}
	base, err := url.Parse(a.cfg.ServerURL)
	if err != nil {
		log.Printf("[remote] invalid server URL: %v", err)
		return
	}
	scheme := "ws"
	if base.Scheme == "https" {
		scheme = "wss"
	}
	query := url.Values{}
	query.Set("key", a.cfg.APIKey)
	query.Set("device_id", a.cfg.DeviceID)
	relayURL := fmt.Sprintf("%s://%s/ws/relay/%s/agent?%s", scheme, base.Host, sessionID, query.Encode())
	conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
	if err != nil {
		log.Printf("[remote] relay connection failed: %v", err)
		return
	}

	a.remoteMu.Lock()
	if a.remoteConn != nil {
		_ = a.remoteConn.Close()
	}
	a.remoteConn = conn
	a.remoteMu.Unlock()
	defer func() {
		releaseRemoteInputs()
		_ = conn.Close()
		a.remoteMu.Lock()
		if a.remoteConn == conn {
			a.remoteConn = nil
		}
		a.remoteMu.Unlock()
	}()

	currentDesktopName := activeInputDesktopName()
	state := &remoteScreenState{monitor: 0, bounds: screenshot.GetDisplayBounds(0), quality: 52, maxWidth: 1600, frameInterval: 60 * time.Millisecond}
	var writeMu sync.Mutex
	writeMessage := func(messageType int, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		return conn.WriteMessage(messageType, payload)
	}
	sendJSON := func(payload interface{}) error {
		data, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return marshalErr
		}
		return writeMessage(websocket.TextMessage, data)
	}
	sendScreenInfo := func() error {
		state.RLock()
		monitor, bounds := state.monitor, state.bounds
		state.RUnlock()
		monitors := make([]map[string]int, 0, monitorCount)
		for i := 0; i < monitorCount; i++ {
			item := screenshot.GetDisplayBounds(i)
			monitors = append(monitors, map[string]int{"index": i, "width": item.Dx(), "height": item.Dy()})
		}
		return sendJSON(map[string]interface{}{"type": "ready", "agent_version": a.version, "width": bounds.Dx(), "height": bounds.Dy(), "monitor": monitor, "monitors": monitors, "capabilities": map[string]bool{"file_transfer": true, "protection": runtime.GOOS == "windows"}})
	}
	if err := sendScreenInfo(); err != nil {
		return
	}
	mode := "normal worker"
	if a.secureDesktopOnly {
		mode = "direct Winlogon worker"
	}
	if desktopName == "" {
		desktopName = "tidak terdeteksi"
	}
	relayInfo := "Agent v" + a.version + ": desktop input Windows " + desktopName + " (" + mode + ")"
	log.Printf("[remote] %s", relayInfo)
	_ = sendJSON(map[string]string{"type": "relay_info", "message": relayInfo})

	readDone := make(chan struct{})
	stopping := make(chan struct{})
	defer close(stopping)
	conn.SetReadLimit(128 * 1024)
	inputCommands := make(chan remoteCommand, 128)
	go func() {
		defer close(readDone)
		var transfer remoteFileTransfer
		defer transfer.abort()
		for {
			deadline := time.Time{}
			if transfer.file != nil {
				deadline = transfer.updated.Add(45 * time.Second)
			}
			_ = conn.SetReadDeadline(deadline)
			messageType, payload, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			if messageType != websocket.TextMessage {
				continue
			}
			var command remoteCommand
			if json.Unmarshal(payload, &command) != nil {
				continue
			}
			switch command.Type {
			case "file_start", "file_chunk", "file_end", "file_cancel":
				response, transferErr := transfer.handle(command)
				if transferErr != nil {
					transfer.abort()
					_ = sendJSON(map[string]string{"type": "file_error", "message": transferErr.Error()})
				} else {
					_ = sendJSON(response)
				}
			case "set_monitor":
				if command.Monitor >= 0 && command.Monitor < monitorCount {
					state.Lock()
					state.monitor = command.Monitor
					state.bounds = screenshot.GetDisplayBounds(command.Monitor)
					state.Unlock()
					_ = sendScreenInfo()
				}
			case "clipboard_set":
				if err := setRemoteClipboard(command.Text); err != nil {
					_ = sendJSON(map[string]string{"type": "clipboard_error", "message": err.Error()})
				}
			case "clipboard_get":
				text, clipboardErr := getRemoteClipboard()
				if clipboardErr != nil {
					_ = sendJSON(map[string]string{"type": "clipboard_error", "message": clipboardErr.Error()})
				} else {
					_ = sendJSON(map[string]string{"type": "clipboard", "text": text})
				}
			case "set_quality":
				state.Lock()
				switch command.Profile {
				case "smooth":
					state.quality, state.maxWidth, state.frameInterval = 42, 1280, 50*time.Millisecond
				case "sharp":
					state.quality, state.maxWidth, state.frameInterval = 68, 2560, 100*time.Millisecond
				default:
					state.quality, state.maxWidth, state.frameInterval = 52, 1600, 60*time.Millisecond
				}
				state.Unlock()
			default:
				select {
				case inputCommands <- command:
				case <-stopping:
					return
				}
			}
		}
	}()

	ticker := time.NewTicker(remoteFrameTick)
	defer ticker.Stop()
	var lastChecksum uint32
	hasChecksum := false
	lastSent := time.Time{}
	lastCapture := time.Time{}
	protectionDeadline := time.Time{}
	for {
		select {
		case <-readDone:
			return
		case command := <-inputCommands:
			if command.Type == "protection_heartbeat" {
				if !protectionDeadline.IsZero() {
					protectionDeadline = time.Now().Add(15 * time.Second)
					protection.heartbeat()
				}
				continue
			}
			if command.Type == "block_input" || command.Type == "privacy_mode" {
				err := protection.set(command.Type, command.Enabled)
				if err != nil {
					_ = sendJSON(map[string]string{"type": "protection_error", "message": err.Error()})
				}
				protectionDeadline = time.Time{}
				if protection.active() {
					protectionDeadline = time.Now().Add(15 * time.Second)
					protection.heartbeat()
				}
				_ = sendJSON(protection.state())
				continue
			}
			// The capture goroutine is pinned to a Windows OS thread and has
			// successfully attached that thread to the active input desktop. Run
			// all mouse/keyboard injection here as well; a separate goroutine can
			// otherwise remain on Default while capture is on Winlogon.
			if desktopErr := desktop.prepare(); desktopErr != nil {
				_ = sendJSON(map[string]string{"type": "input_error", "message": desktopErr.Error()})
				continue
			}
			if command.Type == "hotkey" {
				if inputErr := sendRemoteHotkey(command.Keys); inputErr != nil {
					_ = sendJSON(map[string]string{"type": "input_error", "message": inputErr.Error()})
				}
				continue
			}
			state.RLock()
			bounds := state.bounds
			state.RUnlock()
			if inputErr := handleRemoteInput(command, bounds); inputErr != nil {
				log.Printf("[remote] input ignored: %v", inputErr)
				_ = sendJSON(map[string]string{"type": "input_error", "message": inputErr.Error()})
			}
		case <-ticker.C:
			if !protectionDeadline.IsZero() && time.Now().After(protectionDeadline) {
				protection.close()
				protectionDeadline = time.Time{}
				_ = sendJSON(protection.state())
				_ = sendJSON(map[string]string{"type": "protection_error", "message": "Proteksi dilepas: heartbeat timeout."})
				return
			}
			activeDesktop := activeInputDesktopName()
			if activeDesktop != "" && !strings.EqualFold(activeDesktop, currentDesktopName) {
				currentDesktopName = activeDesktop
				protection.onDesktopChange(activeDesktop)
			}
			// On Windows this re-attaches the current OS thread to the desktop
			// that is actually receiving input. A SYSTEM console worker can then
			// follow Default <-> Winlogon transitions without dropping the relay.
			if desktopErr := desktop.prepare(); desktopErr != nil {
				secureDesktop := isSecureInputDesktop()
				if !a.secureDesktopOnly && a.secureRelayStarter != nil && secureDesktop {
					log.Printf("[remote] Winlogon desktop requires secure worker delegate: %v", desktopErr)
					_ = sendJSON(map[string]string{"type": "desktop_transition", "message": "Windows terkunci; menyambungkan kontrol lock screen…"})
					return
				}
				log.Printf("[remote] input desktop unavailable: %v", desktopErr)
				continue
			}
			state.RLock()
			bounds, quality, maxWidth, frameInterval := state.bounds, state.quality, state.maxWidth, state.frameInterval
			state.RUnlock()
			if time.Since(lastCapture) < frameInterval {
				continue
			}
			lastCapture = time.Now()
			img, captureErr := screenshot.CaptureRect(bounds)
			if captureErr != nil {
				log.Printf("[remote] screen capture failed: %v", captureErr)
				return
			}
			checksum := crc32.ChecksumIEEE(img.Pix)
			if hasChecksum && checksum == lastChecksum && time.Since(lastSent) < 2*time.Second {
				continue
			}
			frame, encodeErr := encodeRemoteFrameProfile(img, quality, maxWidth)
			if encodeErr != nil {
				log.Printf("[remote] frame encoding failed: %v", encodeErr)
				return
			}
			if err := writeMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
			lastChecksum, hasChecksum = checksum, true
			lastSent = time.Now()
		}
	}
}

func captureRemoteFrame(bounds image.Rectangle) ([]byte, error) {
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}
	return encodeRemoteFrame(img)
}

func encodeRemoteFrame(img image.Image) ([]byte, error) {
	return encodeRemoteFrameProfile(img, 52, 0)
}

func encodeRemoteFrameProfile(img image.Image, quality, maxWidth int) ([]byte, error) {
	if maxWidth > 0 && img.Bounds().Dx() > maxWidth {
		height := img.Bounds().Dy() * maxWidth / img.Bounds().Dx()
		resized := image.NewRGBA(image.Rect(0, 0, maxWidth, height))
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = resized
	}
	var frame bytes.Buffer
	if err := jpeg.Encode(&frame, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return frame.Bytes(), nil
}
