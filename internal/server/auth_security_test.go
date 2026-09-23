package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func securityServer(t *testing.T) *Server {
	t.Helper()
	db, err := NewDB(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Server{db: db, cfg: Config{JWTSecret: "security-test-secret", APIKey: "agent-test-key"}}
}

func authRequest(t *testing.T, s *Server, path, token string, body interface{}, handler http.HandlerFunc, status int) map[string]interface{} {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", path, strings.NewReader(string(payload)))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler(w, req)
	if w.Code != status {
		t.Fatalf("%s: got %d want %d: %s", path, w.Code, status, w.Body.String())
	}
	result := map[string]interface{}{}
	_ = json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func TestPasswordHashAndPolicy(t *testing.T) {
	pass := "empat kata unik 4938"
	hash := hashPassword(pass)
	if hash == hashPassword(pass) {
		t.Fatal("password salts are not random")
	}
	if !verifyPassword(hash, pass) || verifyPassword(hash, "wrong") {
		t.Fatal("password verification failed")
	}
	if !verifyPassword(tokenDigest(pass), pass) {
		t.Fatal("legacy migration cannot verify password")
	}
	for _, invalid := range []string{"short", "123456789012345", "passwordpassword", strings.Repeat("a", 16), strings.Repeat(" ", 15), strings.Repeat("x", 129), "long password\nwith newline"} {
		if validatePassword(invalid) == nil {
			t.Fatalf("accepted weak password %q", invalid)
		}
	}
	for _, valid := range []string{pass, "  ruang kata aman 4938  ", strings.Repeat("aB", 64), strings.Repeat("界", 14) + "語"} {
		if err := validatePassword(valid); err != nil {
			t.Fatalf("valid password rejected: %v", err)
		}
	}
	for _, invalid := range []string{"pbkdf2-sha256$999999999$aa$bb", "pbkdf2-sha256$600000$no$no", "not-a-hash"} {
		if verifyPassword(invalid, pass) {
			t.Fatal("invalid hash accepted")
		}
	}
}

func TestMandatoryOnboardingAndSessionRevocation(t *testing.T) {
	s := securityServer(t)
	initial := "initial unique phrase 4839"
	changed := "my own unique phrase 7492"
	if err := s.db.CreateUser("newuser", hashPassword(initial), "admin", ""); err != nil {
		t.Fatal(err)
	}
	login := func(password string) map[string]interface{} {
		return authRequest(t, s, "/api/auth/login", "", map[string]string{"username": "newuser", "password": password}, s.handleLogin, 200)
	}
	token := login(initial)["token"].(string)
	protected := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, path := range []string{"/api/devices", "/api/users", "/ws/viewer", "/api/auth/change-username"} {
		authRequest(t, s, path, token, nil, protected, 403)
	}
	authRequest(t, s, "/api/agent/download", token, nil, s.handleAgentDownload, 403)
	// Relay has its own authentication entrypoint.
	w := httptest.NewRecorder()
	s.handleRelayWS(w, httptest.NewRequest("GET", "/ws/relay/session/viewer?device_id=pc&token="+token, nil))
	if w.Code != 403 {
		t.Fatalf("relay bypass: %d", w.Code)
	}
	req := httptest.NewRequest("GET", "/ws/viewer?token="+token, nil)
	req.Header.Set("Upgrade", "websocket")
	w = httptest.NewRecorder()
	protected(w, req)
	if w.Code != 403 {
		t.Fatalf("websocket bypass: %d", w.Code)
	}
	authRequest(t, s, "/api/auth/mfa/setup", token, nil, s.authMiddleware(s.handleMFASetup), 403)
	oldToken := token
	result := authRequest(t, s, "/api/auth/change-password", token, map[string]string{"old_password": initial, "new_password": changed, "confirm_password": changed}, s.authMiddleware(s.handleChangePassword), 200)
	token = result["token"].(string)
	authRequest(t, s, "/api/auth/me", oldToken, nil, s.authMiddleware(s.handleMe), 401)
	authRequest(t, s, "/api/devices", token, nil, protected, 403)
	setup := authRequest(t, s, "/api/auth/mfa/setup", token, nil, s.authMiddleware(s.handleMFASetup), 200)
	secret := setup["secret"].(string)
	code, err := GenerateTOTPCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	oldToken = token
	result = authRequest(t, s, "/api/auth/mfa/enable", token, map[string]string{"secret": "untrusted client value", "code": code}, s.authMiddleware(s.handleMFAEnable), 200)
	token = result["token"].(string)
	authRequest(t, s, "/api/devices", oldToken, nil, protected, 401)
	authRequest(t, s, "/api/devices", token, nil, protected, 204)
	authRequest(t, s, "/api/auth/mfa/disable", token, nil, s.authMiddleware(s.handleMFADisable), 403)
	authRequest(t, s, "/api/auth/mfa/setup", token, nil, s.authMiddleware(s.handleMFASetup), 403)
	authRequest(t, s, "/api/auth/mfa/enable", token, map[string]string{"secret": secret, "code": code}, s.authMiddleware(s.handleMFAEnable), 403)
	challenge := login(changed)["mfa_ticket"].(string)
	authRequest(t, s, "/api/devices", challenge, nil, protected, 401)
	verified := authRequest(t, s, "/api/auth/login/mfa", "", map[string]interface{}{"mfa_ticket": challenge, "code": code, "remember_device": true}, s.handleLoginMFA, 200)
	token = verified["token"].(string)
	authRequest(t, s, "/api/devices", token, nil, protected, 204)
	// Reset invalidates sessions, challenge tickets, and trusted devices.
	trusted := s.trustedDeviceDigest("newuser", "random-device-value")
	if err := s.db.TrustDevice("newuser", trusted, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	id, _, _, _, _ := s.db.GetUser("newuser")
	if err := s.db.ResetUserMFA(int64(id)); err != nil {
		t.Fatal(err)
	}
	authRequest(t, s, "/api/devices", token, nil, protected, 401)
	authRequest(t, s, "/api/auth/login/mfa", "", map[string]string{"mfa_ticket": challenge, "code": code}, s.handleLoginMFA, 401)
	if s.db.IsTrustedDevice("newuser", s.trustedDeviceDigest("newuser", "random-device-value")) {
		t.Fatal("trusted device survived reset")
	}
	token = login(changed)["token"].(string)
	authRequest(t, s, "/api/devices", token, nil, protected, 403)
}

func TestLegacyLoginUpgradesHashAndRequiresStrongPassword(t *testing.T) {
	s := securityServer(t)
	if err := s.db.CreateUser("legacy", tokenDigest("oldpass"), "admin", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.db.Exec(`UPDATE users SET must_change_password=0 WHERE username='legacy'`); err != nil {
		t.Fatal(err)
	}
	result := authRequest(t, s, "/api/auth/login", "", map[string]string{"username": "legacy", "password": "oldpass"}, s.handleLogin, 200)
	if result["must_change_password"] != true {
		t.Fatal("weak legacy password not gated")
	}
	_, hash, _, _, err := s.db.GetUser("legacy")
	if err != nil || !strings.HasPrefix(hash, "pbkdf2-sha256$") || !verifyPassword(hash, "oldpass") {
		t.Fatal("legacy hash not upgraded")
	}
}

func TestPasswordPolicyAcrossManagementRoutes(t *testing.T) {
	s := securityServer(t)
	if err := s.db.CreateUser("target", hashPassword("initial unique phrase 4839"), "viewer", ""); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, method string
		body         map[string]string
		handler      http.HandlerFunc
	}{
		{"/api/users", "POST", map[string]string{"username": "another", "password": "short", "role": "viewer"}, s.handleUsers},
		{"/api/users/1", "PUT", map[string]string{"username": "target", "password": "short", "role": "viewer"}, s.handleUserUpdate},
		{"/api/users/1/reset-password", "POST", map[string]string{"new_password": "short"}, s.handleResetUserPassword},
	} {
		payload, _ := json.Marshal(tc.body)
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(string(payload)))
		req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
		w := httptest.NewRecorder()
		s.authMiddleware(tc.handler)(w, req)
		if w.Code != 400 {
			t.Fatalf("%s accepted weak password: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
	if err := s.db.UpdatePassword("target", hashPassword("my own unique phrase 7492")); err != nil {
		t.Fatal(err)
	}
	if err := s.db.UpdateUser(1, "target", "admin changed phrase 492", "viewer", ""); err != nil {
		t.Fatal(err)
	}
	_, _, _, mustChange, err := s.accountState("target")
	if err != nil || !mustChange {
		t.Fatal("admin edit did not force password change")
	}
	if err := s.db.UpdatePassword("target", hashPassword("my own unique phrase 7492")); err != nil {
		t.Fatal(err)
	}
	if err := s.db.ResetUserPassword(1, hashPassword("admin reset phrase 3948")); err != nil {
		t.Fatal(err)
	}
	_, _, _, mustChange, err = s.accountState("target")
	if err != nil || !mustChange {
		t.Fatal("admin reset did not force password change")
	}
}

func TestCredentialTokenIntegrityAndAccountChanges(t *testing.T) {
	s := securityServer(t)
	if err := s.db.CreateUser("account", hashPassword("unique phrase for tests 3948"), "admin", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.db.SetUserMFA("account", GenerateTOTPSecret(), true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.db.Exec(`UPDATE users SET must_change_password=0 WHERE username='account'`); err != nil {
		t.Fatal(err)
	}
	token := s.issueCredentialToken("account", "session")
	protected := s.authMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	authRequest(t, s, "/api/devices", token, nil, protected, 204)
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + strings.Repeat("0", 64)
	authRequest(t, s, "/api/devices", tampered, nil, protected, 401)
	if _, err := s.db.db.Exec(`UPDATE users SET role='viewer' WHERE username='account'`); err != nil {
		t.Fatal(err)
	}
	authRequest(t, s, "/api/devices", token, nil, protected, 401)
	token = s.issueCredentialToken("account", "session")
	if err := s.db.ResetUserPassword(1, hashPassword("administrator reset phrase 382")); err != nil {
		t.Fatal(err)
	}
	authRequest(t, s, "/api/devices", token, nil, protected, 401)
}
