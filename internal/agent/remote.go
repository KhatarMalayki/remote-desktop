package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kbinani/screenshot"
)

const remoteFrameInterval = 250 * time.Millisecond

type remoteCommand struct {
	Type   string  `json:"type"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Button int     `json:"button"`
	Key    string  `json:"key"`
	Code   string  `json:"code"`
	Ctrl   bool    `json:"ctrl"`
	Alt    bool    `json:"alt"`
	Shift  bool    `json:"shift"`
	Meta   bool    `json:"meta"`
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
	if screenshot.NumActiveDisplays() == 0 {
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
		_ = conn.Close()
		a.remoteMu.Lock()
		if a.remoteConn == conn {
			a.remoteConn = nil
		}
		a.remoteMu.Unlock()
	}()

	bounds := screenshot.GetDisplayBounds(0)
	ready, _ := json.Marshal(map[string]interface{}{
		"type": "ready", "width": bounds.Dx(), "height": bounds.Dy(),
	})
	if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
		return
	}

	readDone := make(chan struct{})
	go func() {
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
			if json.Unmarshal(payload, &command) == nil {
				if inputErr := handleRemoteInput(command, bounds); inputErr != nil {
					log.Printf("[remote] input ignored: %v", inputErr)
				}
			}
		}
	}()

	ticker := time.NewTicker(remoteFrameInterval)
	defer ticker.Stop()
	for {
		select {
		case <-readDone:
			return
		case <-ticker.C:
			frame, captureErr := captureRemoteFrame(bounds)
			if captureErr != nil {
				log.Printf("[remote] screen capture failed: %v", captureErr)
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				return
			}
		}
	}
}

func captureRemoteFrame(bounds image.Rectangle) ([]byte, error) {
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}
	var frame bytes.Buffer
	if err := jpeg.Encode(&frame, img, &jpeg.Options{Quality: 55}); err != nil {
		return nil, err
	}
	return frame.Bytes(), nil
}
