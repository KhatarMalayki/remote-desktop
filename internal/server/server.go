package server

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/remote-desktop/internal/models"
	"github.com/user/remote-desktop/internal/versioncmp"
)

type Config struct {
	Addr           string
	DBPath         string
	APIKey         string
	AdminUser      string
	AdminPass      string
	JWTSecret      string
	Version        string
	AgentsDir      string
	RustDeskConfig string
}

type UserClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Branch   string `json:"branch"`
}

func canDeleteAssets(role string) bool  { return role == "admin" || role == "ga_pusat" }
func isCentralRole(role string) bool    { return role == "admin" || role == "ga_pusat" }
func canApproveSwitch(role string) bool { return role == "adh" || isCentralRole(role) }
func assetResponsibilityWarning() string {
	return "Asset ini tercatat sebagai tanggung jawab pemegangnya. Pastikan kondisi, kelengkapan, dan data serah terima sudah benar sebelum melakukan switch."
}

func deviceRecommendation(dev *models.Device) string {
	var notes []string
	if dev.MemoryTotal > 0 && dev.MemoryTotal < 8*1024*1024*1024 {
		notes = append(notes, "RAM di bawah 8GB; pertimbangkan upgrade untuk operasional aplikasi modern.")
	}
	if dev.CPUCores > 0 && dev.CPUCores < 4 {
		notes = append(notes, "CPU kurang dari 4 core; performa multitasking kemungkinan terbatas.")
	}
	if dev.DiskTotal > 0 && float64(dev.DiskUsed)/float64(dev.DiskTotal) >= 0.85 {
		notes = append(notes, "Disk terpakai 85% atau lebih; rawan lambat dan perlu pembersihan/upgrade.")
	}
	if dev.AcquisitionYear > 0 && time.Now().Year()-dev.AcquisitionYear >= 5 {
		notes = append(notes, "Usia aset 5 tahun atau lebih berdasarkan tahun pengadaan; evaluasi kondisi, dukungan OS, dan kebutuhan penggantian.")
	}
	if len(notes) == 0 {
		return "Belum ada indikasi upgrade dari data yang tersedia. Kelayakan tetap perlu disesuaikan dengan aplikasi kerja."
	}
	return strings.Join(notes, " ")
}

func manualAssetRecommendation(a *models.ManualAsset) string {
	if a.Category != "pc" && a.Category != "laptop" {
		return ""
	}
	specs := strings.ToLower(a.Specs)
	var notes []string
	if !strings.Contains(specs, "ssd") {
		notes = append(notes, "Belum terdeteksi SSD di spesifikasi; untuk operasional sekarang SSD sangat disarankan.")
	}
	compact := " " + strings.Join(strings.Fields(strings.ReplaceAll(specs, " gb", "gb")), " ") + " "
	if strings.Contains(compact, " 4gb ") || strings.Contains(compact, " 2gb ") {
		notes = append(notes, "RAM tampak di bawah standar 8GB; pertimbangkan upgrade.")
	}
	if a.AcquisitionYear > 0 && time.Now().Year()-a.AcquisitionYear >= 5 {
		notes = append(notes, "Usia aset 5 tahun atau lebih berdasarkan tahun pengadaan; evaluasi kondisi, dukungan OS, dan kebutuhan penggantian.")
	}
	return strings.Join(notes, " ")
}

type contextKey string

const userClaimsKey contextKey = "userClaims"

type loginAttempt struct {
	count     int
	firstFail time.Time
}

var (
	loginMu       sync.Mutex
	loginAttempts = make(map[string]*loginAttempt)
)

func (s *Server) checkLoginRateLimit(ip string) bool {
	settings := s.db.GetSecuritySettings()
	if !settings.RateLimitEnabled {
		return true
	}

	// Check whitelist
	for _, w := range strings.Split(settings.IPWhitelist, ",") {
		wClean := strings.TrimSpace(w)
		if wClean != "" && (wClean == ip || strings.HasPrefix(ip, wClean)) {
			return true
		}
	}

	loginMu.Lock()
	defer loginMu.Unlock()
	att, exists := loginAttempts[ip]
	if !exists {
		return true
	}
	blockDuration := time.Duration(settings.BlockDurationMin) * time.Minute
	if time.Since(att.firstFail) > blockDuration {
		delete(loginAttempts, ip)
		return true
	}
	return att.count < settings.MaxLoginAttempts
}

func (s *Server) getBlockedIPs() []models.BlockedIPInfo {
	settings := s.db.GetSecuritySettings()
	blockDuration := time.Duration(settings.BlockDurationMin) * time.Minute

	loginMu.Lock()
	defer loginMu.Unlock()

	var list []models.BlockedIPInfo
	now := time.Now()
	for ip, att := range loginAttempts {
		if att.count >= settings.MaxLoginAttempts {
			elapsed := now.Sub(att.firstFail)
			if elapsed <= blockDuration {
				expiresAt := att.firstFail.Add(blockDuration)
				minsLeft := int(expiresAt.Sub(now).Minutes()) + 1
				list = append(list, models.BlockedIPInfo{
					IP:          ip,
					FailedCount: att.count,
					BlockedAt:   att.firstFail,
					ExpiresAt:   expiresAt,
					MinutesLeft: minsLeft,
				})
			}
		}
	}
	return list
}

func (s *Server) unblockIP(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, ip)
}

func recordLoginFail(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	att, exists := loginAttempts[ip]
	if !exists || time.Since(att.firstFail) > 15*time.Minute {
		loginAttempts[ip] = &loginAttempt{count: 1, firstFail: time.Now()}
		return
	}
	att.count++
}

func recordLoginSuccess(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, ip)
}

type Server struct {
	cfg      Config
	db       *DB
	hub      *Hub
	upgrader websocket.Upgrader
	webFS    fs.FS
	scanMu   sync.RWMutex
	scans    map[string]*models.NetworkScan
}

func New(cfg Config, webFS embed.FS) (*Server, error) {
	db, err := NewDB(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("db init: %w", err)
	}

	if cfg.APIKey == "" {
		b := make([]byte, 16)
		rand.Read(b)
		cfg.APIKey = hex.EncodeToString(b)
		log.Printf("[server] generated API key: %s", cfg.APIKey)
	}

	if cfg.JWTSecret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		cfg.JWTSecret = hex.EncodeToString(b)
	}

	passHash := hashPassword(cfg.AdminPass)
	db.EnsureAdmin(cfg.AdminUser, passHash)

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfg:   cfg,
		db:    db,
		hub:   NewHub(db),
		webFS: sub,
		scans: make(map[string]*models.NetworkScan),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
	return s, nil
}

