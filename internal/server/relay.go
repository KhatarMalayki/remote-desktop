package server

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type relaySession struct {
	mu        sync.Mutex
	once      sync.Once
	deviceID  string
	agent     *websocket.Conn
	viewer    *websocket.Conn
	started   bool
	startedAt time.Time
	audit     relayAudit
}

type relayAudit struct {
	onStart   func()
	onEnd     func(time.Duration)
	onControl func(string)
}

var (
	relayMu       sync.Mutex
	relaySessions = map[string]*relaySession{}
)

func createViewerRelay(sessionID, deviceID string, conn *websocket.Conn, audit relayAudit) error {
	relayMu.Lock()
	if _, exists := relaySessions[sessionID]; exists {
		relayMu.Unlock()
		return fmt.Errorf("relay session already exists")
	}
	sess := &relaySession{deviceID: deviceID, viewer: conn, audit: audit}
	relaySessions[sessionID] = sess
	relayMu.Unlock()

	go func() {
		time.Sleep(30 * time.Second)
		sess.mu.Lock()
		started := sess.started
		sess.mu.Unlock()
		if !started {
			closeRelaySession(sessionID, sess)
		}
	}()
	return nil
}

func attachAgentRelay(sessionID, deviceID string, conn *websocket.Conn) error {
	relayMu.Lock()
	sess := relaySessions[sessionID]
	relayMu.Unlock()
	if sess == nil {
		return fmt.Errorf("relay session not found or expired")
	}

	sess.mu.Lock()
	if sess.deviceID != deviceID {
		sess.mu.Unlock()
		return fmt.Errorf("relay session belongs to another device")
	}
	if sess.agent != nil || sess.started {
		sess.mu.Unlock()
		return fmt.Errorf("relay agent already connected")
	}
	sess.agent = conn
	sess.started = true
	sess.startedAt = time.Now()
	viewer := sess.viewer
	onStart := sess.audit.onStart
	sess.mu.Unlock()
	if onStart != nil {
		onStart()
	}

	go pipeRelay(sessionID, sess, conn, viewer)
	go pipeRelay(sessionID, sess, viewer, conn)
	return nil
}

func pipeRelay(sessionID string, sess *relaySession, src, dst *websocket.Conn) {
	done := make(chan struct{})
	defer close(done)
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := src.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					closeRelaySession(sessionID, sess, "relay_keepalive_failed")
					return
				}
			}
		}
	}()
	for {
		msgType, data, err := src.ReadMessage()
		if err != nil {
			log.Printf("[relay] %s read ended: %v", sessionID, err)
			reason := "agent_connection_closed"
			if src == sess.viewer {
				reason = "viewer_connection_closed"
			}
			closeRelaySession(sessionID, sess, reason)
			return
		}
		_ = dst.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := dst.WriteMessage(msgType, data); err != nil {
			log.Printf("[relay] %s write ended: %v", sessionID, err)
			closeRelaySession(sessionID, sess, "relay_write_failed")
			return
		}
		if src == sess.viewer && msgType == websocket.TextMessage && sess.audit.onControl != nil {
			var command struct {
				Type    string
				Enabled bool
			}
			if json.Unmarshal(data, &command) == nil {
				switch command.Type {
				case "block_input", "privacy_mode":
					sess.audit.onControl(fmt.Sprintf("%s requested enabled=%t", command.Type, command.Enabled))
				case "file_start", "file_end", "file_cancel":
					sess.audit.onControl(command.Type + " requested")
				}
			}
		}
	}
}

func closeRelaySession(sessionID string, sess *relaySession, reasons ...string) {
	sess.once.Do(func() {
		relayMu.Lock()
		if relaySessions[sessionID] == sess {
			delete(relaySessions, sessionID)
		}
		relayMu.Unlock()
		sess.mu.Lock()
		startedAt := sess.startedAt
		onEnd := sess.audit.onEnd
		if sess.viewer != nil {
			reason := "agent_connect_timeout"
			if len(reasons) > 0 {
				reason = reasons[0]
			}
			_ = sess.viewer.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, reason), time.Now().Add(time.Second))
			_ = sess.viewer.Close()
		}
		if sess.agent != nil {
			_ = sess.agent.Close()
		}
		sess.mu.Unlock()
		if !startedAt.IsZero() && onEnd != nil {
			onEnd(time.Since(startedAt).Round(time.Second))
		}
	})
}
