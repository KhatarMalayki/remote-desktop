package server

import (
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const passwordIterations = 600000

// Password hashing is separate from deterministic digests of random device tokens.
func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	if err != nil {
		panic(err)
	}
	return "pbkdf2-sha256$600000$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(key)
}

func verifyPassword(encoded, password string) bool {
	if len(password) > 4096 {
		return false
	}
	if !strings.HasPrefix(encoded, "pbkdf2-sha256$") {
		// Compatibility only: successful legacy login immediately upgrades this hash.
		return len(encoded) == 64 && subtle.ConstantTimeCompare([]byte(encoded), []byte(tokenDigest(password))) == 1
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[1] != "600000" {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := hex.DecodeString(parts[3])
	if err != nil || len(expected) != 32 {
		return false
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, 32)
	return err == nil && subtle.ConstantTimeCompare(key, expected) == 1
}

func validatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if !utf8.ValidString(password) || n < 15 || n > 128 || strings.ContainsAny(password, "\r\n\x00") {
		return fmt.Errorf("Password harus 15–128 karakter; boleh memakai spasi, tanpa baris baru")
	}
	normalized := strings.ToLower(strings.TrimSpace(password))
	// Local common-password blocklist. This is not a complete breached-password database.
	for _, blocked := range []string{"passwordpassword", "password123456789", "123456789012345", "1234567890123456", "qwertyuiopasdfgh", "qwertyuiopasdfghjkl", "adminadminadmin", "letmeinletmeinletmein", "iloveyouiloveyou", "changemechangeme", "remotedeskremotedesk"} {
		if normalized == blocked {
			return fmt.Errorf("Password terlalu umum; gunakan frasa unik")
		}
	}
	if normalized == "" || len(strings.Trim(normalized, string([]rune(normalized)[0]))) == 0 {
		return fmt.Errorf("Password terlalu mudah ditebak")
	}
	return nil
}

type credentialToken struct {
	Username string `json:"u"`
	Kind     string `json:"k"`
	State    string `json:"s"`
	Expires  int64  `json:"e"`
}

func (s *Server) accountState(username string) (*UserClaims, string, bool, bool, error) {
	var hash, role, branch, secret string
	var enabled, mustChange bool
	err := s.db.db.QueryRow(`SELECT password_hash, role, branch, mfa_secret, mfa_enabled, must_change_password FROM users WHERE username=?`, username).Scan(&hash, &role, &branch, &secret, &enabled, &mustChange)
	if err != nil {
		return nil, "", false, false, err
	}
	raw, _ := json.Marshal([]interface{}{username, hash, role, branch, secret, enabled, mustChange})
	return &UserClaims{Username: username, Role: role, Branch: branch}, tokenDigest(string(raw)), enabled, mustChange, nil
}

func (s *Server) issueCredentialToken(username, kind string) string {
	_, state, enabled, _, err := s.accountState(username)
	if err != nil {
		return ""
	}
	if kind == "session" && !enabled {
		kind = "enrollment"
	}
	ttl := 24 * time.Hour
	if kind != "session" {
		ttl = 15 * time.Minute
	}
	payload, _ := json.Marshal(credentialToken{username, kind, state, time.Now().Add(ttl).Unix()})
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + hex.EncodeToString(mac.Sum(nil))
}

func (s *Server) readCredentialToken(token string) (*UserClaims, string, bool, bool, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, "", false, false, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, "", false, false, false
	}
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, "", false, false, false
	}
	mac := hmac.New(sha256.New, []byte(s.cfg.JWTSecret))
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, "", false, false, false
	}
	var tok credentialToken
	if json.Unmarshal(payload, &tok) != nil || tok.Expires <= time.Now().Unix() {
		return nil, "", false, false, false
	}
	claims, state, enabled, mustChange, err := s.accountState(tok.Username)
	if err != nil || state != tok.State {
		return nil, "", false, false, false
	}
	return claims, tok.Kind, enabled, mustChange, true
}

func (s *Server) authorizeUserToken(w http.ResponseWriter, r *http.Request, token string) (*UserClaims, bool) {
	claims, kind, enabled, mustChange, ok := s.readCredentialToken(token)
	if !ok || (kind != "session" && kind != "enrollment") {
		jsonError(w, "Silakan login ulang", 401)
		return nil, false
	}
	path := r.URL.Path
	allowed := path == "/api/auth/me" || path == "/api/auth/change-password" || path == "/api/auth/mfa/status"
	if !mustChange && !enabled && (path == "/api/auth/mfa/setup" || path == "/api/auth/mfa/enable") {
		allowed = true
	}
	if (mustChange || !enabled || kind == "enrollment") && !allowed {
		jsonError(w, "Selesaikan ganti password dan aktivasi MFA terlebih dahulu", 403)
		return nil, false
	}
	return claims, true
}

func (s *Server) writeSession(w http.ResponseWriter, username string) {
	claims, _, _, mustChange, err := s.accountState(username)
	if err != nil {
		jsonError(w, "Akun tidak ditemukan", 401)
		return
	}
	jsonResp(w, map[string]interface{}{"token": s.issueCredentialToken(username, "session"), "username": username, "role": claims.Role, "branch": claims.Branch, "must_change_password": mustChange}, 200)
}

func (s *Server) trustedDeviceDigest(username, value string) string {
	_, state, _, _, err := s.accountState(username)
	if err != nil {
		return ""
	}
	return tokenDigest(value + ":" + state)
}