func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/auth/login", s.handleLogin)
	mux.HandleFunc("/api/auth/login/mfa", s.handleLoginMFA)
	mux.HandleFunc("/api/auth/me", s.authMiddleware(s.handleMe))
	mux.HandleFunc("/api/auth/change-password", s.authMiddleware(s.handleChangePassword))
	mux.HandleFunc("/api/auth/change-username", s.authMiddleware(s.handleChangeUsername))
	mux.HandleFunc("/api/auth/mfa/status", s.authMiddleware(s.handleMFAStatus))
	mux.HandleFunc("/api/auth/mfa/setup", s.authMiddleware(s.handleMFASetup))
	mux.HandleFunc("/api/auth/mfa/enable", s.authMiddleware(s.handleMFAEnable))
	mux.HandleFunc("/api/auth/mfa/disable", s.authMiddleware(s.handleMFADisable))
	mux.HandleFunc("/api/auth/logs", s.authMiddleware(s.handleAuthLogs))
	mux.HandleFunc("/api/security/settings", s.authMiddleware(s.handleSecuritySettings))
	mux.HandleFunc("/api/security/unblock", s.authMiddleware(s.handleUnblockIP))
	mux.HandleFunc("/api/devices", s.authMiddleware(s.handleDevices))
	mux.HandleFunc("/api/devices/", s.authMiddleware(s.handleDevice))
	mux.HandleFunc("/api/stats", s.authMiddleware(s.handleStats))
	mux.HandleFunc("/api/groups", s.authMiddleware(s.handleGroups))
	mux.HandleFunc("/api/branches", s.authMiddleware(s.handleBranches))
	mux.HandleFunc("/api/branches/", s.authMiddleware(s.handleBranchSubroute))
	mux.HandleFunc("/api/location-types/rename", s.authMiddleware(s.handleLocationTypeRename))
	mux.HandleFunc("/api/business-units", s.authMiddleware(s.handleBusinessUnits))
	mux.HandleFunc("/api/branches/stats", s.authMiddleware(s.handleBranchStats))
	mux.HandleFunc("/api/network-scans", s.authMiddleware(s.handleNetworkScans))
	mux.HandleFunc("/api/network-scans/", s.authMiddleware(s.handleNetworkScan))
	mux.HandleFunc("/api/assets/manual", s.authMiddleware(s.handleManualAssets))
	mux.HandleFunc("/api/assets/manual/", s.authMiddleware(s.handleManualAsset))
	mux.HandleFunc("/api/assets/verify", s.authMiddleware(s.handleVerifyAsset))
	mux.HandleFunc("/api/assets/verifications", s.authMiddleware(s.handleAssetVerifications))
	mux.HandleFunc("/api/assets/switch-requests", s.authMiddleware(s.handleSwitchRequests))
	mux.HandleFunc("/api/assets/switch-requests/", s.authMiddleware(s.handleSwitchRequestSubroute))
	mux.HandleFunc("/api/agent/version", s.handleAgentVersion)
	mux.HandleFunc("/api/agent/download", s.handleAgentDownload)
	mux.HandleFunc("/api/agent/package", s.authMiddleware(s.handleAgentPackageDownload))
	mux.HandleFunc("/api/agent/update", s.authMiddleware(s.handleAgentUpdate))
	mux.HandleFunc("/api/agent/broadcast-update", s.authMiddleware(s.handleBroadcastAgentUpdate))
	mux.HandleFunc("/api/agent/reconfigure", s.authMiddleware(s.handleReconfigureAgents))
	mux.HandleFunc("/api/rustdesk/settings", s.authMiddleware(s.handleRustDeskSettings))
	mux.HandleFunc("/api/rustdesk/manage", s.authMiddleware(s.handleRustDeskManage))
	mux.HandleFunc("/api/users", s.authMiddleware(s.handleUsers))
	mux.HandleFunc("/api/users/", s.authMiddleware(s.handleUserSubroute))
	mux.HandleFunc("/api/logs/", s.authMiddleware(s.handleLogs))

	mux.HandleFunc("/ws/agent", s.handleAgentWS)
	mux.HandleFunc("/ws/viewer", s.authMiddleware(s.handleViewerWS))
	mux.HandleFunc("/ws/relay/", s.handleRelayWS)

	mux.Handle("/", http.FileServer(http.FS(s.webFS)))

	log.Printf("[server] listening on %s", s.cfg.Addr)
	return http.ListenAndServe(s.cfg.Addr, mux)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	ip := r.RemoteAddr
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		ip = strings.TrimSpace(strings.Split(f, ",")[0])
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = strings.TrimSpace(realIP)
	}

	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		Code           string `json:"code"`
		RememberDevice bool   `json:"remember_device"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}

	if !s.checkLoginRateLimit(ip) {
		_ = s.db.RecordAuthLog(req.Username, ip, "blocked", "IP diblokir sementara (rate limit 15 menit)", r.UserAgent())
		log.Printf("[security] LOGIN BLOCKED ip=%s username=%s", ip, req.Username)
		jsonError(w, "Terlalu banyak percobaan login gagal. Diblokir sementara selama 15 menit.", 429)
		return
	}

	_, storedHash, role, branch, err := s.db.GetUser(req.Username)
	if err != nil || storedHash != hashPassword(req.Password) {
		recordLoginFail(ip)
		_ = s.db.RecordAuthLog(req.Username, ip, "failed", "Password salah atau username tidak ditemukan", r.UserAgent())
		log.Printf("[security] LOGIN FAILED ip=%s username=%s", ip, req.Username)
		jsonError(w, "invalid credentials", 401)
		return
	}

	mfaEnabled, mfaSecret, _ := s.db.GetUserMFA(req.Username)
	trusted := false
	if c, err := r.Cookie("rd_trusted_device"); err == nil {
		trusted = s.db.IsTrustedDevice(req.Username, hashPassword(c.Value))
	}
	if mfaEnabled && !trusted {
		if req.Code == "" {
			ticket := generateMFATicket(req.Username, role, branch, s.cfg.JWTSecret)
			jsonResp(w, map[string]interface{}{
				"mfa_required": true,
				"mfa_ticket":   ticket,
				"username":     req.Username,
			}, 200)
			return
		}
		if !ValidateTOTPCode(mfaSecret, req.Code) {
			recordLoginFail(ip)
			_ = s.db.RecordAuthLog(req.Username, ip, "mfa_failed", "Kode 2FA salah atau kedaluwarsa", r.UserAgent())
			log.Printf("[security] MFA LOGIN FAILED ip=%s username=%s", ip, req.Username)
			jsonError(w, "Kode 2FA / Authenticator salah atau kedaluwarsa", 401)
			return
		}
	}

	recordLoginSuccess(ip)
	_ = s.db.RecordAuthLog(req.Username, ip, "success", "Login berhasil", r.UserAgent())
	log.Printf("[security] LOGIN SUCCESS ip=%s username=%s role=%s", ip, req.Username, role)
	token := generateToken(req.Username, role, branch, s.cfg.JWTSecret)
	if req.RememberDevice && mfaEnabled {
		s.setTrustedDeviceCookie(w, req.Username)
	}
	jsonResp(w, map[string]string{
		"token":    token,
		"username": req.Username,
		"role":     role,
		"branch":   branch,
	}, 200)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	jsonResp(w, claims, 200)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	switch r.Method {
	case http.MethodGet:
		group := r.URL.Query().Get("group")
		if claims.Role == "adh" && claims.Branch != "" {
			group = claims.Branch
		}
		search := r.URL.Query().Get("search")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		var devices []*models.Device
		var total int
		var err error
		if claims.Role == "user" {
			devices, total, err = s.db.ListDevicesForOwner(claims.Username, search, limit, offset)
		} else {
			devices, total, err = s.db.ListDevices(group, search, limit, offset)
		}
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}

		onlineIDs := s.hub.OnlineIDs()
		onlineSet := map[string]bool{}
		for _, id := range onlineIDs {
			onlineSet[id] = true
		}
		for _, d := range devices {
			d.Online = onlineSet[d.ID]
			d.Recommendation = deviceRecommendation(d)
		}

		jsonResp(w, map[string]interface{}{
			"devices":        devices,
			"total":          total,
			"limit":          limit,
			"offset":         offset,
			"server_version": s.cfg.Version,
		}, 200)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleDevice(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/")
	if id == "" {
		jsonError(w, "missing device id", 400)
		return
	}

	claims := getClaims(r)
	dev, err := s.db.GetDevice(id)
	if err != nil {
		jsonError(w, "device not found", 404)
		return
	}

	// Strict branch isolation for ADH
	if claims.Role == "adh" && claims.Branch != "" {
		devBranch := dev.Branch
		if devBranch == "" {
			devBranch = dev.GroupName
		}
		if devBranch != claims.Branch {
			jsonError(w, "forbidden: perangkat milik cabang lain", 403)
			return
		}
	}
	if claims.Role == "user" && dev.OwnerUsername != claims.Username {
		jsonError(w, "forbidden: asset bukan milik user ini", 403)
		return
	}

	switch r.Method {
	case http.MethodGet:
		dev.Online = s.hub.IsOnline(id)
		dev.Recommendation = deviceRecommendation(dev)
		jsonResp(w, dev, 200)

	case http.MethodPut:
		if claims.Role == "viewer" {
			jsonError(w, "forbidden", 403)
			return
		}
		var req struct {
			Tags            string `json:"tags"`
			Group           string `json:"group"`
			Note            string `json:"note"`
			OwnerUsername   string `json:"owner_username"`
			AcquisitionYear *int   `json:"acquisition_year"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid body", 400)
			return
		}
		if claims.Role == "adh" && claims.Branch != "" {
			req.Group, req.Tags = dev.GroupName, dev.Tags
		}
		if claims.Role == "user" {
			req.Group, req.Tags, req.OwnerUsername = dev.GroupName, dev.Tags, dev.OwnerUsername
		}
		// Ownership changes must go through the audited switch workflow.
		req.OwnerUsername = dev.OwnerUsername
		if req.AcquisitionYear != nil && claims.Role != "user" {
			if *req.AcquisitionYear != 0 && (*req.AcquisitionYear < 1970 || *req.AcquisitionYear > time.Now().Year()) {
				jsonError(w, "Tahun pengadaan tidak valid", 400)
				return
			}
			if _, err := s.db.db.Exec(`UPDATE devices SET acquisition_year=? WHERE id=?`, *req.AcquisitionYear, id); err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
		}
		if err := s.db.UpdateDeviceMeta(id, req.Tags, req.Group, req.Note, strings.TrimSpace(req.OwnerUsername)); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		s.db.AddLog(id, "update_meta", fmt.Sprintf("tags=%s group=%s owner=%s", req.Tags, req.Group, req.OwnerUsername))
		jsonResp(w, map[string]string{"status": "ok"}, 200)

	case http.MethodDelete:
		if claims.Role == "viewer" || claims.Role == "user" {
			jsonError(w, "forbidden", 403)
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Reason) == "" {
			jsonError(w, "deletion reason is required", 400)
			return
		}
		branch := dev.Branch
		if branch == "" {
			branch = dev.GroupName
		}
		if err := s.db.RecordAssetDeletion(id, "device", dev.Hostname, branch, claims.Username, strings.TrimSpace(req.Reason)); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		if err := s.db.DeleteDevice(id); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "deleted"}, 200)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

// handleNetworkScans asks an online agent to discover hosts on its own /24 LAN.
// No caller-supplied subnet is accepted, preventing scans of arbitrary networks.
func (s *Server) handleNetworkScans(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims.Role == "viewer" || claims.Role == "user" {
		jsonError(w, "forbidden", 403)
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.DeviceID) == "" {
		jsonError(w, "device_id is required", 400)
		return
	}
	dev, err := s.db.GetDevice(req.DeviceID)
	if err != nil {
		jsonError(w, "device not found", 404)
		return
	}
	branch := dev.Branch
	if branch == "" {
		branch = dev.GroupName
	}
	if claims.Role == "adh" && claims.Branch != "" && branch != claims.Branch {
		jsonError(w, "forbidden: perangkat milik cabang lain", 403)
		return
	}
	if !s.hub.IsOnline(req.DeviceID) {
		jsonError(w, "agent must be online to scan its local network", 409)
		return
	}
	scan := &models.NetworkScan{ID: fmt.Sprintf("scan-%x", time.Now().UnixNano()), DeviceID: req.DeviceID, Status: "queued", StartedAt: time.Now()}
	s.scanMu.Lock()
	s.scans[scan.ID] = scan
	s.scanMu.Unlock()
	payload, _ := json.Marshal(map[string]interface{}{"action": "network_scan", "data": map[string]string{"scan_id": scan.ID}})
	if !s.hub.SendToAgent(req.DeviceID, payload) {
		s.scanMu.Lock()
		delete(s.scans, scan.ID)
		s.scanMu.Unlock()
		jsonError(w, "agent is no longer online", 409)
		return
	}
	s.db.AddLog(req.DeviceID, "network_scan_requested", fmt.Sprintf("scan=%s by=%s", scan.ID, claims.Username))
	jsonResp(w, scan, http.StatusAccepted)
}

func (s *Server) handleNetworkScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/network-scans/")
	s.scanMu.RLock()
	scan, ok := s.scans[id]
	s.scanMu.RUnlock()
	if !ok {
		jsonError(w, "scan not found", 404)
		return
	}
	claims := getClaims(r)
	dev, err := s.db.GetDevice(scan.DeviceID)
	if err != nil {
		jsonError(w, "device not found", 404)
		return
	}
	branch := dev.Branch
	if branch == "" {
		branch = dev.GroupName
	}
	if claims.Role == "viewer" || claims.Role == "user" || (claims.Role == "adh" && claims.Branch != "" && branch != claims.Branch) {
		jsonError(w, "forbidden", 403)
		return
	}
	jsonResp(w, scan, 200)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonResp(w, map[string]int{}, 200)
		return
	}
	stats, err := s.db.Stats()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	stats["online_devices"] = s.hub.OnlineCount()
	jsonResp(w, stats, 200)
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonResp(w, []string{}, 200)
		return
	}
	groups, err := s.db.GetGroups()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonResp(w, []string{}, 200)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if r.URL.Query().Get("detail") == "true" || r.URL.Query().Get("full") == "true" {
			branches, err := s.db.ListBranches()
			if err != nil {
				jsonError(w, err.Error(), 500)
				return
			}
			jsonResp(w, branches, 200)
			return
		}
		branches, err := s.db.GetBranches()
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, branches, 200)

	case http.MethodPost:
		claims := getClaims(r)
		if claims.Role != "admin" {
			jsonError(w, "only admin can manage branches", 403)
			return
		}
		var req struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			BusinessUnit  string   `json:"business_unit"`
			BusinessUnits []string `json:"business_units"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if req.Name == "" {
			jsonError(w, "Nama lokasi wajib diisi", 400)
			return
		}
		if req.Type == "" {
			req.Type = "cabang"
		}
		if err := s.db.CreateBranch(req.Name, req.Type, req.BusinessUnit); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if err := s.db.SetBranchBusinessUnitsByName(req.Name, req.BusinessUnits); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "created"}, 201)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleBusinessUnits(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role != "admin" {
		jsonError(w, "only admin can manage business units", 403)
		return
	}
	if r.Method == http.MethodGet {
		units, err := s.db.ListBusinessUnits()
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, units, 200)
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		if err := s.db.AddBusinessUnit(req.Name); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		jsonResp(w, map[string]string{"status": "created"}, 201)
		return
	}
	http.Error(w, "method not allowed", 405)
}

func (s *Server) handleLocationTypeRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if getClaims(r).Role != "admin" {
		jsonError(w, "only admin can manage location types", 403)
		return
	}
	var req struct {
		OldType string `json:"old_type"`
		NewType string `json:"new_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	req.OldType, req.NewType = strings.TrimSpace(req.OldType), strings.TrimSpace(req.NewType)
	if !validBranchType(req.NewType) || req.OldType == "" {
		jsonError(w, "valid old_type and new_type are required", 400)
		return
	}
	if _, err := s.db.db.Exec(`UPDATE branches SET type=? WHERE type=?`, req.NewType, req.OldType); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "updated"}, 200)
}

