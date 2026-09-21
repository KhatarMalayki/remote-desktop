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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
	"golang.org/x/image/draw"
)

const remoteFrameTick = 25 * time.Millisecond

type remoteCommand struct {
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
	// A Windows desktop is bound to an OS thread, not a Go goroutine.  Keep the
	// capture path on one thread for the complete relay lifetime so it remains
	// attached when Windows switches Default <-> Winlogon on lock/unlock.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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
		return sendJSON(map[string]interface{}{"type": "ready", "width": bounds.Dx(), "height": bounds.Dy(), "monitor": monitor, "monitors": monitors})
	}
	if err := sendScreenInfo(); err != nil {
		return
	}

	readDone := make(chan struct{})
	go func() {
		// Keyboard/mouse injection needs its own stable desktop-bound thread.
		// Without this, Go may move the reader goroutine to a different thread
		// after OpenInputDesktop succeeds, which breaks input on the lock screen.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(readDone)
		for {
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
			case "hotkey":
				if inputErr := sendRemoteHotkey(command.Keys); inputErr != nil {
					_ = sendJSON(map[string]string{"type": "input_error", "message": inputErr.Error()})
				}
			default:
				state.RLock()
				bounds := state.bounds
				state.RUnlock()
				if inputErr := handleRemoteInput(command, bounds); inputErr != nil {
					log.Printf("[remote] input ignored: %v", inputErr)
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
	for {
		select {
		case <-readDone:
			return
		case <-ticker.C:
			// On Windows this re-attaches the current OS thread to the desktop
			// that is actually receiving input. A SYSTEM console worker can then
			// follow Default <-> Winlogon transitions without dropping the relay.
			if desktopErr := prepareRemoteDesktop(); desktopErr != nil {
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
