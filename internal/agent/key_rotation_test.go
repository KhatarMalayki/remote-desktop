package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/websocket"
)

func TestAgentConnectEncodesSpecialKeyAndDeviceID(t *testing.T) {
	key := "key#&+%?= /中文-123456"
	id := "pc #&+%?"
	checked := make(chan bool, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		checked <- r.URL.Query().Get("key") == key && r.URL.Query().Get("id") == id
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()
	agent := &Agent{cfg: AgentConfig{ServerURL: server.URL, APIKey: key, DeviceID: id}}
	if err := agent.connect(); err != nil {
		t.Fatal(err)
	}
	defer agent.conn.Close()
	if !<-checked {
		t.Fatal("key or device ID changed during URL encoding")
	}
}

func TestAgentKeyRecoveryAfterRejectedConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "working-old-key" {
			http.Error(w, "unauthorized", 401)
			return
		}
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			conn.Close()
		}
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "agent.json")
	agent := &Agent{cfg: AgentConfig{ServerURL: server.URL, DeviceID: "pilot", APIKey: "rejected-key", PreviousAPIKey: "working-old-key"}, cfgPath: path}
	if err := agent.saveConfig(); err != nil {
		t.Fatal(err)
	}
	if err := agent.connectWithKeyRecovery(); err == nil {
		t.Fatal("rejected key connected")
	}
	stored, err := LoadConfig(path)
	if err != nil || stored.APIKey != "working-old-key" || stored.PreviousAPIKey != "" {
		t.Fatal("rollback was not persisted")
	}
	if err := agent.connectWithKeyRecovery(); err != nil {
		t.Fatal(err)
	}
	agent.conn.Close()
}

func TestAgentReconfigureKeepsRecoveryKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.json")
	agent := &Agent{cfg: AgentConfig{APIKey: "old-key"}, cfgPath: path}
	if err := agent.saveConfig(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"api_key": "new-#&+key"})
	envelope, _ := json.Marshal(map[string]interface{}{"action": "reconfigure", "data": json.RawMessage(raw)})
	agent.handleMessage(envelope)
	stored, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if stored.APIKey != "new-#&+key" || stored.PreviousAPIKey != "old-key" {
		t.Fatal("new key or rollback key not persisted")
	}
	agent.cfgPath = filepath.Join(t.TempDir(), "missing", "agent.json")
	previous := agent.cfg
	envelope, _ = json.Marshal(map[string]interface{}{"action": "reconfigure", "data": map[string]string{"api_key": "another-key"}})
	agent.handleMessage(envelope)
	if agent.cfg != previous {
		t.Fatal("failed save changed in-memory credentials")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("previous config lost")
	}
}
