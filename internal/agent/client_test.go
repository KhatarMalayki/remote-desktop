package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAcceptsUTF8BOM(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "agent.json")
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"server_url":"https://example.test","api_key":"test","device_id":"device-1"}`)...)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig rejected UTF-8 BOM: %v", err)
	}
	if cfg.DeviceID != "device-1" {
		t.Fatalf("device ID = %q, want device-1", cfg.DeviceID)
	}
}

func TestNewAgentPersistsGeneratedDeviceID(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "agent.json")
	agent := NewAgent(AgentConfig{ServerURL: "https://example.test", APIKey: "test"}, "test", configPath)
	if agent.cfg.DeviceID == "" {
		t.Fatal("generated device ID is empty")
	}
	if _, err := os.ReadFile(configPath); err != nil {
		t.Fatalf("generated identity was not persisted: %v", err)
	}
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceID != agent.cfg.DeviceID {
		t.Fatalf("persisted ID = %q, want %q", loaded.DeviceID, agent.cfg.DeviceID)
	}
}
