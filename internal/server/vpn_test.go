package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
	"github.com/user/remote-desktop/internal/vpn"
)

func TestDeviceListVPNStatusScopeAndFreshness(t *testing.T) {
	server := securityServer(t)
	server.hub = NewHub(server.db)
	for _, id := range []string{"visible", "hidden"} {
		if err := server.db.UpsertDevice(&models.Device{ID: id, Hostname: id, GroupName: id, OS: "windows"}); err != nil {
			t.Fatal(err)
		}
	}
	server.hub.agents["visible"] = &Client{Send: make(chan []byte, 1)}
	session := &vpnSession{status: vpn.Status{State: "connected", Address: "10.77.0.2", PublicKey: "must-not-leak", Updated: time.Now().Unix()}}
	server.vpnPilot = &vpnServer{sessions: map[string]*vpnSession{"visible": session, "hidden": session}}
	read := func() map[string]map[string]string {
		request := httptest.NewRequest("GET", "/api/devices?group=visible", nil)
		request = request.WithContext(context.WithValue(request.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
		response := httptest.NewRecorder()
		server.handleDevices(response, request)
		if response.Code != 200 {
			t.Fatalf("response: %d %s", response.Code, response.Body.String())
		}
		var result struct {
			Statuses map[string]map[string]string `json:"vpn_statuses"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if _, leaked := result.Statuses["hidden"]; leaked || strings.Contains(response.Body.String(), "must-not-leak") {
			t.Fatal("VPN metadata escaped device scope")
		}
		return result.Statuses
	}
	if status := read()["visible"]; status["state"] != "connected" || status["address"] != "10.77.0.2" {
		t.Fatalf("missing VPN status: %v", status)
	}
	session.status.Updated = time.Now().Add(-vpn.Lease - time.Second).Unix()
	if read()["visible"]["state"] != "unknown" {
		t.Fatal("stale session shown as connected")
	}
	session.status.Updated = time.Now().Unix()
	delete(server.hub.agents, "visible")
	if read()["visible"]["state"] != "unknown" {
		t.Fatal("offline agent shown as connected")
	}
	delete(server.vpnPilot.sessions, "visible")
	if read()["visible"]["state"] != "disconnected" {
		t.Fatal("missing session not shown as off")
	}
	server.vpnPilot = nil
	if len(read()) != 0 {
		t.Fatal("unavailable VPN returned known status")
	}
}

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

func TestVPNHandshakeRecoveryGetsFreshTimeout(t *testing.T) {
	now := time.Now()
	session := &vpnSession{started: now.Add(-6 * time.Minute), connectingAt: now.Add(-6 * time.Minute), provisioned: true, status: vpn.Status{ID: strings.Repeat("a", 32), State: "connecting", PublicKey: "peer"}}
	hub := NewHub(nil)
	hub.agents["device"] = &Client{Send: make(chan []byte, 20)}
	handshake := now.Unix()
	removed := false
	server := &Server{hub: hub, vpnPilot: &vpnServer{sessions: map[string]*vpnSession{"device": session}, run: func(_ string, args ...string) ([]byte, error) {
		if args[0] == "set" {
			removed = true
			return nil, nil
		}
		return []byte(fmt.Sprintf("peer\t%d\n", handshake)), nil
	}}}
	report, _ := json.Marshal(vpn.Status{ID: session.status.ID, State: "running"})
	for attempt := 0; attempt < 2; attempt++ {
		handshake = time.Now().Unix()
		server.receiveVPN("device", report)
		if session.status.State != "connected" {
			t.Fatalf("healthy handshake rejected: %+v", session.status)
		}
		handshake = now.Add(-301 * time.Second).Unix()
		server.receiveVPN("device", report)
		if removed || session.status.State != "connecting" || time.Since(session.connectingAt) > time.Second {
			t.Fatalf("recovery reused initial connection deadline: %+v", session)
		}
		deadlineStart := session.connectingAt
		server.receiveVPN("device", report)
		if session.connectingAt != deadlineStart {
			t.Fatal("repeated failed handshake extended recovery deadline")
		}
	}
	session.connectingAt = time.Now().Add(-61 * time.Second)
	server.receiveVPN("device", report)
	if !removed || session.provisioned || session.status.State != "disconnecting" {
		t.Fatalf("expired recovery did not revoke peer: %+v", session)
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

func TestVPNCleanupPreservesInitialAgentFailure(t *testing.T) {
	session := &vpnSession{provisioned: true, status: vpn.Status{ID: "session", State: "connected"}}
	server := &Server{hub: NewHub(nil), vpnPilot: &vpnServer{sessions: map[string]*vpnSession{"device": session}, run: func(string, ...string) ([]byte, error) { return nil, nil }}}
	for _, report := range []vpn.Status{
		{ID: "session", State: "error", Detail: "penyimpanan lease gagal: sharing violation"},
		{ID: "session", State: "disconnected", Detail: "VPN nonaktif; service RemoteDesk tetap berjalan"},
	} {
		raw, _ := json.Marshal(report)
		server.receiveVPN("device", raw)
		if !strings.Contains(session.status.Detail, "sharing violation") {
			t.Fatalf("cleanup erased initial failure: %+v", session.status)
		}
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
