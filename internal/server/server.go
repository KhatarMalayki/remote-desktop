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
)

type Config struct {
	Addr      string
	DBPath    string
	APIKey    string
	AdminUser string
	AdminPass string
	JWTSecret string
	Version   string
	AgentsDir string
}

type UserClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Branch   string `json:"branch"`
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
	mux.HandleFunc("/api/branches/stats", s.authMiddleware(s.handleBranchStats))
	mux.HandleFunc("/api/assets/manual", s.authMiddleware(s.handleManualAssets))
	mux.HandleFunc("/api/assets/manual/", s.authMiddleware(s.handleManualAsset))
	mux.HandleFunc("/api/assets/verify", s.authMiddleware(s.handleVerifyAsset))
	mux.HandleFunc("/api/assets/verifications", s.authMiddleware(s.handleAssetVerifications))
	mux.HandleFunc("/api/agent/version", s.handleAgentVersion)
	mux.HandleFunc("/api/agent/download", s.handleAgentDownload)
	mux.HandleFunc("/api/agent/package", s.authMiddleware(s.handleAgentPackageDownload))
	mux.HandleFunc("/api/agent/broadcast-update", s.authMiddleware(s.handleBroadcastAgentUpdate))
	mux.HandleFunc("/api/agent/reconfigure", s.authMiddleware(s.handleReconfigureAgents))
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
		Username string `json:"username"`
		Password string `json:"password"`
		Code     string `json:"code"`
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
	if mfaEnabled {
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
		if claims.Role == "kacab" && claims.Branch != "" {
			group = claims.Branch
		}
		search := r.URL.Query().Get("search")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		if limit <= 0 || limit > 100 {
			limit = 50
		}

		devices, total, err := s.db.ListDevices(group, search, limit, offset)
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
		}

		jsonResp(w, map[string]interface{}{
			"devices": devices,
			"total":   total,
			"limit":   limit,
			"offset":  offset,
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

	// Strict branch isolation for Kacab
	if claims.Role == "kacab" && claims.Branch != "" {
		devBranch := dev.Branch
		if devBranch == "" {
			devBranch = dev.GroupName
		}
		if devBranch != claims.Branch {
			jsonError(w, "forbidden: perangkat milik cabang lain", 403)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		dev.Online = s.hub.IsOnline(id)
		jsonResp(w, dev, 200)

	case http.MethodPut:
		if claims.Role == "viewer" {
			jsonError(w, "forbidden", 403)
			return
		}
		var req struct {
			Tags  string `json:"tags"`
			Group string `json:"group"`
			Note  string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid body", 400)
			return
		}
		if claims.Role == "kacab" && claims.Branch != "" {
			req.Group = claims.Branch
		}
		if err := s.db.UpdateDeviceMeta(id, req.Tags, req.Group, req.Note); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		s.db.AddLog(id, "update_meta", fmt.Sprintf("tags=%s group=%s", req.Tags, req.Group))
		jsonResp(w, map[string]string{"status": "ok"}, 200)

	case http.MethodDelete:
		if claims.Role != "admin" {
			jsonError(w, "only admin can delete monitored devices", 403)
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

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.Stats()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	stats["online_devices"] = s.hub.OnlineCount()
	jsonResp(w, stats, 200)
}

func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.db.GetGroups()
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, groups, 200)
}

func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
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
			Name string `json:"name"`
			Type string `json:"type"`
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
		if err := s.db.CreateBranch(req.Name, req.Type); err != nil {
			jsonError(w, err.Error(), 400)
			return
		}
		jsonResp(w, map[string]string{"status": "created"}, 201)

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func (s *Server) handleBranchStats(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	branch := r.URL.Query().Get("branch")
	if claims.Role == "kacab" && claims.Branch != "" {
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
		if claims.Role == "kacab" && claims.Branch != "" {
			branch = claims.Branch
		}
		category := r.URL.Query().Get("category")
		verificationStatus := r.URL.Query().Get("verification_status")
		search := r.URL.Query().Get("search")

		assets, err := s.db.ListManualAssets(branch, category, verificationStatus, search)
		if err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, assets, 200)

	case http.MethodPost:
		if claims.Role == "viewer" {
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
		if claims.Role == "kacab" && claims.Branch != "" {
			asset.Branch = claims.Branch
		}
		if asset.Branch == "" {
			asset.Branch = "Pusat"
		}
		asset.CreatedBy = claims.Username

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
	if claims.Role == "kacab" && claims.Branch != "" && existing.Branch != claims.Branch {
		jsonError(w, "forbidden: asset belongs to different branch", 403)
		return
	}

	switch r.Method {
	case http.MethodGet:
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
		if claims.Role == "kacab" && claims.Branch != "" {
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
		if err := s.db.UpdateManualAsset(&upd); err != nil {
			jsonError(w, err.Error(), 500)
			return
		}
		jsonResp(w, map[string]string{"status": "updated"}, 200)

	case http.MethodDelete:
		if claims.Role == "viewer" {
			jsonError(w, "forbidden", 403)
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

// ---------------- VERIFICATION ----------------

func (s *Server) handleVerifyAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	claims := getClaims(r)
	if claims.Role == "viewer" {
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

	if claims.Role == "kacab" && claims.Branch != "" {
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
	claims := getClaims(r)
	assetID := r.URL.Query().Get("asset_id")
	branch := r.URL.Query().Get("branch")
	if claims.Role == "kacab" && claims.Branch != "" {
		branch = claims.Branch
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	logs, err := s.db.GetAssetVerifications(assetID, branch, limit)
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
			Role     string `json:"role"`   // admin, kacab, viewer
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
			req.Role = "kacab"
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
	if len(parts) < 2 {
		http.Error(w, "bad relay path, use /ws/relay/{session}/{role}", 400)
		return
	}
	sessionID := parts[0]
	role := parts[1]

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	s.hub.HandleRelay(sessionID, role, conn)
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

		if s.cfg.Version != "" && dev.Version != "" && dev.Version != s.cfg.Version {
			log.Printf("[server] agent %s is on version %s, server is %s. Triggering upgrade.", dev.ID, dev.Version, s.cfg.Version)
			upMsg := map[string]interface{}{
				"action": "upgrade",
				"data": map[string]string{
					"version":      s.cfg.Version,
					"download_url": fmt.Sprintf("/api/agent/download?os=%s&arch=%s&key=%s", dev.OS, dev.Arch, s.cfg.APIKey),
				},
			}
			if upRaw, err := json.Marshal(upMsg); err == nil {
				select {
				case c.Send <- upRaw:
				default:
				}
			}
		}

	case "heartbeat":
		var hb models.DeviceHeartbeat
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			return
		}
		hb.ID = c.DeviceID
		s.db.UpdateHeartbeat(&hb)

	case "signal":
		var sig models.SignalMessage
		if err := json.Unmarshal(msg.Data, &sig); err != nil {
			return
		}
		sig.From = c.DeviceID
		s.hub.ForwardSignal(&sig)
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
	s.hub.mu.RLock()
	count := len(s.hub.agents)
	for _, client := range s.hub.agents {
		upMsg := map[string]interface{}{
			"action": "upgrade",
			"data": map[string]string{
				"version":      s.cfg.Version,
				"download_url": fmt.Sprintf("/api/agent/download?key=%s", s.cfg.APIKey),
			},
		}
		if upRaw, err := json.Marshal(upMsg); err == nil {
			select {
			case client.Send <- upRaw:
			default:
			}
		}
	}
	s.hub.mu.RUnlock()
	jsonResp(w, map[string]interface{}{
		"status":          "broadcast_sent",
		"agents_notified": count,
		"version":         s.cfg.Version,
	}, 200)
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
		MFATicket string `json:"mfa_ticket"`
		Code      string `json:"code"`
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
	jsonResp(w, map[string]string{
		"token":    token,
		"username": claims.Username,
		"role":     claims.Role,
		"branch":   claims.Branch,
	}, 200)
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
	logs, err := s.db.GetAuthLogs(limit)
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
	if claims.Role == "viewer" {
		jsonError(w, "forbidden", 403)
		return
	}

	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	if claims.Role == "kacab" && claims.Branch != "" {
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
		"server_url":        serverURL,
		"api_key":           s.cfg.APIKey,
		"branch":            branch,
		"heartbeat_seconds": 60,
		"update_url":        "https://github.com/KhatarMalayki/remote-desktop/releases/latest",
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
		// 1. start-hidden.vbs for silent background execution
		vbsContent := `Set WshShell = CreateObject("WScript.Shell")
WshShell.CurrentDirectory = CreateObject("Scripting.FileSystemObject").GetParentFolderName(WScript.ScriptFullName)
WshShell.Run chr(34) & WshShell.CurrentDirectory & "\rd-agent.exe" & chr(34), 0, False
`
		if fVbs, err := zw.Create("start-hidden.vbs"); err == nil {
			_, _ = fVbs.Write([]byte(vbsContent))
		}

		// 2. pasang-otomatis.bat (auto-start via Startup folder without requiring Run As Administrator)
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

:: Matikan proses agent lama jika sedang berjalan
taskkill /f /im rd-agent.exe >nul 2>&1

:: Daftarkan ke folder Startup Windows (Otomatis Jalan Tiap Komputer Nyala)
set "STARTUP_FOLDER=%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\Startup"
echo Set oWS = WScript.CreateObject("WScript.Shell") > "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo sLinkFile = "%%STARTUP_FOLDER%%\RemoteDesk-Agent.lnk" >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo Set oLink = oWS.CreateShortcut(sLinkFile) >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo oLink.TargetPath = "wscript.exe" >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo oLink.Arguments = chr(34) ^^& "%%~dp0start-hidden.vbs" ^^& chr(34) >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo oLink.WorkingDirectory = "%%~dp0" >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo oLink.WindowStyle = 7 >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
echo oLink.Save >> "%%TEMP%%\CreateRemoteDeskLnk.vbs"
cscript //nologo "%%TEMP%%\CreateRemoteDeskLnk.vbs"
del "%%TEMP%%\CreateRemoteDeskLnk.vbs" >nul 2>&1

:: Jalankan agent sekarang di background secara silent (tanpa jendela hitam)
wscript.exe "%%~dp0start-hidden.vbs"

echo [OK] Shortcut Startup berhasil dipasang.
echo [OK] Agent RemoteDesk sudah aktif di background!
echo.
echo =========================================================
echo   SUKSES!
echo   Komputer ini sekarang sudah terhubung ke server dan
echo   akan OTOMATIS JALAN setiap kali komputer dinyalakan.
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

		// 4. hapus-otomatis.bat (uninstaller from Startup)
		uninstallBat := `@echo off
title Hapus RemoteDesk Auto-Start
echo Mematikan dan mencopot auto-start RemoteDesk...
taskkill /f /im rd-agent.exe >nul 2>&1
del "%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\RemoteDesk-Agent.lnk" >nul 2>&1
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
2. Cukup KLIK DUA KALI file "pasang-otomatis.bat" (TIDAK PERLU Run as Administrator!).
3. SELESAI!
   - Agent akan langsung berjalan diam-diam di background (tanpa jendela hitam).
   - Agent akan OTOMATIS JALAN SENDIRI setiap kali komputer dinyalakan / direstart.

KETERANGAN FILE:
- pasang-otomatis.bat : Mengaktifkan auto-start dan menjalankan agent di background.
- run-agent.bat       : Menjalankan agent di jendela hitam untuk tes melihat log.
- hapus-otomatis.bat  : Menghapus agent dari daftar startup Windows.
`, branch, serverURL)
	if fReadme, err := zw.Create("PETUNJUK_CARA_PAKAI.txt"); err == nil {
		_, _ = fReadme.Write([]byte(readmeContent))
	}

	_ = zw.Close()

	cleanBranch := strings.ReplaceAll(branch, " ", "-")
	zipName := fmt.Sprintf("RemoteDesk-Agent-%s.zip", cleanBranch)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", zipName))
	w.Header().Set("Content-Type", "application/zip")
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
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "invalid request body", 400)
			return
		}
		req.Name = strings.TrimSpace(req.Name)
		if err := s.db.UpdateBranch(id, req.Name, req.Type); err != nil {
			jsonError(w, err.Error(), 400)
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
