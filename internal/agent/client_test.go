package agent

import (
	"os"
	"path/filepath"
	"testing"
)

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
