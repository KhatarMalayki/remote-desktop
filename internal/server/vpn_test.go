package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVPNPolicyRequiresAdminAndValidation(t *testing.T) {
	s := securityServer(t)
	s.initVPN()
	req := httptest.NewRequest("GET", "/api/vpn/pilot", nil)
	w := httptest.NewRecorder()
	s.handleVPN(w, req)
	if w.Code != 403 {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	claims := &UserClaims{Username: "operator", Role: "viewer"}
	req = httptest.NewRequest("POST", "/api/vpn/pilot", strings.NewReader(`{"device":"d1","operation":"connect"}`))
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, claims))
	w = httptest.NewRecorder()
	s.handleVPN(w, req)
	if w.Code != 403 {
		t.Fatalf("expected viewer 403, got %d", w.Code)
	}

	admin := &UserClaims{Username: "admin", Role: "admin"}
	req = httptest.NewRequest("POST", "/api/vpn/pilot", strings.NewReader(`{"device":"d1","operation":"unknown"}`))
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, admin))
	w = httptest.NewRecorder()
	s.handleVPN(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid operation, got %d", w.Code)
	}
}