func (s *Server) handleBranchStats(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonResp(w, map[string]int{}, 200)
		return
	}
	claims := getClaims(r)
	branch := r.URL.Query().Get("branch")
	if claims.Role == "adh" && claims.Branch != "" {
		branch = claims.Branch
	}
	stats, err := s.db.GetBranchStats(branch)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, stats, 200)
}

// ---------------- MANUAL ASSETS ----------------

func (s *Server) handleManualAssets(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	switch r.Method {
	case http.MethodGet:
		branch := r.URL.Query().Get("branch")
		if claims.Role == "adh" && claims.Branch != "" {
			branch = claims.Branch
		}
		category := r.URL.Query().Get("category")
		verificationStatus := r.URL.Query().Get("verification_status")
		search := r.URL.Query().Get("search")

		var assets []models.ManualAsset
		var err error
		if claims.Role == "user" {
			assets, err = s.db.ListManualAssetsForOwner(claims.Username, category, verificationStatus, search)
		} else {
			assets, err = s.db.ListManualAssets(branch, category, verificationStatus, search)
		}
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		for i := range assets {
			assets[i].Recommendation = manualAssetRecommendation(&assets[i])
		}
		jsonResp(w, assets, 200)

	case http.MethodPost:
		if claims.Role == "viewer" || claims.Role == "user" {
			jsonError(w, "forbidden", 403)
			return
		}
		var asset models.ManualAsset
		if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		if asset.Name == "" || asset.AssetTag == "" {
			jsonError(w, "asset name and tag are required", 400)
			return
		}
		if claims.Role == "adh" && claims.Branch != "" {
			asset.Branch = claims.Branch
		}
		if asset.Branch == "" {
			asset.Branch = "Pusat"
		}
		asset.CreatedBy = claims.Username
		asset.OwnerUsername = "" // Assign a holder through the audited switch workflow.
		if asset.AcquisitionYear != 0 && (asset.AcquisitionYear < 1970 || asset.AcquisitionYear > time.Now().Year()) {
			jsonError(w, "Tahun pengadaan tidak valid", 400)
			return
		}

		if err := s.db.CreateManualAsset(&asset); err != nil {
			jsonError(w, fmt.Sprintf("failed to save asset: %v", err), 500)
			return
		}
		jsonResp(w, asset, 201)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleManualAsset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/assets/manual/")
	if id == "" {
		jsonError(w, "missing asset id", 400)
		return
	}
	claims := getClaims(r)

	existing, err := s.db.GetManualAsset(id)
	if err != nil {
		jsonError(w, "asset not found", 404)
		return
	}
	if claims.Role == "adh" && claims.Branch != "" && existing.Branch != claims.Branch {
		jsonError(w, "forbidden: asset belongs to different branch", 403)
		return
	}
	if claims.Role == "user" && existing.OwnerUsername != claims.Username {
		jsonError(w, "forbidden: asset bukan milik user ini", 403)
		return
	}

	switch r.Method {
	case http.MethodGet:
		existing.Recommendation = manualAssetRecommendation(existing)
		jsonResp(w, existing, 200)

	case http.MethodPut:
		if claims.Role == "viewer" {
			jsonError(w, "forbidden", 403)
			return
		}
		var upd models.ManualAsset
		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		upd.ID = id
		if claims.Role == "user" {
			upd.AssetTag = existing.AssetTag
			upd.Name = existing.Name
			upd.Category = existing.Category
			upd.Branch = existing.Branch
			upd.OwnerUsername = existing.OwnerUsername
			upd.CreatedBy = existing.CreatedBy
			if upd.AssignedTo == "" {
				upd.AssignedTo = existing.AssignedTo
			}
			if upd.Status == "" {
				upd.Status = existing.Status
			}
		} else if claims.Role == "adh" && claims.Branch != "" {
			upd.Branch = claims.Branch
		} else if upd.Branch == "" {
			upd.Branch = existing.Branch
		}
		if upd.AssetTag == "" {
			upd.AssetTag = existing.AssetTag
		}
		if upd.Name == "" {
			upd.Name = existing.Name
		}
		upd.OwnerUsername = existing.OwnerUsername
		if claims.Role == "user" {
			upd.AssignedTo = existing.AssignedTo
			upd.AcquisitionYear = existing.AcquisitionYear
		}
		if upd.AcquisitionYear != 0 && (upd.AcquisitionYear < 1970 || upd.AcquisitionYear > time.Now().Year()) {
			jsonError(w, "Tahun pengadaan tidak valid", 400)
			return
		}
		if err := s.db.UpdateManualAsset(&upd); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "updated"}, 200)

	case http.MethodDelete:
		if claims.Role == "viewer" || claims.Role == "user" {
			jsonError(w, "forbidden", 403)
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Reason) == "" {
			jsonError(w, "deletion reason is required", 400)
			return
		}
		if err := s.db.RecordAssetDeletion(id, "manual", existing.Name, existing.Branch, claims.Username, strings.TrimSpace(req.Reason)); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		if err := s.db.DeleteManualAsset(id); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "deleted"}, 200)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleSwitchRequests(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role == "adh" && claims.Branch == "" {
		jsonError(w, "ADH belum memiliki lokasi", 403)
		return
	}
	switch r.Method {
	case http.MethodGet:
		branch := r.URL.Query().Get("branch")
		username := ""
		if claims.Role == "adh" && claims.Branch != "" {
			branch = claims.Branch
		}
		if claims.Role == "user" {
			username = claims.Username
		}
		status := r.URL.Query().Get("status")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := s.db.ListSwitchRequests(branch, username, status, limit)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, items, 200)

	case http.MethodPost:
		var req struct {
			AssetID   string `json:"asset_id"`
			AssetType string `json:"asset_type"`
			ToOwner   string `json:"to_owner"`
			Reason    string `json:"reason"`
			SwapTag   string `json:"swap_tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		req.AssetID = strings.TrimSpace(req.AssetID)
		req.AssetType = strings.TrimSpace(req.AssetType)
		req.ToOwner = strings.TrimSpace(req.ToOwner)
		req.Reason = strings.TrimSpace(req.Reason)
		if req.AssetID == "" || req.ToOwner == "" || len([]rune(req.Reason)) < 10 || (req.AssetType != "manual" && req.AssetType != "device") {
			jsonError(w, "Aset dan pemegang tujuan wajib diisi; alasan minimal 10 karakter", 400)
			return
		}

		sw := &models.AssetSwitchRequest{
			AssetID:        req.AssetID,
			AssetType:      req.AssetType,
			ToOwner:        req.ToOwner,
			RequestedBy:    claims.Username,
			Reason:         req.Reason,
			Status:         "pending",
			Responsibility: assetResponsibilityWarning(),
		}

		if req.AssetType == "manual" {
			asset, err := s.db.GetManualAsset(req.AssetID)
			if err != nil {
				jsonError(w, "asset not found", 404)
				return
			}
			if claims.Role == "adh" && claims.Branch != "" && asset.Branch != claims.Branch {
				jsonError(w, "forbidden: asset belongs to different branch", 403)
				return
			}
			if claims.Role == "user" && asset.OwnerUsername != claims.Username {
				jsonError(w, "forbidden: asset bukan milik user ini", 403)
				return
			}
			sw.AssetName, sw.Branch, sw.FromOwner = asset.Name, asset.Branch, asset.OwnerUsername
			sw.Recommendation = manualAssetRecommendation(asset)
		} else {
			dev, err := s.db.GetDevice(req.AssetID)
			if err != nil {
				jsonError(w, "device not found", 404)
				return
			}
			branch := dev.Branch
			if branch == "" {
				branch = dev.GroupName
			}
			if claims.Role == "adh" && claims.Branch != "" && branch != claims.Branch {
				jsonError(w, "forbidden: perangkat milik cabang lain", 403)
				return
			}
			if claims.Role == "user" && dev.OwnerUsername != claims.Username {
				jsonError(w, "forbidden: asset bukan milik user ini", 403)
				return
			}
			sw.AssetName, sw.Branch, sw.FromOwner = dev.Hostname, branch, dev.OwnerUsername
			sw.Recommendation = deviceRecommendation(dev)
		}

		if claims.Role != "user" && !canApproveSwitch(claims.Role) {
			jsonError(w, "forbidden", 403)
			return
		}
		_, _, targetRole, targetBranch, targetErr := s.db.GetUser(req.ToOwner)
		if targetErr != nil || targetRole != "user" || targetBranch != sw.Branch || sw.FromOwner == sw.ToOwner {
			jsonError(w, "Pilih akun user lain yang terdaftar di lokasi aset", 400)
			return
		}
		if strings.TrimSpace(req.SwapTag) != "" {
			if sw.FromOwner == "" {
				jsonError(w, "Aset awal belum memiliki pemegang", 400)
				return
			}
			rows, err := s.db.db.Query(`SELECT id, 'manual' FROM manual_assets WHERE asset_tag=? AND owner_username=? AND branch=? UNION ALL SELECT id, 'device' FROM devices WHERE hostname=? AND owner_username=? AND COALESCE(NULLIF(branch,''),group_name)=?`, strings.TrimSpace(req.SwapTag), sw.ToOwner, sw.Branch, strings.TrimSpace(req.SwapTag), sw.ToOwner, sw.Branch)
			if err != nil {
				jsonError(w, "Gagal mencari aset pengganti", 500)
				return
			}
			count := 0
			for rows.Next() {
				if err := rows.Scan(&sw.SwapAssetID, &sw.SwapAssetType); err != nil {
					rows.Close()
					jsonError(w, "Gagal membaca aset pengganti", 500)
					return
				}
				count++
			}
			rows.Close()
			if count != 1 {
				jsonError(w, "Tag/hostname pengganti harus cocok dengan satu aset pemegang tujuan di lokasi yang sama", 400)
				return
			}
		}
		if err := s.db.CreateSwitchRequest(sw); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		if canApproveSwitch(claims.Role) {
			if err := s.db.ReviewSwitchRequest(sw.ID, "approved", claims.Username, "Switch langsung oleh "+claims.Role); err != nil {
				jsonError(w, err.Error(), 409)
				return
			}
			sw, _ = s.db.GetSwitchRequest(sw.ID)
		}
		jsonResp(w, sw, http.StatusCreated)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleSwitchRequestSubroute(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "adh" && getClaims(r).Branch == "" {
		jsonError(w, "ADH belum memiliki lokasi", 403)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if !canApproveSwitch(claims.Role) {
		jsonError(w, "only ADH, GA Pusat, or admin can review switch requests", 403)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/assets/switch-requests/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 {
		jsonError(w, "invalid switch request path", 400)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		jsonError(w, "invalid switch request id", 400)
		return
	}
	action := parts[1]
	status := "approved"
	if action == "reject" {
		status = "rejected"
	} else if action != "approve" {
		jsonError(w, "invalid switch action", 400)
		return
	}
	reqItem, err := s.db.GetSwitchRequest(id)
	if err != nil {
		jsonError(w, "switch request not found", 404)
		return
	}
	if claims.Role == "adh" && claims.Branch != "" && reqItem.Branch != claims.Branch {
		jsonError(w, "forbidden: request belongs to different branch", 403)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.db.ReviewSwitchRequest(id, status, claims.Username, body.Note); err != nil {
		jsonError(w, err.Error(), 409)
		return
	}
	jsonResp(w, map[string]string{"status": status}, 200)
}

// ---------------- VERIFICATION ----------------

func (s *Server) handleVerifyAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims.Role == "viewer" || claims.Role == "user" {
		jsonError(w, "forbidden", 403)
		return
	}

	var req struct {
		AssetID   string `json:"asset_id"`
		AssetType string `json:"asset_type"` // manual, device
		Status    string `json:"status"`     // verified, discrepancy
		Condition string `json:"condition"`  // good, fair, damaged
		Notes     string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	if req.AssetID == "" {
		jsonError(w, "asset_id is required", 400)
		return
	}
	if req.Status != "verified" && req.Status != "discrepancy" && req.Status != "unverified" {
		req.Status = "verified"
	}

	if claims.Role == "adh" && claims.Branch != "" {
		if req.AssetType == "manual" {
			existing, err := s.db.GetManualAsset(req.AssetID)
			if err != nil {
				jsonError(w, "asset not found", 404)
				return
			}
			if existing.Branch != claims.Branch {
				jsonError(w, "forbidden: aset milik cabang lain", 403)
				return
			}
		} else if req.AssetType == "device" {
			dev, err := s.db.GetDevice(req.AssetID)
			if err != nil {
				jsonError(w, "device not found", 404)
				return
			}
			devBranch := dev.Branch
			if devBranch == "" {
				devBranch = dev.GroupName
			}
			if devBranch != claims.Branch {
				jsonError(w, "forbidden: perangkat milik cabang lain", 403)
				return
			}
		}
	}

	verifier := claims.Username
	if claims.Branch != "" {
		verifier = fmt.Sprintf("%s (%s)", claims.Username, claims.Branch)
	}

	var err error
	if req.AssetType == "device" {
		err = s.db.VerifyDevice(req.AssetID, req.Status, req.Condition, verifier, req.Notes)
		s.db.AddLog(req.AssetID, "verified", fmt.Sprintf("status=%s by=%s note=%s", req.Status, verifier, req.Notes))
	} else {
		err = s.db.VerifyManualAsset(req.AssetID, req.Status, req.Condition, verifier, req.Notes)
	}

	if err != nil {
		jsonError(w, fmt.Sprintf("failed to verify asset: %v", err), 500)
		return
	}

	jsonResp(w, map[string]interface{}{
		"status":      "success",
		"asset_id":    req.AssetID,
		"verified_by": verifier,
	}, 200)
}

func (s *Server) handleAssetVerifications(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonResp(w, []models.AssetVerification{}, 200)
		return
	}
	claims := getClaims(r)
	assetID := r.URL.Query().Get("asset_id")
	branch := r.URL.Query().Get("branch")
	if claims.Role == "adh" && claims.Branch != "" {
		branch = claims.Branch
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := s.db.GetAssetVerificationsFiltered(
		assetID,
		branch,
		strings.TrimSpace(r.URL.Query().Get("status")),
		strings.TrimSpace(r.URL.Query().Get("verifier")),
		strings.TrimSpace(r.URL.Query().Get("from")),
		strings.TrimSpace(r.URL.Query().Get("to")),
		limit,
	)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, logs, 200)
}

// ---------------- USER MANAGEMENT ----------------

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can manage users", 403)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users, err := s.db.ListUsers()
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, users, 200)

	case http.MethodPost:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`   // admin, adh, viewer
			Branch   string `json:"branch"` // Cabang Surabaya, dll
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		if req.Username == "" || req.Password == "" {
			jsonError(w, "username and password are required", 400)
			return
		}
		if req.Role == "" {
			req.Role = "adh"
		}
		req.Username = strings.TrimSpace(req.Username)
		req.Branch = strings.TrimSpace(req.Branch)
		if req.Role != "admin" && req.Role != "ga_pusat" && req.Role != "adh" && req.Role != "viewer" && req.Role != "user" {
			jsonError(w, "invalid role", 400)
			return
		}
		if (req.Role == "adh" || req.Role == "user") && strings.TrimSpace(req.Branch) == "" {
			jsonError(w, "Lokasi wajib diisi", 400)
			return
		}
		if err := s.db.CreateUser(req.Username, hashPassword(req.Password), req.Role, req.Branch); err != nil {
			jsonError(w, fmt.Sprintf("failed to create user: %v", err), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "created"}, 201)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can delete users", 403)
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", 405)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/api/users/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid user id", 400)
		return
	}
	if err := s.db.DeleteUser(id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]string{"status": "deleted"}, 200)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	deviceID := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	logs, err := s.db.GetLogs(deviceID, limit)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, logs, 200)
}

