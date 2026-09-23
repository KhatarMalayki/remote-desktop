package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := getClaims(r).Username
	key := "mfa-manage:" + username
	if !s.checkLoginRateLimit(key) {
		jsonError(w, "Terlalu banyak percobaan; coba lagi nanti", 429)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req) != nil {
		jsonError(w, "invalid body", 400)
		return
	}
	enabled, secret, err := s.db.GetUserMFA(username)
	if err != nil || !enabled || !ValidateTOTPCode(secret, req.Code) {
		recordLoginFail(key)
		jsonError(w, "Kode MFA salah atau kedaluwarsa", 403)
		return
	}
	recordLoginSuccess(key)
	jsonResp(w, map[string]string{"management_ticket": s.issueCredentialToken(username, "mfa-manage")}, 200)
}

func (s *Server) requireMFAManagement(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	claims, kind, enabled, mustChange, ok := s.readCredentialToken(r.Header.Get("X-MFA-Management"))
	if !ok || kind != "mfa-manage" || !enabled || mustChange || claims.Username != getClaims(r).Username {
		jsonError(w, "Verifikasi ulang MFA untuk mengelola authenticator", 403)
		return false
	}
	return true
}
