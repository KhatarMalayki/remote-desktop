package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/user/remote-desktop/internal/agent"
)

const version = "0.1.0"

func main() {
	serverURL := flag.String("server", envOr("RD_SERVER_URL", ""), "server URL")
	apiKey := flag.String("key", envOr("RD_API_KEY", ""), "API key")
	deviceID := flag.String("id", envOr("RD_DEVICE_ID", ""), "device ID (auto-generated if empty)")
	heartbeat := flag.Int("heartbeat", 30, "heartbeat interval in seconds")
	configFile := flag.String("config", "", "config file path (JSON)")
	flag.Parse()

	var cfg agent.AgentConfig
	configPath := *configFile

	// If no config flag specified, look for agent.json in cwd or executable directory
	if configPath == "" {
		candidates := []string{"agent.json"}
		if exePath, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "agent.json"))
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && !info.IsDir() {
				configPath = c
				break
			}
		}
	}

	if configPath != "" {
		var err error
		cfg, err = agent.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("failed to load config from %s: %v", configPath, err)
		}
		log.Printf("[agent] loaded configuration from %s", configPath)
	} else {
		cfg = agent.AgentConfig{
			ServerURL: *serverURL,
			APIKey:    *apiKey,
			DeviceID:  *deviceID,
			Heartbeat: *heartbeat,
		}
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = envOr("RD_SERVER_URL", "http://localhost:8080")
	}

	if cfg.APIKey == "" {
		fmt.Println("===============================================================")
		fmt.Println(" [RemoteDesk Agent] API Key belum diset!")
		fmt.Println(" Agar bisa langsung klik 2 kali, siapkan file 'agent.json'")
		fmt.Println(" di folder yang sama dengan rd-agent.exe atau jalankan run-agent.bat")
		fmt.Println(" Contoh isi agent.json:")
		fmt.Println(`   { "server_url": "http://IP-SERVER:8080", "api_key": "YOUR_KEY" }`)
		fmt.Println("===============================================================")
		log.Fatal("API key is required (use -key, RD_API_KEY, or agent.json)")
	}

	log.Printf("[agent] version %s starting...", version)
	a := agent.NewAgent(cfg, version, configPath)
	a.Run()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