func (s *Server) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("key")
	if apiKey != s.cfg.APIKey {
		http.Error(w, "unauthorized", 401)
		return
	}
	deviceID := r.URL.Query().Get("id")
	if deviceID == "" {
		http.Error(w, "missing device id", 400)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		DeviceID: deviceID,
		Conn:     conn,
		Send:     make(chan []byte, 64),
		Hub:      s.hub,
		IsAgent:  true,
	}

	s.hub.RegisterAgent(client)
	go client.WritePump()
	client.ReadPump(s.handleAgentMessage)
}

func (s *Server) handleViewerWS(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role == "user" {
		jsonError(w, "forbidden", 403)
		return
	}
	viewerID := r.URL.Query().Get("viewer_id")
	if viewerID == "" {
		viewerID = "viewer-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		DeviceID: viewerID,
		Conn:     conn,
		Send:     make(chan []byte, 64),
		Hub:      s.hub,
		IsAgent:  false,
	}

	s.hub.RegisterViewer(client)
	go client.WritePump()
	client.ReadPump(s.handleViewerMessage)
}

func (s *Server) handleRelayWS(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/relay/"), "/")
	if len(parts) != 2 || !validRelayID(parts[0]) {
		http.Error(w, "bad relay path, use /ws/relay/{session}/{role}", 400)
		return
	}
	sessionID := parts[0]
	role := parts[1]
	deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
	var audit relayAudit
	if deviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}

	switch role {
	case "viewer":
		claims, ok := parseToken(r.URL.Query().Get("token"), s.cfg.JWTSecret)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if claims.Role == "viewer" || claims.Role == "user" {
			http.Error(w, "remote access is not permitted for this role", http.StatusForbidden)
			return
		}
		dev, err := s.db.GetDevice(deviceID)
		if err != nil {
			http.Error(w, "device not found", http.StatusNotFound)
			return
		}
		deviceBranch := dev.Branch
		if deviceBranch == "" {
			deviceBranch = dev.GroupName
		}
		if claims.Role == "adh" && claims.Branch != "" && deviceBranch != claims.Branch {
			http.Error(w, "device is outside your location", http.StatusForbidden)
			return
		}
		if !s.hub.IsOnline(deviceID) {
			http.Error(w, "device is offline", http.StatusConflict)
			return
		}
		username, ip, userAgent := claims.Username, r.RemoteAddr, r.UserAgent()
		audit = relayAudit{
			onStart: func() {
				_ = s.db.RecordAuthLog(username, ip, "remote_start", "Remote desktop mulai: "+deviceID, userAgent)
			},
			onEnd: func(duration time.Duration) {
				_ = s.db.RecordAuthLog(username, ip, "remote_end", fmt.Sprintf("Remote desktop selesai: %s (%s)", deviceID, duration), userAgent)
			},
		}
	case "agent":
		if r.URL.Query().Get("key") != s.cfg.APIKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	default:
		http.Error(w, "invalid relay role", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if role == "viewer" {
		err = createViewerRelay(sessionID, deviceID, conn, audit)
	} else {
		err = attachAgentRelay(sessionID, deviceID, conn)
	}
	if err != nil {
		log.Printf("[relay] rejected %s for %s: %v", role, sessionID, err)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","message":"relay connection rejected"}`))
		_ = conn.Close()
	}
}

func validRelayID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, ch := range value {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && ch != '-' && ch != '_' {
			return false
		}
	}
	return true
}

func (s *Server) handleAgentMessage(c *Client, raw []byte) {
	var msg models.WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Action {
	case "register":
		var dev models.Device
		if err := json.Unmarshal(msg.Data, &dev); err != nil {
			return
		}
		dev.ID = c.DeviceID
		dev.LastSeen = time.Now()
		if dev.RegisteredAt.IsZero() {
			dev.RegisteredAt = time.Now()
		}
		if dev.Status == "" {
			dev.Status = "active"
		}
		if dev.Branch != "" {
			dev.GroupName = dev.Branch
		} else if dev.GroupName == "" {
			dev.GroupName = "default"
		}
		s.db.UpsertDevice(&dev)
		s.db.AddLog(dev.ID, "register", fmt.Sprintf("%s %s v%s", dev.Hostname, dev.OS, dev.Version))
		s.queueRustDeskBootstrap(&dev)

		// Updates are intentionally not pushed during registration. Administrators
		// stage them from the dashboard after validating a pilot device.

	case "heartbeat":
		var hb models.DeviceHeartbeat
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			return
		}
		hb.ID = c.DeviceID
		s.db.UpdateHeartbeat(&hb)

	case "rustdesk_result":
		var result models.RustDeskResult
		if err := json.Unmarshal(msg.Data, &result); err != nil {
			return
		}
		if result.RustDeskID != "" {
			_ = s.db.UpdateDeviceRustDeskID(c.DeviceID, result.RustDeskID)
		}
		detail := fmt.Sprintf("operation=%s request=%s", result.Operation, result.RequestID)
		if result.Error != "" {
			detail += " failed=" + result.Error
			if result.Operation == "install" {
				_ = s.db.SetSystemSetting(rustDeskBootstrapSetting(c.DeviceID), "")
			}
		} else {
			detail += " success"
			if result.Operation == "install" {
				if device, err := s.db.GetDevice(c.DeviceID); err == nil {
					_ = s.db.SetSystemSetting(rustDeskBootstrapSetting(c.DeviceID), device.Version)
				}
			}
		}
		_ = s.db.AddLog(c.DeviceID, "rustdesk_manage", detail)

	case "signal":
		var sig models.SignalMessage
		if err := json.Unmarshal(msg.Data, &sig); err != nil {
			return
		}
		sig.From = c.DeviceID
		s.hub.ForwardSignal(&sig)

	case "network_scan_result":
		var result struct {
			ScanID string                   `json:"scan_id"`
			Subnet string                   `json:"subnet"`
			Hosts  []models.NetworkScanHost `json:"hosts"`
			Error  string                   `json:"error"`
		}
		if err := json.Unmarshal(msg.Data, &result); err != nil || result.ScanID == "" {
			return
		}
		s.scanMu.Lock()
		if scan := s.scans[result.ScanID]; scan != nil && scan.DeviceID == c.DeviceID {
			now := time.Now()
			scan.Subnet, scan.Hosts, scan.Error, scan.FinishedAt = result.Subnet, result.Hosts, result.Error, &now
			if result.Error != "" {
				scan.Status = "failed"
			} else {
				scan.Status = "completed"
			}
		}
		s.scanMu.Unlock()
	}
}

