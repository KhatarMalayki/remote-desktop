package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type relaySession struct {
	mu       sync.Mutex
	once     sync.Once
	deviceID string
	agent    *websocket.Conn
	viewer   *websocket.Conn
	started  bool
}

var (
	relayMu       sync.Mutex
	relaySessions = map[string]*relaySession{}
)

func createViewerRelay(sessionID, deviceID string, conn *websocket.Conn) error {
	relayMu.Lock()
	if _, exists := relaySessions[sessionID]; exists {
		relayMu.Unlock()
		return fmt.Errorf("relay session already exists")
	}
	sess := &relaySession{deviceID: deviceID, viewer: conn}
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
	viewer := sess.viewer
	sess.mu.Unlock()

	go pipeRelay(sessionID, sess, conn, viewer)
	go pipeRelay(sessionID, sess, viewer, conn)
	return nil
}

func pipeRelay(sessionID string, sess *relaySession, src, dst *websocket.Conn) {
	for {
		msgType, data, err := src.ReadMessage()
		if err != nil {
			log.Printf("[relay] %s read ended: %v", sessionID, err)
			closeRelaySession(sessionID, sess)
			return
		}
		if err := dst.WriteMessage(msgType, data); err != nil {
			log.Printf("[relay] %s write ended: %v", sessionID, err)
			closeRelaySession(sessionID, sess)
			return
		}
	}
}

func closeRelaySession(sessionID string, sess *relaySession) {
	sess.once.Do(func() {
		relayMu.Lock()
		if relaySessions[sessionID] == sess {
			delete(relaySessions, sessionID)
		}
		relayMu.Unlock()
		sess.mu.Lock()
		if sess.viewer != nil {
			_ = sess.viewer.Close()
		}
		if sess.agent != nil {
			_ = sess.agent.Close()
		}
		sess.mu.Unlock()
	})
}
