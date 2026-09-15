package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type AgentConfig struct {
	ServerURL string `json:"server_url"`
	APIKey    string `json:"api_key"`
	DeviceID  string `json:"device_id"`
	Branch    string `json:"branch"`
	Heartbeat int    `json:"heartbeat_seconds"`
	UpdateURL string `json:"update_url"`
}

type Agent struct {
	cfg        AgentConfig
	cfgPath    string
	version    string
	conn       *websocket.Conn
	done       chan struct{}
	updatingMu sync.Mutex
	isUpdating bool
}

func NewAgent(cfg AgentConfig, version, cfgPath string) *Agent {
	if cfg.Heartbeat <= 0 {
		cfg.Heartbeat = 30
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = generateDeviceID()
	}
	if version == "" {
		version = "0.1.0"
	}
	return &Agent{
		cfg:     cfg,
		cfgPath: cfgPath,
		version: version,
		done:    make(chan struct{}),
	}
}

func (a *Agent) Run() {
	CleanupOldExecutable()

	// Run update check in background independent of server connection status
	go a.periodicUpdateCheck()

	for {
		if err := a.connect(); err != nil {
			log.Printf("[agent] connection failed: %v, retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		a.register()
		go a.heartbeatLoop()
		a.readLoop()

		log.Println("[agent] disconnected, reconnecting in 3s...")
		time.Sleep(3 * time.Second)
	}
}

func (a *Agent) connect() error {
	u, err := url.Parse(a.cfg.ServerURL)
	if err != nil {
		return err
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}

	wsURL := fmt.Sprintf("%s://%s/ws/agent?key=%s&id=%s",
		scheme, u.Host, a.cfg.APIKey, a.cfg.DeviceID)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	a.conn = conn
	log.Printf("[agent] connected to %s as %s (v%s)", u.Host, a.cfg.DeviceID, a.version)
	return nil
}

func (a *Agent) register() {
	info := CollectSystemInfo()
	info.Version = a.version
	info.Branch = a.cfg.Branch
	data, _ := json.Marshal(info)
	msg := map[string]interface{}{
		"action": "register",
		"data":   json.RawMessage(data),
	}
	raw, _ := json.Marshal(msg)
	a.conn.WriteMessage(websocket.TextMessage, raw)
	log.Printf("[agent] registered: %s (%s/%s) v%s mem=%s disk=%s",
		info.Hostname, info.OS, info.Arch, info.Version,
		FormatBytes(info.MemoryTotal), FormatBytes(info.DiskTotal))
}

func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(a.cfg.Heartbeat) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, memUsed := getMemoryInfo()
			_, diskUsed := getDiskInfo()
			hb := map[string]interface{}{
				"cpu_usage":   GetCPUUsage(),
				"memory_used": memUsed,
				"disk_used":   diskUsed,
			}
			data, _ := json.Marshal(hb)
			msg := map[string]interface{}{
				"action": "heartbeat",
				"data":   json.RawMessage(data),
			}
			raw, _ := json.Marshal(msg)
			if err := a.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				return
			}
		case <-a.done:
			return
		}
	}
}

func (a *Agent) periodicUpdateCheck() {
	if a.cfg.UpdateURL == "" {
		return
	}

	time.Sleep(30 * time.Second)
	a.checkGitHubRelease()

	ticker := time.NewTicker(2 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		a.checkGitHubRelease()
	}
}