func (s *Server) handleViewerMessage(c *Client, raw []byte) {
	var msg models.WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Action {
	case "signal":
		var sig models.SignalMessage
		if err := json.Unmarshal(msg.Data, &sig); err != nil {
			return
		}
		sig.From = c.DeviceID
		s.hub.ForwardSignal(&sig)
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	handler := next
	next = func(w http.ResponseWriter, r *http.Request) {
		c := getClaims(r)
		if c.Role == "user" {
			_, _, role, branch, err := s.db.GetUser(c.Username)
			if err != nil || role != c.Role || branch != c.Branch {
				jsonError(w, "unauthorized", 401)
				return
			}
			p := r.URL.Path
			allowed := strings.HasPrefix(p, "/api/auth/") || p == "/api/devices" || strings.HasPrefix(p, "/api/devices/") || p == "/api/assets/manual" || strings.HasPrefix(p, "/api/assets/manual/") || strings.HasPrefix(p, "/api/assets/switch-requests") || p == "/api/assets/verifications" || p == "/api/stats" || p == "/api/groups" || p == "/api/branches" || p == "/api/branches/stats"
			if !allowed {
				jsonError(w, "forbidden", 403)
				return
			}
		}
		handler(w, r)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			token := r.URL.Query().Get("token")
			if token == "" {
				token = r.URL.Query().Get("key")
			}
			if token != "" {
				if token == s.cfg.APIKey {
					ctx := context.WithValue(r.Context(), userClaimsKey, &UserClaims{
						Username: "api_key",
						Role:     "admin",
					})
					next(w, r.WithContext(ctx))
					return
				}
				if claims, ok := parseToken(token, s.cfg.JWTSecret); ok {
					ctx := context.WithValue(r.Context(), userClaimsKey, claims)
					next(w, r.WithContext(ctx))
					return
				}
			}
			http.Error(w, "unauthorized", 401)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			auth = r.URL.Query().Get("token")
		} else {
			auth = strings.TrimPrefix(auth, "Bearer ")
		}

		if auth == s.cfg.APIKey {
			ctx := context.WithValue(r.Context(), userClaimsKey, &UserClaims{
				Username: "api_key",
				Role:     "admin",
			})
			next(w, r.WithContext(ctx))
			return
		}

		if claims, ok := parseToken(auth, s.cfg.JWTSecret); ok {
			ctx := context.WithValue(r.Context(), userClaimsKey, claims)
			next(w, r.WithContext(ctx))
			return
		}

		jsonError(w, "unauthorized", 401)
	}
}

func getClaims(r *http.Request) *UserClaims {
	if c, ok := r.Context().Value(userClaimsKey).(*UserClaims); ok && c != nil {
		return c
	}
	return &UserClaims{Username: "anonymous", Role: "viewer"}
}

func jsonResp(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	jsonResp(w, map[string]string{"error": msg}, status)
}

func hashPassword(pass string) string {
	h := sha256.Sum256([]byte(pass))
	return hex.EncodeToString(h[:])
}

func generateToken(username, role, branch, secret string) string {
	payload := fmt.Sprintf("%s|%s|%s|%d", username, role, branch, time.Now().Add(24*time.Hour).Unix())
	h := sha256.Sum256([]byte(payload + secret))
	sig := hex.EncodeToString(h[:8])
	return hex.EncodeToString([]byte(payload)) + "." + sig
}

func parseToken(token, secret string) (*UserClaims, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	payloadBytes, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	payload := string(payloadBytes)

	h := sha256.Sum256([]byte(payload + secret))
	expectedSig := hex.EncodeToString(h[:8])
	if parts[1] != expectedSig {
		return nil, false
	}

	fields := strings.Split(payload, "|")
	if len(fields) == 4 {
		exp, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil || time.Now().Unix() >= exp {
			return nil, false
		}
		return &UserClaims{
			Username: fields[0],
			Role:     fields[1],
			Branch:   fields[2],
		}, true
	} else if len(fields) == 3 {
		exp, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || time.Now().Unix() >= exp {
			return nil, false
		}
		return &UserClaims{
			Username: fields[0],
			Role:     fields[1],
			Branch:   "",
		}, true
	}
	return nil, false
}

func (s *Server) handleAgentVersion(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, map[string]interface{}{
		"version":      s.cfg.Version,
		"download_url": "/api/agent/download",
	}, 200)
}

func (s *Server) handleAgentDownload(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key != s.cfg.APIKey {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != s.cfg.APIKey && func() bool { _, ok := parseToken(token, s.cfg.JWTSecret); return !ok }() {
			http.Error(w, "unauthorized", 401)
			return
		}
	}

	targetOS := strings.ToLower(r.URL.Query().Get("os"))
	targetArch := strings.ToLower(r.URL.Query().Get("arch"))
	if targetOS == "" {
		targetOS = "windows"
	}
	if targetArch == "" {
		targetArch = "amd64"
	}

	candidates := []string{}
	if s.cfg.AgentsDir != "" {
		candidates = append(candidates,
			filepath.Join(s.cfg.AgentsDir, fmt.Sprintf("rd-agent-%s-%s.exe", targetOS, targetArch)),
			filepath.Join(s.cfg.AgentsDir, fmt.Sprintf("rd-agent-%s-%s", targetOS, targetArch)),
			filepath.Join(s.cfg.AgentsDir, "rd-agent.exe"),
			filepath.Join(s.cfg.AgentsDir, "rd-agent"),
		)
	}
	candidates = append(candidates,
		filepath.Join("bin", "agents", fmt.Sprintf("rd-agent-%s-%s.exe", targetOS, targetArch)),
		filepath.Join("bin", "agents", fmt.Sprintf("rd-agent-%s-%s", targetOS, targetArch)),
		filepath.Join("bin", "rd-agent.exe"),
		filepath.Join("bin", "rd-agent"),
		"rd-agent.exe",
		"rd-agent",
	)

	var foundPath string
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
			foundPath = p
			break
		}
	}

	if foundPath == "" {
		http.Error(w, "agent binary not found on server", 404)
		return
	}

	fileName := filepath.Base(foundPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, foundPath)
}

func (s *Server) handleBroadcastAgentUpdate(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "forbidden", 403)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	count := 0
	for _, deviceID := range s.hub.OnlineIDs() {
		if status, err := s.queueAgentUpdate(deviceID, claims.Username); err == nil && status == "queued" {
			count++
		}
	}
	_ = s.db.RecordAuthLog(claims.Username, r.RemoteAddr, "agent_update_batch", fmt.Sprintf("Update agent massal ke v%s: %d perangkat", s.cfg.Version, count), r.UserAgent())
	jsonResp(w, map[string]interface{}{
		"status":          "broadcast_sent",
		"agents_notified": count,
		"version":         s.cfg.Version,
	}, 200)
}

func (s *Server) handleAgentUpdate(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.DeviceID) == "" {
		jsonError(w, "device_id is required", http.StatusBadRequest)
		return
	}
	status, err := s.queueAgentUpdate(strings.TrimSpace(req.DeviceID), claims.Username)
	if err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	_ = s.db.RecordAuthLog(claims.Username, r.RemoteAddr, "agent_update", fmt.Sprintf("Update agent %s ke v%s: %s", req.DeviceID, s.cfg.Version, status), r.UserAgent())
	jsonResp(w, map[string]interface{}{"status": status, "device_id": req.DeviceID, "version": s.cfg.Version}, http.StatusAccepted)
}

const rustDeskWindowsDownloadURL = "https://github.com/rustdesk/rustdesk/releases/download/1.4.6/rustdesk-1.4.6-x86_64.exe"
const rustDeskWindowsSHA256 = "422ce31131e6537ea4f611ebf4a4d1804f28a6f58c83aa05065071c5958f1551"
const rustDeskAutoInstallAgentVersion = "0.2.32"

func (s *Server) rustDeskInstallCommand(password string) models.RustDeskCommand {
	return models.RustDeskCommand{
		RequestID:   randomHex(12),
		Operation:   "install",
		Password:    password,
		Config:      s.rustDeskConfig(),
		DownloadURL: rustDeskWindowsDownloadURL,
		SHA256:      rustDeskWindowsSHA256,
	}
}

// queueRustDeskBootstrap makes RustDesk part of the Windows agent deployment.
// Starting with v0.2.32, an agent that reports no RustDesk ID receives an
// install/configure command as soon as it registers. The permanent password is
// intentionally left unset here and remains an explicit admin action.
func (s *Server) queueRustDeskBootstrap(dev *models.Device) {
	if dev == nil || !strings.HasPrefix(strings.ToLower(dev.OS), "windows") {
		return
	}
	if versioncmp.IsNewer(rustDeskAutoInstallAgentVersion, dev.Version) {
		return
	}
	if strings.TrimSpace(dev.RustDeskID) != "" && s.db.GetSystemSetting(rustDeskBootstrapSetting(dev.ID)) == dev.Version {
		return
	}
	command := s.rustDeskInstallCommand("")
	if command.Config == "" {
		_ = s.db.AddLog(dev.ID, "rustdesk_bootstrap", "skipped: self-host config is unavailable")
		return
	}
	raw, err := json.Marshal(map[string]interface{}{"action": "rustdesk_manage", "data": command})
	if err != nil || !s.hub.SendToAgent(dev.ID, raw) {
		_ = s.db.AddLog(dev.ID, "rustdesk_bootstrap", "failed to queue automatic install")
		return
	}
	_ = s.db.AddLog(dev.ID, "rustdesk_bootstrap", "automatic install/configuration queued")
}

func rustDeskBootstrapSetting(deviceID string) string {
	return "rustdesk_bootstrap:" + deviceID
}

