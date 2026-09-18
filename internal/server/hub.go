package server

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/remote-desktop/internal/models"
)

type Client struct {
	DeviceID string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
	IsAgent  bool
}

type Hub struct {
	mu      sync.RWMutex
	agents  map[string]*Client
	viewers map[*Client]bool
	db      *DB
}

func NewHub(db *DB) *Hub {
	return &Hub{
		agents:  make(map[string]*Client),
		viewers: make(map[*Client]bool),
		db:      db,
	}
}

func (h *Hub) RegisterAgent(c *Client) {
	h.mu.Lock()
	previous := h.agents[c.DeviceID]
	h.agents[c.DeviceID] = c
	h.mu.Unlock()
	// An upgrade/reconnect can briefly overlap the old socket. Close the old
	// connection so its later cleanup cannot leave a ghost agent behind.
	if previous != nil && previous != c {
		_ = previous.Conn.Close()
	}
	log.Printf("[hub] agent registered: %s", c.DeviceID)
	h.broadcastStatus()
}

func (h *Hub) UnregisterAgent(c *Client) {
	h.mu.Lock()
	if existing, ok := h.agents[c.DeviceID]; ok && existing == c {
		delete(h.agents, c.DeviceID)
	}
	h.mu.Unlock()
	log.Printf("[hub] agent disconnected: %s", c.DeviceID)
	h.broadcastStatus()
}

func (h *Hub) RegisterViewer(c *Client) {
	h.mu.Lock()
	h.viewers[c] = true
	h.mu.Unlock()
}

func (h *Hub) UnregisterViewer(c *Client) {
	h.mu.Lock()
	delete(h.viewers, c)
	h.mu.Unlock()
}

func (h *Hub) IsOnline(deviceID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[deviceID]
	return ok
}

func (h *Hub) SendToAgent(deviceID string, data []byte) bool {
	h.mu.RLock()
	agent, ok := h.agents[deviceID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case agent.Send <- data:
		return true
	default:
		return false
	}
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.agents)
}

func (h *Hub) OnlineIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

func (h *Hub) ForwardSignal(msg *models.SignalMessage) {
	h.mu.RLock()
	target, ok := h.agents[msg.To]
	h.mu.RUnlock()

	if !ok {
		h.mu.RLock()
		for v := range h.viewers {
			if v.DeviceID == msg.To {
				target = v
				ok = true
				break
			}
		}
		h.mu.RUnlock()
	}

	if ok {
		data, _ := json.Marshal(map[string]interface{}{
			"action": "signal",
			"data":   msg,
		})
		select {
		case target.Send <- data:
		default:
		}
	}
}

func (h *Hub) broadcastStatus() {
	h.mu.RLock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	viewers := make([]*Client, 0, len(h.viewers))
	for v := range h.viewers {
		viewers = append(viewers, v)
	}
	h.mu.RUnlock()

	data, _ := json.Marshal(map[string]interface{}{
		"action": "online_status",
		"data": map[string]interface{}{
			"online_ids": ids,
			"count":      len(ids),
		},
	})

	for _, v := range viewers {
		select {
		case v.Send <- data:
		default:
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(handler func(*Client, []byte)) {
	defer func() {
		if c.IsAgent {
			c.Hub.UnregisterAgent(c)
		} else {
			c.Hub.UnregisterViewer(c)
		}
		close(c.Send)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(512 * 1024)
	c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		handler(c, msg)
	}
}
