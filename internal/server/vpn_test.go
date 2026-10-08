package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
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

func TestVPNDisconnectPreservesServerReason(t *testing.T) {
	session := &vpnSession{status: vpn.Status{ID: "session", State: "connected"}}
	server := &Server{hub: NewHub(nil), vpnPilot: &vpnServer{sessions: map[string]*vpnSession{"device": session}}}
	server.stopVPNSession("device", session, "Lease pilot habis")
	server.stopVPNSession("device", session, "Batas sesi pilot 15 menit tercapai")
	report, _ := json.Marshal(vpn.Status{ID: "session", State: "disconnected", Detail: "VPN nonaktif; service RemoteDesk tetap berjalan"})
	server.receiveVPN("device", report)
	if session.status.State != "disconnected" || !strings.Contains(session.status.Detail, "Lease pilot habis") {
		t.Fatalf("disconnect reason lost: %+v", session.status)
	}
	server.receiveVPN("device", report)
	if !strings.Contains(session.status.Detail, "Lease pilot habis") {
		t.Fatalf("repeated report erased reason: %+v", session.status)
	}
}

func TestVPNKeepalivePrecedesHandshakeAndTimeoutRevokesPeer(t *testing.T) {
	for _, validHandshake := range []bool{true, false} {
		now := time.Now()
		session := &vpnSession{started: now.Add(-time.Minute), connectingAt: now.Add(-61 * time.Second), provisioned: true, status: vpn.Status{ID: strings.Repeat("a", 32), State: "connecting", PublicKey: "peer"}}
		hub := NewHub(nil)
		client := &Client{Send: make(chan []byte, 5)}
		hub.agents["device"] = client
		removed := false
		server := &Server{hub: hub, vpnPilot: &vpnServer{sessions: map[string]*vpnSession{"device": session}}}
		server.vpnPilot.run = func(name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "set" {
				removed = true
				return nil, nil
			}
			select {
			case command := <-client.Send:
				if !strings.Contains(string(command), "keepalive") {
					t.Fatalf("expected keepalive first: %s", command)
				}
			default:
				t.Fatal("handshake inspection blocked keepalive")
			}
			if validHandshake {
				return []byte("peer\t" + fmt.Sprint(now.Unix()) + "\n"), nil
			}
			return []byte("peer\t0\n"), nil
		}
		raw, _ := json.Marshal(vpn.Status{ID: session.status.ID, State: "running"})
		server.receiveVPN("device", raw)
		if validHandshake {
			if session.status.State != "connected" || removed {
				t.Fatalf("valid handshake failed: %+v", session.status)
			}
		} else if !removed || session.provisioned || session.status.State != "disconnecting" || !strings.Contains(session.status.Detail, "timeout") {
			t.Fatalf("unverified tunnel not revoked: %+v", session.status)
		}
	}
}