func (s *Server) handleRustDeskSettings(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet {
		jsonResp(w, map[string]bool{"configured": s.rustDeskConfig() != ""}, http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Config string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	config := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(req.Config), `\`))
	if len(config) < 40 || len(config) > 4096 || !strings.HasPrefix(config, "=") {
		jsonError(w, "config RustDesk tidak valid", http.StatusBadRequest)
		return
	}
	if err := s.db.SetSystemSetting("rustdesk_config", config); err != nil {
		jsonError(w, "gagal menyimpan konfigurasi", http.StatusInternalServerError)
		return
	}
	_ = s.db.RecordAuthLog(claims.Username, r.RemoteAddr, "rustdesk_config", "Konfigurasi deployment RustDesk diperbarui", r.UserAgent())
	jsonResp(w, map[string]interface{}{"configured": true}, http.StatusOK)
}

func (s *Server) handleRustDeskManage(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		DeviceID  string `json:"device_id"`
		Operation string `json:"operation"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	if req.DeviceID == "" || (req.Operation != "install" && req.Operation != "set_password") {
		jsonError(w, "device_id atau operation tidak valid", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 || len(req.Password) > 64 || strings.ContainsAny(req.Password, "\r\n\x00") {
		jsonError(w, "password harus 8-64 karakter tanpa baris baru", http.StatusBadRequest)
		return
	}
	if !s.hub.IsOnline(req.DeviceID) {
		jsonError(w, "device sedang offline", http.StatusConflict)
		return
	}
	command := models.RustDeskCommand{
		RequestID: randomHex(12),
		Operation: req.Operation,
		Password:  req.Password,
	}
	if req.Operation == "install" {
		command = s.rustDeskInstallCommand(req.Password)
		if command.Config == "" {
			jsonError(w, "konfigurasi RustDesk self-host belum disimpan", http.StatusConflict)
			return
		}
		command.DownloadURL = rustDeskWindowsDownloadURL
		command.SHA256 = rustDeskWindowsSHA256
	}
	raw, err := json.Marshal(map[string]interface{}{"action": "rustdesk_manage", "data": command})
	if err != nil || !s.hub.SendToAgent(req.DeviceID, raw) {
		jsonError(w, "perintah RustDesk gagal dikirim", http.StatusConflict)
		return
	}
	_ = s.db.RecordAuthLog(claims.Username, r.RemoteAddr, "rustdesk_manage", fmt.Sprintf("%s untuk device %s", req.Operation, req.DeviceID), r.UserAgent())
	jsonResp(w, map[string]string{"status": "queued", "request_id": command.RequestID}, http.StatusAccepted)
}

func randomHex(byteCount int) string {
	buf := make([]byte, byteCount)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func (s *Server) rustDeskConfig() string {
	if configured := strings.TrimSpace(s.db.GetSystemSetting("rustdesk_config")); configured != "" {
		return configured
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s.cfg.RustDeskConfig), `\`))
}

func (s *Server) queueAgentUpdate(deviceID, actor string) (string, error) {
	device, err := s.db.GetDevice(deviceID)
	if err != nil {
		return "", fmt.Errorf("device not found")
	}
	if !versioncmp.IsNewer(s.cfg.Version, device.Version) {
		return "up_to_date", nil
	}
	if !s.hub.IsOnline(deviceID) {
		return "", fmt.Errorf("device is offline; update can be sent after it reconnects")
	}

	upMsg := map[string]interface{}{
		"action": "upgrade",
		"data": map[string]string{
			"version":      s.cfg.Version,
			"download_url": fmt.Sprintf("/api/agent/download?os=%s&arch=%s&key=%s", device.OS, device.Arch, s.cfg.APIKey),
		},
	}
	upRaw, err := json.Marshal(upMsg)
	if err != nil || !s.hub.SendToAgent(deviceID, upRaw) {
		return "", fmt.Errorf("agent update queue is unavailable")
	}
	_ = s.db.AddLog(deviceID, "agent_update_requested", fmt.Sprintf("v%s -> v%s by=%s", device.Version, s.cfg.Version, actor))
	return "queued", nil
}

func (s *Server) handleReconfigureAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "forbidden", 403)
		return
	}

	var req struct {
		ServerURL string `json:"server_url"`
		APIKey    string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	if req.ServerURL == "" {
		jsonError(w, "server_url is required", 400)
		return
	}

	s.hub.mu.RLock()
	count := len(s.hub.agents)
	for _, client := range s.hub.agents {
		msg := map[string]interface{}{
			"action": "reconfigure",
			"data": map[string]string{
				"server_url": req.ServerURL,
				"api_key":    req.APIKey,
			},
		}
		if raw, err := json.Marshal(msg); err == nil {
			select {
			case client.Send <- raw:
			default:
			}
		}
	}
	s.hub.mu.RUnlock()

	log.Printf("[server] reconfigure broadcast sent to %d agents: new endpoint=%s", count, req.ServerURL)
	jsonResp(w, map[string]interface{}{
		"status":          "reconfigure_sent",
		"agents_notified": count,
		"new_server_url":  req.ServerURL,
	}, 200)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims == nil || claims.Username == "" || claims.Username == "anonymous" {
		jsonError(w, "unauthorized", 401)
		return
	}

	var req struct {
		OldPassword     string `json:"old_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	if len(req.NewPassword) < 8 {
		jsonError(w, "Password baru minimal 8 karakter", 400)
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		jsonError(w, "Konfirmasi password baru tidak cocok", 400)
		return
	}
	if req.NewPassword == req.OldPassword {
		jsonError(w, "Password baru tidak boleh sama dengan password lama", 400)
		return
	}

	_, storedHash, _, _, err := s.db.GetUser(claims.Username)
	if err != nil {
		jsonError(w, "user not found", 404)
		return
	}
	if storedHash != hashPassword(req.OldPassword) {
		jsonError(w, "Password lama salah", 400)
		return
	}

	newHash := hashPassword(req.NewPassword)
	if err := s.db.UpdatePassword(claims.Username, newHash); err != nil {
		jsonError(w, fmt.Sprintf("failed to update password: %v", err), 500)
		return
	}
	s.db.RevokeTrustedDevices(claims.Username)

	log.Printf("[auth] user %s successfully changed password", claims.Username)
	jsonResp(w, map[string]string{"status": "success", "message": "Password berhasil diubah"}, 200)
}

func (s *Server) handleUserSubroute(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/reset-password") {
		s.handleResetUserPassword(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/reset-mfa") {
		s.handleResetUserMFA(w, r)
		return
	}
	s.handleUserDelete(w, r)
}

func (s *Server) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can reset user passwords", 403)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/users/")
	idStr = strings.TrimSuffix(idStr, "/reset-password")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid user id", 400)
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	if len(req.NewPassword) < 8 {
		jsonError(w, "Password baru minimal 8 karakter", 400)
		return
	}

	if err := s.db.ResetUserPassword(id, hashPassword(req.NewPassword)); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	log.Printf("[auth] admin %s reset password for user ID %d", claims.Username, id)
	jsonResp(w, map[string]string{"status": "success", "message": "Password user berhasil direset"}, 200)
}

func (s *Server) handleLoginMFA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	ip := r.RemoteAddr
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		ip = strings.TrimSpace(strings.Split(f, ",")[0])
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = strings.TrimSpace(realIP)
	}

	if !s.checkLoginRateLimit(ip) {
		jsonError(w, "Terlalu banyak percobaan login gagal. Diblokir sementara 15 menit.", 429)
		return
	}

	var req struct {
		MFATicket      string `json:"mfa_ticket"`
		Code           string `json:"code"`
		RememberDevice bool   `json:"remember_device"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	claims, ok := parseMFATicket(req.MFATicket, s.cfg.JWTSecret)
	if !ok {
		jsonError(w, "Sesi 2FA kedaluwarsa, silakan login ulang", 401)
		return
	}

	_, mfaSecret, err := s.db.GetUserMFA(claims.Username)
	if err != nil || !ValidateTOTPCode(mfaSecret, req.Code) {
		recordLoginFail(ip)
		_ = s.db.RecordAuthLog(claims.Username, ip, "mfa_failed", "Kode 2FA salah atau kedaluwarsa", r.UserAgent())
		log.Printf("[security] MFA LOGIN FAILED ip=%s username=%s", ip, claims.Username)
		jsonError(w, "Kode 2FA / Authenticator salah atau kedaluwarsa", 401)
		return
	}

	recordLoginSuccess(ip)
	_ = s.db.RecordAuthLog(claims.Username, ip, "success", "Login 2FA berhasil", r.UserAgent())
	log.Printf("[security] MFA LOGIN SUCCESS ip=%s username=%s role=%s", ip, claims.Username, claims.Role)
	token := generateToken(claims.Username, claims.Role, claims.Branch, s.cfg.JWTSecret)
	if req.RememberDevice {
		s.setTrustedDeviceCookie(w, claims.Username)
	}
	jsonResp(w, map[string]string{
		"token":    token,
		"username": claims.Username,
		"role":     claims.Role,
		"branch":   claims.Branch,
	}, 200)
}

func (s *Server) setTrustedDeviceCookie(w http.ResponseWriter, username string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return
	}
	value := hex.EncodeToString(b)
	expires := time.Now().Add(14 * 24 * time.Hour)
	if s.db.TrustDevice(username, hashPassword(value), expires) != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "rd_trusted_device", Value: value, Path: "/", Expires: expires, MaxAge: 14 * 24 * 60 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}

func (s *Server) handleMFAStatus(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	enabled, _, err := s.db.GetUserMFA(claims.Username)
	if err != nil {
		jsonError(w, "failed to get MFA status", 500)
		return
	}
	jsonResp(w, map[string]interface{}{"mfa_enabled": enabled}, 200)
}

func (s *Server) handleMFASetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	secret := GenerateTOTPSecret()
	otpauthURL := fmt.Sprintf("otpauth://totp/RemoteDesk:%s?secret=%s&issuer=RemoteDesk", url.PathEscape(claims.Username), secret)
	jsonResp(w, map[string]string{
		"secret":      secret,
		"otpauth_url": otpauthURL,
	}, 200)
}

func (s *Server) handleMFAEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)

	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}

	if !ValidateTOTPCode(req.Secret, req.Code) {
		jsonError(w, "Kode 2FA salah, pastikan waktu di HP Anda sudah akurat", 400)
		return
	}

	if err := s.db.SetUserMFA(claims.Username, req.Secret, true); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	log.Printf("[mfa] user %s enabled 2FA / MFA", claims.Username)
	jsonResp(w, map[string]string{"status": "success", "message": "2FA / MFA berhasil diaktifkan"}, 200)
}

func (s *Server) handleMFADisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid body", 400)
		return
	}

	_, storedHash, _, _, err := s.db.GetUser(claims.Username)
	if err != nil || storedHash != hashPassword(req.Password) {
		jsonError(w, "Password salah", 400)
		return
	}

	if err := s.db.SetUserMFA(claims.Username, "", false); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	log.Printf("[mfa] user %s disabled 2FA / MFA", claims.Username)
	jsonResp(w, map[string]string{"status": "success", "message": "2FA / MFA dinonaktifkan"}, 200)
}

func (s *Server) handleResetUserMFA(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can reset user MFA", 403)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/users/")
	idStr = strings.TrimSuffix(idStr, "/reset-mfa")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid user id", 400)
		return
	}

	if err := s.db.ResetUserMFA(id); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}

	log.Printf("[mfa] admin %s reset MFA for user ID %d", claims.Username, id)
	jsonResp(w, map[string]string{"status": "success", "message": "2FA / MFA user berhasil direset"}, 200)
}

func (s *Server) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims == nil || claims.Username == "" || claims.Username == "anonymous" {
		jsonError(w, "unauthorized", 401)
		return
	}

	var req struct {
		NewUsername string `json:"new_username"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}

	newUsername := strings.TrimSpace(req.NewUsername)
	if len(newUsername) < 3 || len(newUsername) > 32 {
		jsonError(w, "Username baru harus antara 3 - 32 karakter", 400)
		return
	}

	// Verify current password
	_, storedHash, role, branch, err := s.db.GetUser(claims.Username)
	if err != nil || storedHash != hashPassword(req.Password) {
		jsonError(w, "Password salah", 400)
		return
	}

	if newUsername == claims.Username {
		jsonError(w, "Username baru tidak boleh sama dengan username saat ini", 400)
		return
	}

	if err := s.db.UpdateUsername(claims.Username, newUsername); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}

	log.Printf("[auth] user %s renamed to %s", claims.Username, newUsername)
	token := generateToken(newUsername, role, branch, s.cfg.JWTSecret)
	jsonResp(w, map[string]interface{}{
		"status":   "success",
		"message":  "Username berhasil diubah",
		"username": newUsername,
		"token":    token,
	}, 200)
}

