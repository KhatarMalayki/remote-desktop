package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/user/remote-desktop/internal/versioncmp"
)

type apiKeyState struct {
	Current  string            `json:"current"`
	Old      []string          `json:"old"`
	Verified map[string]string `json:"verified"`
}

func keyID(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (s *Server) loadAPIKeys() error {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	var raw string
	err := s.db.db.QueryRow("SELECT value FROM system_settings WHERE key = ?", "agent_api_keys").Scan(&raw)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("read API key state: %w", err)
	}
	state := apiKeyState{Current: s.cfg.APIKey, Verified: map[string]string{}}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &state); err != nil || state.Current == "" {
			return fmt.Errorf("invalid persisted agent API key state")
		}
	}
	if state.Verified == nil {
		state.Verified = map[string]string{}
	}
	return s.saveAPIKeysLocked(state)
}

func (s *Server) saveAPIKeysLocked(state apiKeyState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := s.db.SetSystemSetting("agent_api_keys", string(raw)); err != nil {
		return err
	}
	s.keys = state
	return nil
}

func (s *Server) currentAPIKey() string {
	s.keyMu.RLock()
	defer s.keyMu.RUnlock()
	if s.keys.Current == "" {
		return s.cfg.APIKey
	}
	return s.keys.Current
}

func (s *Server) acceptsAPIKey(key string) bool {
	s.keyMu.RLock()
	defer s.keyMu.RUnlock()
	if key == "" {
		return false
	}
	if s.keys.Current == "" {
		return key == s.cfg.APIKey
	}
	if key == s.keys.Current {
		return true
	}
	for _, old := range s.keys.Old {
		if key == old {
			return true
		}
	}
	return false
}

func (s *Server) recordDeviceKey(id, key string) error {
	s.keyMu.Lock()
	defer s.keyMu.Unlock()
	accepted := key != "" && key == s.keys.Current
	for _, old := range s.keys.Old {
		accepted = accepted || key == old
	}
	if !accepted {
		return fmt.Errorf("key no longer accepted")
	}
	if s.keys.Verified[id] == keyID(key) {
		return nil
	}
	state := s.keys
	state.Verified = make(map[string]string, len(s.keys.Verified)+1)
	for deviceID, fingerprint := range s.keys.Verified {
		state.Verified[deviceID] = fingerprint
	}
	state.Verified[id] = keyID(key)
	return s.saveAPIKeysLocked(state)
}

func (s *Server) deviceKeyStatus(id string) string {
	s.keyMu.RLock()
	defer s.keyMu.RUnlock()
	fingerprint := s.keys.Verified[id]
	if fingerprint == "" {
		return "unknown"
	}
	if fingerprint == keyID(s.keys.Current) {
		return "current"
	}
	return "old"
}

func validAPIKey(key string) bool {
	if len(key) < 16 || len(key) > 512 {
		return false
	}
	return !strings.ContainsFunc(key, unicode.IsControl)
}

