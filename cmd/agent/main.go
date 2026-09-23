package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/user/remote-desktop/internal/agent"
)

var version = "0.2.44"

func main() {
	serverURL := flag.String("server", envOr("RD_SERVER_URL", ""), "server URL")
	apiKey := flag.String("key", envOr("RD_API_KEY", ""), "API key")
	deviceID := flag.String("id", envOr("RD_DEVICE_ID", ""), "device ID (auto-generated if empty)")
	branch := flag.String("branch", envOr("RD_BRANCH", ""), "branch name (e.g. Medan, Surabaya)")
	heartbeat := flag.Int("heartbeat", 30, "heartbeat interval in seconds")
	configFile := flag.String("config", "", "config file path (JSON)")
	logFile := flag.String("log-file", "", "append agent runtime log to this file")
	systemService := flag.Bool("system-service", false, "run as the RemoteDesk Windows system service")
	systemWorker := flag.Bool("system-worker", false, "run as a SYSTEM helper in the active console session")
	secureRelay := flag.String("secure-relay", "", "run one remote relay directly on the Windows Winlogon desktop")
	flag.Parse()
	if *logFile != "" {
		if file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
			log.SetOutput(file)
		} else {
			log.Printf("[agent] cannot open log file %s: %v", *logFile, err)
		}
	}

	// The service deliberately has no network/relay role of its own.  It only
	// creates a LocalSystem worker inside the currently active console session.
	// That is what keeps the capture/input process alive when Windows swaps the
	// user desktop for the lock / Winlogon desktop.
	if *systemService {
		runSystemService(*configFile)
		return
	}
	if *systemWorker && *secureRelay == "" {
		release, acquired, err := acquireSystemWorkerLock()
		if err != nil {
			log.Fatalf("[agent] cannot acquire SYSTEM worker lock: %v", err)
		}
		if !acquired {
			log.Printf("[agent] another SYSTEM console worker is already running; exiting duplicate")
			return
		}
		defer release()
	}

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
			Branch:    *branch,
			Heartbeat: *heartbeat,
		}
	}

	if cfg.Branch == "" && *branch != "" {
		cfg.Branch = *branch
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

	if *systemWorker {
		log.Printf("[agent] SYSTEM console worker v%s starting...", version)
	} else {
		log.Printf("[agent] version %s starting...", version)
	}
	a := agent.NewAgent(cfg, version, configPath)
	a.SetServiceManaged(*systemWorker)
	if *secureRelay != "" {
		log.Printf("[agent] secure Winlogon relay v%s starting", version)
		a.SetSecureDesktopOnly(true)
		a.RunRemoteRelay(*secureRelay)
		return
	}
	if *systemWorker {
		a.SetSecureRelayStarter(func(sessionID string) error {
			return startSecureDesktopRelay(configPath, sessionID)
		})
	}
	a.Run()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