func (s *Server) handleAuthLogs(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can view security logs", 403)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := s.db.GetAuthLogsFiltered(
		strings.TrimSpace(r.URL.Query().Get("username")),
		strings.TrimSpace(r.URL.Query().Get("ip")),
		strings.TrimSpace(r.URL.Query().Get("status")),
		strings.TrimSpace(r.URL.Query().Get("from")),
		strings.TrimSpace(r.URL.Query().Get("to")),
		limit,
	)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, logs, 200)
}

func (s *Server) handleSecuritySettings(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can manage security settings", 403)
		return
	}

	switch r.Method {
	case http.MethodGet:
		settings := s.db.GetSecuritySettings()
		blocked := s.getBlockedIPs()
		jsonResp(w, map[string]interface{}{
			"settings":    settings,
			"blocked_ips": blocked,
		}, 200)

	case http.MethodPost:
		var req models.SecuritySettings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		if req.MaxLoginAttempts <= 0 {
			req.MaxLoginAttempts = 5
		}
		if req.BlockDurationMin <= 0 {
			req.BlockDurationMin = 15
		}
		if err := s.db.SaveSecuritySettings(req); err != nil {
			jsonError(w, fmt.Sprintf("failed to save settings: %v", err), 500)
			return
		}
		log.Printf("[security] admin %s updated security settings: max_attempts=%d block_duration=%dm enabled=%v",
			claims.Username, req.MaxLoginAttempts, req.BlockDurationMin, req.RateLimitEnabled)
		jsonResp(w, map[string]string{"status": "success", "message": "Pengaturan keamanan berhasil disimpan"}, 200)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleUnblockIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can unblock IPs", 403)
		return
	}

	var req struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IP == "" {
		jsonError(w, "ip is required", 400)
		return
	}

	s.unblockIP(req.IP)
	log.Printf("[security] admin %s manually unblocked IP %s", claims.Username, req.IP)
	jsonResp(w, map[string]string{"status": "success", "message": fmt.Sprintf("IP %s berhasil dibuka blokirnya", req.IP)}, 200)
}