func (s *Server) handleKeyRotation(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims == nil || claims.Role != "admin" {
		jsonError(w, "forbidden", 403)
		return
	}
	if r.Method == http.MethodGet {
		s.keyMu.RLock()
		count := len(s.keys.Old)
		s.keyMu.RUnlock()
		jsonResp(w, map[string]interface{}{"old_key_count": count, "minimum_agent_version": "0.2.62"}, 200)
		return
	}
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", 405)
		return
	}
	var req struct {
		Operation string `json:"operation"`
		APIKey    string `json:"api_key"`
		DeviceID  string `json:"device_id"`
		ServerURL string `json:"server_url"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
		jsonError(w, "invalid request body", 400)
		return
	}
	switch req.Operation {
	case "prepare", "retain":
		valid := validAPIKey(req.APIKey)
		validationMessage := "Key baru harus 16-512 byte tanpa karakter kontrol; untuk memulihkan key pendek gunakan Terima key lama tambahan"
		if req.Operation == "retain" {
			valid = strings.TrimSpace(req.APIKey) != "" && len(req.APIKey) <= 512 && !strings.ContainsFunc(req.APIKey, unicode.IsControl)
			validationMessage = "Key pemulihan wajib diisi, maksimal 512 byte, tanpa karakter kontrol"
		}
		if !valid {
			jsonError(w, validationMessage, 400)
			return
		}
		s.keyMu.Lock()
		state := s.keys
		if state.Current == "" {
			state.Current = s.cfg.APIKey
		}
		duplicate := req.APIKey == state.Current
		for _, old := range state.Old {
			duplicate = duplicate || old == req.APIKey
		}
		if duplicate {
			s.keyMu.Unlock()
			jsonError(w, "Key sudah terdaftar", 409)
			return
		}
		state.Old = append([]string(nil), state.Old...)
		if req.Operation == "prepare" {
			state.Old = append(state.Old, state.Current)
			state.Current = req.APIKey
		} else {
			state.Old = append(state.Old, req.APIKey)
		}
		err := s.saveAPIKeysLocked(state)
		s.keyMu.Unlock()
		if err != nil {
			jsonError(w, "Gagal menyimpan key; tidak ada perubahan", 500)
			return
		}
		jsonResp(w, map[string]string{"status": "saved_without_broadcast"}, 200)
	case "migrate":
		if req.DeviceID == "" || req.APIKey != "" || req.ServerURL != "" {
			jsonError(w, "Pilih satu device; gunakan key yang sudah disiapkan", 400)
			return
		}
		device, err := s.db.GetDevice(req.DeviceID)
		if err != nil || device == nil {
			jsonError(w, "Device tidak ditemukan", 404)
			return
		}
		if !s.hub.IsOnline(req.DeviceID) {
			jsonError(w, "Device offline; migrasi belum dikirim", 409)
			return
		}
		if strings.TrimPrefix(device.Version, "v") != "0.2.62" && !versioncmp.IsNewer(device.Version, "0.2.62") {
			jsonError(w, "Update agent ke v0.2.62 atau lebih baru sebelum migrasi key", 409)
			return
		}
		raw, _ := json.Marshal(map[string]interface{}{"action": "reconfigure", "data": map[string]string{"api_key": s.currentAPIKey()}})
		if !s.hub.SendToAgent(req.DeviceID, raw) {
			jsonError(w, "Antrean penuh atau device offline; coba lagi", 409)
			return
		}
		jsonResp(w, map[string]string{"status": "migration_sent"}, 202)
	case "migrate_all":
		if req.DeviceID != "" || req.APIKey != "" || req.ServerURL != "" {
			jsonError(w, "Migrasi massal hanya memakai key aktif yang sudah disiapkan", 400)
			return
		}
		devices, total, err := s.db.ListDevices("", "", 100000, 0)
		if err != nil || total > len(devices) {
			jsonError(w, "Gagal membaca seluruh perangkat; tidak ada perintah dikirim", 500)
			return
		}
		s.keyMu.RLock()
		defer s.keyMu.RUnlock()
		fingerprint := keyID(s.keys.Current)
		pilotVerified := false
		for _, device := range devices {
			if s.keys.Verified[device.ID] == fingerprint && s.hub.IsOnline(device.ID) {
				pilotVerified = true
				break
			}
		}
		if !pilotVerified || s.keys.Current == "" {
			jsonError(w, "Verifikasi satu PC pilot dengan key aktif dan pastikan online sebelum migrasi massal", 409)
			return
		}
		counts := map[string]int{"sent": 0, "current": 0, "offline": 0, "update_required": 0, "failed": 0}
		raw, _ := json.Marshal(map[string]interface{}{"action": "reconfigure", "data": map[string]string{"api_key": s.keys.Current}})
		for _, device := range devices {
			switch {
			case s.keys.Verified[device.ID] == fingerprint:
				counts["current"]++
			case !s.hub.IsOnline(device.ID):
				counts["offline"]++
			case strings.TrimPrefix(device.Version, "v") != "0.2.62" && !versioncmp.IsNewer(device.Version, "0.2.62"):
				counts["update_required"]++
			case s.hub.SendToAgent(device.ID, raw):
				counts["sent"]++
			default:
				counts["failed"]++
			}
		}
		jsonResp(w, map[string]interface{}{"status": "bulk_migration_sent", "counts": counts}, 202)
	case "revoke":
		devices, total, err := s.db.ListDevices("", "", 100000, 0)
		if err != nil || total > len(devices) {
			jsonError(w, "Gagal memverifikasi seluruh perangkat", 500)
			return
		}
		s.keyMu.Lock()
		for _, device := range devices {
			if s.keys.Verified[device.ID] != keyID(s.keys.Current) {
				s.keyMu.Unlock()
				jsonError(w, "Masih ada perangkat belum terverifikasi dengan key terbaru", 409)
				return
			}
		}
		state := s.keys
		state.Old = nil
		err = s.saveAPIKeysLocked(state)
		s.keyMu.Unlock()
		if err != nil {
			jsonError(w, "Gagal mencabut key lama", 500)
			return
		}
		jsonResp(w, map[string]string{"status": "old_keys_revoked"}, 200)
	case "endpoint":
		endpoint, err := url.Parse(req.ServerURL)
		if err != nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || req.APIKey != "" || req.DeviceID == "" {
			jsonError(w, "Endpoint HTTP(S) dan satu device wajib diisi; key tidak boleh diganti bersamaan", 400)
			return
		}
		raw, _ := json.Marshal(map[string]interface{}{"action": "reconfigure", "data": map[string]string{"server_url": req.ServerURL}})
		if !s.hub.SendToAgent(req.DeviceID, raw) {
			jsonError(w, "Device offline atau antrean penuh", 409)
			return
		}
		jsonResp(w, map[string]string{"status": "endpoint_sent"}, 202)
	default:
		jsonError(w, "Pilih operasi prepare, retain, migrate, revoke, atau endpoint; broadcast dinonaktifkan", 400)
	}
}
