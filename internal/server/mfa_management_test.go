package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMFAManagementRequiresReauthentication(t *testing.T) {
	s := securityServer(t)
	secret := GenerateTOTPSecret()
	if err := s.db.CreateUser("manage", hashPassword("unique management phrase 4938"), "admin", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.db.Exec(`UPDATE users SET must_change_password=0, mfa_enabled=1, mfa_secret=? WHERE username='manage'`, secret); err != nil {
		t.Fatal(err)
	}
	token := s.issueCredentialToken("manage", "session")
	if err := s.db.TrustDevice("manage", s.trustedDeviceDigest("manage", "browser"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	request := func(path, ticket string, body interface{}, handler http.HandlerFunc, status int) map[string]interface{} {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", path, strings.NewReader(string(payload)))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-MFA-Management", ticket)
		req.AddCookie(&http.Cookie{Name: "rd_trusted_device", Value: "browser"})
		response := httptest.NewRecorder()
		s.authMiddleware(handler)(response, req)
		if response.Code != status {
			t.Fatalf("%s: got %d want %d: %s", path, response.Code, status, response.Body.String())
		}
		result := map[string]interface{}{}
		_ = json.Unmarshal(response.Body.Bytes(), &result)
		return result
	}
	request("/api/auth/mfa/setup", "", nil, s.handleMFASetup, 403)
	request("/api/auth/mfa/setup", token, nil, s.handleMFASetup, 403)
	request("/api/auth/mfa/verify", "", map[string]string{"code": "bad"}, s.handleMFAVerify, 403)
	code, err := GenerateTOTPCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	verified := request("/api/auth/mfa/verify", "", map[string]string{"code": code}, s.handleMFAVerify, 200)
	ticket := verified["management_ticket"].(string)
	authRequest(t, s, "/api/auth/me", ticket, nil, s.authMiddleware(s.handleMe), 401)
	setup := request("/api/auth/mfa/setup", ticket, nil, s.handleMFASetup, 200)
	pending := setup["secret"].(string)
	if pending == secret {
		t.Fatal("old secret disclosed")
	}
	enabled, active, err := s.db.GetUserMFA("manage")
	if err != nil || !enabled || active != secret {
		t.Fatal("setup disabled old MFA")
	}
	request("/api/auth/mfa/enable", "", map[string]string{"code": code}, s.handleMFAEnable, 403)
	request("/api/auth/mfa/enable", ticket, map[string]string{"code": "bad"}, s.handleMFAEnable, 400)
	_, active, _ = s.db.GetUserMFA("manage")
	if active != secret {
		t.Fatal("invalid confirmation replaced MFA")
	}
	newCode, err := GenerateTOTPCode(pending, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	result := request("/api/auth/mfa/enable", ticket, map[string]string{"code": newCode}, s.handleMFAEnable, 200)
	authRequest(t, s, "/api/auth/me", token, nil, s.authMiddleware(s.handleMe), 401)
	if s.db.IsTrustedDevice("manage", s.trustedDeviceDigest("manage", "browser")) {
		t.Fatal("trusted browser survived rotation")
	}
	token = result["token"].(string)
	request("/api/auth/mfa/setup", ticket, nil, s.handleMFASetup, 403)
	request("/api/auth/me", "", nil, s.handleMe, 200)
}

func TestAdminCannotBypassMFAWithTrustedDevice(t *testing.T) {
	s := securityServer(t)
	secret := GenerateTOTPSecret()
	pass := "super secure admin phrase 99911"
	if err := s.db.CreateUser("superadmin", hashPassword(pass), "admin", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.db.Exec("UPDATE users SET must_change_password=0, mfa_enabled=1, mfa_secret=? WHERE username='superadmin'", secret); err != nil {
		t.Fatal(err)
	}
	cookieVal := "browser-cookie-token"
	if err := s.db.TrustDevice("superadmin", s.trustedDeviceDigest("superadmin", cookieVal), time.Now().Add(14*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"superadmin","password":"`+pass+`"}`))
	req.AddCookie(&http.Cookie{Name: "rd_trusted_device", Value: cookieVal})
	rec := httptest.NewRecorder()
	s.handleLogin(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200 got %d: %s", rec.Code, rec.Body.String())
	}
	var res map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res["mfa_required"] != true {
		t.Fatalf("expected admin to require MFA every login, got: %v", res)
	}
}