func (s *Server) handleAgentPackageDownload(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role == "viewer" || claims.Role == "user" {
		jsonError(w, "forbidden", 403)
		return
	}

	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	if claims.Role == "adh" && claims.Branch != "" {
		branch = claims.Branch
	}
	if branch == "" {
		branch = "Pusat"
	}

	targetOS := strings.ToLower(r.URL.Query().Get("os"))
	targetArch := strings.ToLower(r.URL.Query().Get("arch"))
	if targetOS == "" {
		targetOS = "windows"
	}
	if targetArch == "" {
		targetArch = "amd64"
	}

	candidates := []string{}
	if s.cfg.AgentsDir != "" {
		candidates = append(candidates,
			filepath.Join(s.cfg.AgentsDir, fmt.Sprintf("rd-agent-%s-%s.exe", targetOS, targetArch)),
			filepath.Join(s.cfg.AgentsDir, fmt.Sprintf("rd-agent-%s-%s", targetOS, targetArch)),
			filepath.Join(s.cfg.AgentsDir, "rd-agent.exe"),
			filepath.Join(s.cfg.AgentsDir, "rd-agent"),
		)
	}
	candidates = append(candidates,
		filepath.Join("bin", "agents", fmt.Sprintf("rd-agent-%s-%s.exe", targetOS, targetArch)),
		filepath.Join("bin", "agents", fmt.Sprintf("rd-agent-%s-%s", targetOS, targetArch)),
		filepath.Join("bin", "rd-agent.exe"),
		filepath.Join("bin", "rd-agent"),
		"rd-agent.exe",
		"rd-agent",
	)

	var foundPath string
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
			foundPath = p
			break
		}
	}

	if foundPath == "" {
		jsonError(w, "File binary agent belum tersedia di server. Jalankan build agent terlebih dahulu.", 404)
		return
	}

	agentBytes, err := os.ReadFile(foundPath)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to read agent binary: %v", err), 500)
		return
	}

	// Determine server URL
	proto := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" || strings.Contains(r.Host, "synology.me") {
		proto = "https"
	}
	serverURL := fmt.Sprintf("%s://%s", proto, r.Host)

	// Create pre-configured agent.json
	cfgObj := map[string]interface{}{
		"server_url": serverURL,
		"api_key":    s.cfg.APIKey,
		"branch":     branch,
		// Keep the identity key present even in a fresh package.  The installer
		// can then safely retain an existing device ID during an in-place upgrade.
		"device_id":         "",
		"heartbeat_seconds": 60,
		// Agent updates are distributed by this server. GitHub Releases currently
		// contains an older legacy build and must not be used as an update source.
		"update_url": "",
	}
	cfgBytes, _ := json.MarshalIndent(cfgObj, "", "  ")

	// Create in-memory zip
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	binName := "rd-agent.exe"
	if targetOS != "windows" {
		binName = "rd-agent"
	}
	fBin, err := zw.Create(binName)
	if err == nil {
		_, _ = fBin.Write(agentBytes)
	}

	fCfg, err := zw.Create("agent.json")
	if err == nil {
		_, _ = fCfg.Write(cfgBytes)
	}

	if targetOS == "windows" {
		uiAccessCandidates := []string{}
		if s.cfg.AgentsDir != "" {
			uiAccessCandidates = append(uiAccessCandidates, filepath.Join(s.cfg.AgentsDir, "rd-agent-uiaccess.exe"))
		}
		uiAccessCandidates = append(uiAccessCandidates,
			filepath.Join("bin", "agents", "rd-agent-uiaccess.exe"),
			filepath.Join("bin", "rd-agent-uiaccess.exe"),
			"rd-agent-uiaccess.exe",
		)
		var uiAccessBytes []byte
		for _, candidate := range uiAccessCandidates {
			if bytes, readErr := os.ReadFile(candidate); readErr == nil && len(bytes) > 0 {
				uiAccessBytes = bytes
				break
			}
		}
		if len(uiAccessBytes) == 0 {
			jsonError(w, "File UIAccess agent belum tersedia di server.", 503)
			return
		}
		if fUIAccess, createErr := zw.Create("rd-agent-uiaccess.exe"); createErr == nil {
			_, _ = fUIAccess.Write(uiAccessBytes)
		}

		// UIAccess only accepts an executable whose certificate chains to a root
		// trusted by Windows.  The root is public (the signing key is never put
		// in this ZIP) and is installed once by the elevated installer below.
		certCandidates := []string{
			filepath.Join(filepath.Dir(s.cfg.AgentsDir), "certs", "RemoteDesk-Internal-Root.cer"),
			filepath.Join("certs", "RemoteDesk-Internal-Root.cer"),
			filepath.Join("..", "..", "certs", "RemoteDesk-Internal-Root.cer"),
		}
		for _, certPath := range certCandidates {
			if certBytes, readErr := os.ReadFile(certPath); readErr == nil && len(certBytes) > 0 {
				if fCert, createErr := zw.Create("RemoteDesk-Internal-Root.cer"); createErr == nil {
					_, _ = fCert.Write(certBytes)
				}
				break
			}
		}

		// 1. start-hidden.vbs for silent background execution
		vbsContent := `Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)
WshShell.Run chr(34) & WshShell.CurrentDirectory & "\rd-agent.exe" & chr(34), 0, False
`
		if fVbs, err := zw.Create("start-hidden.vbs"); err == nil {
			_, _ = fVbs.Write([]byte(vbsContent))
		}

		installServicePS := `$ErrorActionPreference = "Stop"
$packageDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$installDir = Join-Path $env:ProgramFiles "RemoteDesk\Agent"
$configDir = Join-Path $env:ProgramData "RemoteDesk\Agent"
$legacyInstallDir = $configDir
$serviceName = "RemoteDeskAgent"
$previousDeviceID = ""
foreach ($previousConfig in @((Join-Path $configDir "agent.json"), (Join-Path $legacyInstallDir "agent.json"))) {
    if (Test-Path -LiteralPath $previousConfig) {
        try { $previousDeviceID = (Get-Content -LiteralPath $previousConfig -Raw | ConvertFrom-Json).device_id } catch {}
        if (-not [string]::IsNullOrWhiteSpace($previousDeviceID)) { break }
    }
}
if ([string]::IsNullOrWhiteSpace($previousDeviceID)) {
    $legacyIdFile = Join-Path $env:TEMP "rd-device-id"
    if (Test-Path -LiteralPath $legacyIdFile) { $previousDeviceID = (Get-Content -LiteralPath $legacyIdFile -Raw).Trim() }
}
# Verify the new package before interrupting a working service.  The root is
# public; the signing private key is not present on endpoints or in this ZIP.
$rootCert = Join-Path $packageDir "RemoteDesk-Internal-Root.cer"
if (-not (Test-Path -LiteralPath $rootCert)) { throw "Sertifikat root RemoteDesk tidak ditemukan di paket" }
& certutil.exe -addstore -f Root $rootCert | Out-Null
if ($LASTEXITCODE -ne 0) { throw "Gagal memasang sertifikat root RemoteDesk" }
& certutil.exe -addstore -f TrustedPublisher $rootCert | Out-Null
if ($LASTEXITCODE -ne 0) { throw "Gagal memasang sertifikat penerbit RemoteDesk" }
$signature = Get-AuthenticodeSignature -FilePath (Join-Path $packageDir "rd-agent.exe")
if ($signature.Status -ne 'Valid') { throw "Agent tidak memiliki tanda tangan digital yang valid: $($signature.Status)" }
$uiAccessSignature = Get-AuthenticodeSignature -FilePath (Join-Path $packageDir "rd-agent-uiaccess.exe")
if ($uiAccessSignature.Status -ne 'Valid') { throw "UIAccess agent tidak memiliki tanda tangan digital yang valid: $($uiAccessSignature.Status)" }
# Stop every process that can hold rd-agent.exe before overwriting it.  A
# Windows service keeps its image file open, so copying first makes upgrades
# fail deterministically with "being used by another process".
if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    $service = Get-Service -Name $serviceName
    $service.WaitForStatus('Stopped', (New-TimeSpan -Seconds 20))
}
Get-Process -Name "rd-agent", "rd-agent-uiaccess" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
for ($attempt = 0; $attempt -lt 40; $attempt++) {
    $remaining = Get-Process -Name "rd-agent", "rd-agent-uiaccess" -ErrorAction SilentlyContinue
    if ($null -eq $remaining) { break }
    Start-Sleep -Milliseconds 250
}
if (Get-Process -Name "rd-agent", "rd-agent-uiaccess" -ErrorAction SilentlyContinue) {
    throw "Proses RemoteDesk lama tidak dapat dihentikan. Restart Windows lalu jalankan installer kembali."
}
function Copy-WithRetry([string]$Source, [string]$Destination) {
    for ($attempt = 1; $attempt -le 20; $attempt++) {
        try {
            Copy-Item -LiteralPath $Source -Destination $Destination -Force
            return
        } catch {
            if ($attempt -eq 20) { throw }
            Start-Sleep -Milliseconds 250
        }
    }
}
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
New-Item -ItemType Directory -Path $configDir -Force | Out-Null
Copy-WithRetry (Join-Path $packageDir "rd-agent.exe") (Join-Path $installDir "rd-agent.exe")
Copy-WithRetry (Join-Path $packageDir "rd-agent-uiaccess.exe") (Join-Path $installDir "rd-agent-uiaccess.exe")
Copy-WithRetry (Join-Path $packageDir "agent.json") (Join-Path $configDir "agent.json")
$exe = Join-Path $installDir "rd-agent.exe"
$config = Join-Path $configDir "agent.json"
if (-not [string]::IsNullOrWhiteSpace($previousDeviceID)) {
    $newConfig = Get-Content -LiteralPath $config -Raw | ConvertFrom-Json
    # Add-Member handles both packages created before device_id existed and
    # current packages that already include the property.
    $newConfig | Add-Member -NotePropertyName "device_id" -NotePropertyValue $previousDeviceID -Force
    $configJSON = $newConfig | ConvertTo-Json -Depth 8
    [System.IO.File]::WriteAllText($config, $configJSON, (New-Object System.Text.UTF8Encoding($false)))
}
$binPath = '"' + $exe + '" --system-service --config "' + $config + '"'
$existingService = Get-CimInstance -ClassName Win32_Service -Filter "Name='$serviceName'" -ErrorAction SilentlyContinue
if ($null -ne $existingService) {
    $changeResult = Invoke-CimMethod -InputObject $existingService -MethodName Change -Arguments @{ PathName = $binPath; StartMode = 'Automatic' }
    if ($changeResult.ReturnValue -ne 0) { throw "Konfigurasi Windows service gagal diperbarui (kode $($changeResult.ReturnValue))" }
} else {
    New-Service -Name $serviceName -BinaryPathName $binPath -StartupType Automatic -DisplayName "RemoteDesk Agent" | Out-Null
}
& sc.exe description $serviceName "RemoteDesk secure desktop remote service" | Out-Null
& sc.exe failure $serviceName reset= 86400 actions= restart/5000/restart/15000/restart/60000 | Out-Null
Start-Service -Name $serviceName
if ((Get-Service -Name $serviceName).Status -ne 'Running') { throw "Windows service gagal dijalankan" }
`
		if fService, err := zw.Create("install-service.ps1"); err == nil {
			_, _ = fService.Write([]byte(installServicePS))
		}

		// 2. pasang-otomatis.bat (one-time elevation, then silent highest-privilege logon task)
		installBat := fmt.Sprintf(`@echo off
title Pasang RemoteDesk Agent - %s
echo =========================================================
echo   Memasang RemoteDesk Agent (Cabang: %s)
echo =========================================================
echo.

if not exist "%%~dp0rd-agent.exe" (
    echo Error: rd-agent.exe tidak ditemukan di folder ini!
    pause
    exit /b 1
)

:: Remote melalui lock screen membutuhkan Windows service SYSTEM.
:: Satu kali persetujuan Administrator memasang service ini secara permanen.
fltmc >nul 2>&1
if errorlevel 1 (
    set "RD_INSTALLER=%%~f0"
    set "RD_AGENT_DIR=%%~dp0"
    powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Start-Process -FilePath $env:RD_INSTALLER -WorkingDirectory $env:RD_AGENT_DIR -Verb RunAs"
    exit /b
)

:: Hapus Mark of the Web dari seluruh file hasil ekstraksi agar Windows tidak
:: menampilkan Open File - Security Warning setiap kali Agent auto-start.
set "RD_AGENT_DIR=%%~dp0"
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Get-ChildItem -LiteralPath $env:RD_AGENT_DIR -File | Unblock-File"
if errorlevel 1 (
    echo [ERROR] Status blokir file dari internet gagal dihapus.
    echo Klik kanan rd-agent.exe, pilih Properties, lalu centang Unblock.
    pause
    exit /b 1
)

:: Matikan proses agent lama jika sedang berjalan
taskkill /f /im rd-agent.exe >nul 2>&1

:: Hapus mekanisme lama agar tidak ada dua agent untuk device yang sama.
del "%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\Startup\RemoteDesk-Agent.lnk" >nul 2>&1
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Unregister-ScheduledTask -TaskName 'RemoteDesk Agent' -Confirm:$false -ErrorAction SilentlyContinue"
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "%%~dp0install-service.ps1"
if errorlevel 1 (
    echo [ERROR] Windows service RemoteDesk gagal dipasang.
    pause
    exit /b 1
)

echo [OK] Windows service SYSTEM berhasil dipasang.
echo [OK] Agent RemoteDesk sudah aktif di background, termasuk lock screen.
echo.
echo =========================================================
echo   SUKSES!
echo   Komputer ini sekarang sudah terhubung ke server dan
echo   tetap dapat diremote saat layar Windows terkunci.
echo =========================================================
echo.
timeout /t 5
`, branch, branch)
		if fInstall, err := zw.Create("pasang-otomatis.bat"); err == nil {
			_, _ = fInstall.Write([]byte(installBat))
		}

		// 3. run-agent.bat (quick launcher)
		batContent := fmt.Sprintf(`@echo off
title RemoteDesk Agent - %s
echo ===================================================
echo   Memulai RemoteDesk Agent Cabang: %s
echo ===================================================
if exist "%%~dp0rd-agent.exe" (
    "%%~dp0rd-agent.exe" %%*
) else (
    echo Error: rd-agent.exe tidak ditemukan!
    pause
    exit /b 1
)
if %%ERRORLEVEL%% NEQ 0 (
    echo.
    echo Agent terhenti. Pastikan koneksi internet aktif.
    pause
)
`, branch, branch)
		if fBat, err := zw.Create("run-agent.bat"); err == nil {
			_, _ = fBat.Write([]byte(batContent))
		}

		// 4. hapus-otomatis.bat (uninstaller for scheduled task and legacy Startup)
		uninstallBat := `@echo off
title Hapus RemoteDesk Auto-Start
fltmc >nul 2>&1
if errorlevel 1 (
    set "RD_INSTALLER=%~f0"
    set "RD_AGENT_DIR=%~dp0"
    powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Start-Process -FilePath $env:RD_INSTALLER -WorkingDirectory $env:RD_AGENT_DIR -Verb RunAs"
    exit /b
)
echo Mematikan dan mencopot auto-start RemoteDesk...
taskkill /f /im rd-agent.exe >nul 2>&1
del "%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\RemoteDesk-Agent.lnk" >nul 2>&1
powershell.exe -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "Unregister-ScheduledTask -TaskName 'RemoteDesk Agent' -Confirm:$false -ErrorAction SilentlyContinue; Stop-Service -Name 'RemoteDeskAgent' -Force -ErrorAction SilentlyContinue; sc.exe delete RemoteDeskAgent | Out-Null"
echo Selesai! Auto-start telah dihapus.
pause
`
		if fUn, err := zw.Create("hapus-otomatis.bat"); err == nil {
			_, _ = fUn.Write([]byte(uninstallBat))
		}
	}

	readmeContent := fmt.Sprintf(`=========================================================
PANDUAN PEMASANGAN REMOTEDESK AGENT
Cabang / Site : %s
Server URL    : %s
=========================================================

CARA PASANG PALING MUDAH:
1. Ekstrak semua isi file ZIP ini ke salah satu folder (misal: C:\RemoteDesk\).
2. KLIK DUA KALI file "pasang-otomatis.bat", lalu klik Yes satu kali pada permintaan Administrator.
3. SELESAI!
   - Agent akan langsung berjalan diam-diam di background (tanpa jendela hitam).
   - Agent dijalankan oleh Windows service SYSTEM dan tetap tersedia saat lock screen.
   - Installer memasang sertifikat RemoteDesk internal untuk memverifikasi agent bertanda tangan digital.

KETERANGAN FILE:
- pasang-otomatis.bat : Memasang Windows service dan menjalankan agent di background.
- run-agent.bat       : Menjalankan agent di jendela hitam untuk tes melihat log.
- hapus-otomatis.bat  : Menghapus service dan auto-start agent.
`, branch, serverURL)
	if fReadme, err := zw.Create("PETUNJUK_CARA_PAKAI.txt"); err == nil {
		_, _ = fReadme.Write([]byte(readmeContent))
	}

	_ = zw.Close()

	cleanBranch := strings.ReplaceAll(branch, " ", "-")
	versionSuffix := strings.TrimSpace(s.cfg.Version)
	if versionSuffix == "" {
		versionSuffix = "current"
	}
	// The package URL is stable; preventing HTTP caching ensures the binary in
	// a freshly downloaded ZIP is from this server release.
	zipName := fmt.Sprintf("RemoteDesk-Agent-%s-v%s.zip", cleanBranch, versionSuffix)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", zipName))
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) handleBranchSubroute(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "only admin can manage branches", 403)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/branches/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid branch id", 400)
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			BusinessUnit  string   `json:"business_unit"`
			BusinessUnits []string `json:"business_units"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if err := s.db.UpdateBranch(id, req.Name, req.Type, req.BusinessUnit); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		if err := s.db.SetBranchBusinessUnits(id, req.BusinessUnits); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "updated"}, 200)

	case http.MethodDelete:
		if err := s.db.DeleteBranch(id); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "deleted"}, 200)

	default:
		http.Error(w, "method not allowed", 405)
	}
}