func (a *Agent) checkGitHubRelease() {
	if a.cfg.UpdateURL == "" {
		return
	}

	apiURL := a.cfg.UpdateURL
	if strings.Contains(apiURL, "github.com") && !strings.Contains(apiURL, "api.github.com") {
		apiURL = strings.Replace(apiURL, "github.com", "api.github.com/repos", 1)
		if !strings.HasSuffix(apiURL, "/releases/latest") {
			apiURL = strings.TrimRight(apiURL, "/") + "/releases/latest"
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		log.Printf("[agent] update check failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	releaseVersion := strings.TrimPrefix(release.TagName, "v")
	if releaseVersion == "" || releaseVersion == a.version {
		return
	}

	suffix := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}

	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(strings.ToLower(asset.Name), strings.ToLower(suffix)) {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		log.Printf("[agent] new version %s found but no matching binary for %s/%s", releaseVersion, runtime.GOOS, runtime.GOARCH)
		return
	}

	log.Printf("[agent] updating from GitHub: v%s -> v%s", a.version, releaseVersion)
	go a.PerformUpdate(downloadURL)
}

func (a *Agent) readLoop() {
	defer func() {
		select {
		case <-a.done:
		default:
			close(a.done)
		}
		a.done = make(chan struct{})
	}()

	for {
		_, raw, err := a.conn.ReadMessage()
		if err != nil {
			log.Printf("[agent] read error: %v", err)
			return
		}
		a.handleMessage(raw)
	}
}

func (a *Agent) handleMessage(raw []byte) {
	var msg struct {
		Action string          `json:"action"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Action {
	case "upgrade":
		var req struct {
			Version string `json:"version"`
			URL     string `json:"download_url"`
		}
		if err := json.Unmarshal(msg.Data, &req); err == nil && req.URL != "" {
			if req.Version != "" && req.Version == a.version {
				return
			}
			log.Printf("[agent] upgrade command received: %s -> %s", a.version, req.Version)
			go a.PerformUpdate(req.URL)
		}

	case "reconfigure":
		var req struct {
			ServerURL string `json:"server_url"`
			APIKey    string `json:"api_key"`
		}
		if err := json.Unmarshal(msg.Data, &req); err == nil {
			changed := false
			if req.ServerURL != "" && req.ServerURL != a.cfg.ServerURL {
				log.Printf("[agent] server endpoint changed: %s -> %s", a.cfg.ServerURL, req.ServerURL)
				a.cfg.ServerURL = req.ServerURL
				changed = true
			}
			if req.APIKey != "" && req.APIKey != a.cfg.APIKey {
				a.cfg.APIKey = req.APIKey
				changed = true
			}
			if changed {
				a.saveConfig()
				log.Println("[agent] config saved. Reconnecting to new endpoint...")
				if a.conn != nil {
					a.conn.Close()
				}
			}
		}

	case "signal":
		log.Printf("[agent] received signal message")
	case "network_scan":
		var req struct {
			ScanID string `json:"scan_id"`
		}
		if err := json.Unmarshal(msg.Data, &req); err == nil && req.ScanID != "" {
			go func() {
				subnet, hosts, scanErr := ScanLocalNetwork()
				result := map[string]interface{}{"scan_id": req.ScanID, "subnet": subnet, "hosts": hosts}
				if scanErr != nil {
					result["error"] = scanErr.Error()
				}
				data, _ := json.Marshal(result)
				raw, _ := json.Marshal(map[string]interface{}{"action": "network_scan_result", "data": json.RawMessage(data)})
				if err := a.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
					log.Printf("[agent] network scan result failed: %v", err)
				}
			}()
		}
	case "command":
		log.Printf("[agent] received command: %s", string(msg.Data))
	}
}

func (a *Agent) saveConfig() {
	if a.cfgPath == "" {
		candidates := []string{"agent.json"}
		if exePath, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exePath), "agent.json"))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				a.cfgPath = c
				break
			}
		}
		if a.cfgPath == "" {
			if exePath, err := os.Executable(); err == nil {
				a.cfgPath = filepath.Join(filepath.Dir(exePath), "agent.json")
			} else {
				a.cfgPath = "agent.json"
			}
		}
	}

	data, err := json.MarshalIndent(a.cfg, "", "  ")
	if err != nil {
		log.Printf("[agent] failed to marshal config: %v", err)
		return
	}
	if err := os.WriteFile(a.cfgPath, data, 0644); err != nil {
		log.Printf("[agent] failed to save config to %s: %v", a.cfgPath, err)
	} else {
		log.Printf("[agent] config saved to %s", a.cfgPath)
	}
}

func (a *Agent) PerformUpdate(rawURL string) {
	a.updatingMu.Lock()
	if a.isUpdating {
		a.updatingMu.Unlock()
		return
	}
	a.isUpdating = true
	a.updatingMu.Unlock()

	defer func() {
		a.updatingMu.Lock()
		a.isUpdating = false
		a.updatingMu.Unlock()
	}()

	targetURL := rawURL
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		base := strings.TrimRight(a.cfg.ServerURL, "/")
		if !strings.HasPrefix(targetURL, "/") {
			targetURL = "/" + targetURL
		}
		targetURL = base + targetURL
	}

	if !strings.Contains(targetURL, "key=") && a.cfg.APIKey != "" && !strings.Contains(targetURL, "github.com") {
		sep := "?"
		if strings.Contains(targetURL, "?") {
			sep = "&"
		}
		targetURL = fmt.Sprintf("%s%skey=%s", targetURL, sep, url.QueryEscape(a.cfg.APIKey))
	}
	if !strings.Contains(targetURL, "github.com") && !strings.Contains(targetURL, "os=") {
		targetURL = fmt.Sprintf("%s&os=%s&arch=%s", targetURL, runtime.GOOS, runtime.GOARCH)
	}

	log.Printf("[agent] downloading update from: %s", targetURL)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		log.Printf("[agent] update request error: %v", err)
		return
	}
	req.Header.Set("User-Agent", fmt.Sprintf("RemoteDesk-Agent/%s (%s/%s)", a.version, runtime.GOOS, runtime.GOARCH))

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[agent] update download failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[agent] update download rejected: HTTP %d", resp.StatusCode)
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		log.Printf("[agent] could not get executable path: %v", err)
		return
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		log.Printf("[agent] could not resolve executable path: %v", err)
		return
	}

	dir := filepath.Dir(exePath)
	base := filepath.Base(exePath)
	newPath := filepath.Join(dir, base+".new")
	oldPath := filepath.Join(dir, base+".old")

	outFile, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		log.Printf("[agent] failed to create new binary file: %v", err)
		return
	}

	n, err := io.Copy(outFile, resp.Body)
	outFile.Close()
	if err != nil || n < 100000 {
		os.Remove(newPath)
		log.Printf("[agent] downloaded file incomplete or corrupt (bytes=%d, err=%v)", n, err)
		return
	}

	_ = os.Remove(oldPath)
	if err := os.Rename(exePath, oldPath); err != nil {
		os.Remove(newPath)
		log.Printf("[agent] failed to rename current executable: %v", err)
		return
	}
	if err := os.Rename(newPath, exePath); err != nil {
		_ = os.Rename(oldPath, exePath)
		os.Remove(newPath)
		log.Printf("[agent] failed to move new binary in place: %v", err)
		return
	}

	log.Printf("[agent] update applied successfully (%d bytes). Spawning new agent process...", n)

	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Printf("[agent] failed to restart agent process: %v", err)
		_ = os.Rename(oldPath, exePath)
		return
	}

	log.Printf("[agent] new process PID %d started. Exiting old process.", cmd.Process.Pid)
	os.Exit(0)
}

func CleanupOldExecutable() {
	if exePath, err := os.Executable(); err == nil {
		if evalPath, err := filepath.EvalSymlinks(exePath); err == nil {
			oldPath := filepath.Join(filepath.Dir(evalPath), filepath.Base(evalPath)+".old")
			_ = os.Remove(oldPath)
		}
	}
}

func generateDeviceID() string {
	hostname, _ := os.Hostname()
	idFile := filepath.Join(os.TempDir(), "rd-device-id")
	if data, err := os.ReadFile(idFile); err == nil {
		return string(data)
	}

	id := fmt.Sprintf("%s-%d", hostname, time.Now().UnixNano()%100000)
	os.WriteFile(idFile, []byte(id), 0644)
	return id
}

func LoadConfig(path string) (AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AgentConfig{}, err
	}
	var cfg AgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AgentConfig{}, err
	}
	return cfg, nil
}
