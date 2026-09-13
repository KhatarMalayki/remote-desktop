package server

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type relaySession struct {
	mu     sync.Mutex
	agent  *websocket.Conn
	viewer *websocket.Conn
}

var (
	relayMu       sync.Mutex
	relaySessions = map[string]*relaySession{}
)

func (h *Hub) HandleRelay(sessionID, role string, conn *websocket.Conn) {
	relayMu.Lock()
	sess, ok := relaySessions[sessionID]
	if !ok {
		sess = &relaySession{}
		relaySessions[sessionID] = sess
	}
	relayMu.Unlock()

	sess.mu.Lock()
	switch role {
	case "agent":
		sess.agent = conn
	case "viewer":
		sess.viewer = conn
	}
	agent := sess.agent
	viewer := sess.viewer
	sess.mu.Unlock()

	if agent != nil && viewer != nil {
		go pipeRelay(sessionID, agent, viewer)
		go pipeRelay(sessionID, viewer, agent)
	} else {
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		for {
			sess.mu.Lock()
			agent = sess.agent
			viewer = sess.viewer
			sess.mu.Unlock()
			if agent != nil && viewer != nil {
				go pipeRelay(sessionID, agent, viewer)
				go pipeRelay(sessionID, viewer, agent)
				return
			}
			time.Sleep(200 * time.Millisecond)
			if time.Now().After(time.Now().Add(-30 * time.Second)) {
				break
			}
		}
	}
}

func pipeRelay(sessionID string, src, dst *websocket.Conn) {
	defer func() {
		relayMu.Lock()
		delete(relaySessions, sessionID)
		relayMu.Unlock()
		src.Close()
		dst.Close()
	}()

	for {
		msgType, data, err := src.ReadMessage()
		if err != nil {
			log.Printf("[relay] %s read error: %v", sessionID, err)
			return
		}
		if err := dst.WriteMessage(msgType, data); err != nil {
			log.Printf("[relay] %s write error: %v", sessionID, err)
			return
		}
	}
}
