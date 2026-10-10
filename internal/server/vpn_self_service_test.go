package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
)

func saveSelfPolicy(t *testing.T, server *Server, device, role, body string, code int) {
	t.Helper()
	request := httptest.NewRequest("PUT", "/api/vpn/self-service?device="+device, strings.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), userClaimsKey, &UserClaims{Username: role, Role: role}))
	response := httptest.NewRecorder()
	server.handleVPNSelfPolicy(response, request)
	if response.Code != code {
		t.Fatalf("policy: %d %s", response.Code, response.Body.String())
	}
}

func selfConnectReply(t *testing.T, server *Server, device string) vpn.SelfReply {
	t.Helper()
	server.receiveVPNSelfRequest(device, "connect", json.RawMessage(`{"id":"0123456789abcdef0123456789abcdef"}`))
	for {
		select {
		case data := <-server.hub.agents[device].Send:
			var message struct {
				Action string        `json:"action"`
				Data   vpn.SelfReply `json:"data"`
			}
			if err := json.Unmarshal(data, &message); err != nil {
				t.Fatal(err)
			}
			if message.Action == "vpn_self_result" {
				return message.Data
			}
		default:
			t.Fatal("self-service result missing")
			return vpn.SelfReply{}
		}
	}
}

func TestVPNSelfServicePolicyAndRevocation(t *testing.T) {
	server, commands := lanServer(t)
	if reply := selfConnectReply(t, server, "client"); reply.Accepted || len(server.vpnPilot.sessions) != 0 {
		t.Fatal("default deny bypassed")
	}
	saveSelfPolicy(t, server, "client", "viewer", `{"enabled":true}`, 403)
	if server.vpnSelfPolicy("client").Enabled {
		t.Fatal("viewer changed policy")
	}
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true,"route_lan":"0.0.0.0/0"}`, 400)
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true,"advertise_lan":"192.168.1.0/24"}`, 400)
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true} {}`, 400)
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true}`, 200)
	other := &Server{db: server.db}
	if !other.vpnSelfPolicy("client").Enabled {
		t.Fatal("policy not persisted")
	}
	delete(server.vpnPilot.selfRequests, "client")
	if reply := selfConnectReply(t, server, "client"); !reply.Accepted {
		t.Fatal(reply.Detail)
	}
	session := server.vpnPilot.sessions["client"]
	if !session.selfService || session.config.AdvertiseLAN != "" || len(session.config.Routes) != 0 {
		t.Fatal("unexpected self-service config")
	}
	_, public, _ := vpn.NewKey()
	ready, _ := json.Marshal(vpn.Status{ID: session.status.ID, State: "ready", PublicKey: public})
	server.receiveVPN("client", ready)
	if !session.provisioned {
		t.Fatal("approved session not provisioned")
	}
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":false}`, 200)
	if session.provisioned || session.status.State != "disconnecting" {
		t.Fatal("revoked session still active")
	}
	if !strings.Contains(strings.Join(*commands, "\n"), "peer "+public+" remove") {
		t.Fatal("hub peer was not removed")
	}
	delete(server.vpnPilot.selfRequests, "client")
	if reply := selfConnectReply(t, server, "client"); reply.Accepted {
		t.Fatal("revoked policy reconnected")
	}
}

func TestVPNSelfServiceRejectsCallerConfigAndLANEscalation(t *testing.T) {
	server, _ := lanServer(t)
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true,"route_lan":"192.168.1.0/24"}`, 200)
	for _, extra := range []string{`"device":"gateway"`, `"route_lan":"10.0.0.0/16"`, `"advertise_lan":"192.168.1.0/24"`, `"enabled":true`} {
		server.receiveVPNSelfRequest("client", "connect", json.RawMessage(`{"id":"0123456789abcdef0123456789abcdef",`+extra+`}`))
		if len(server.vpnPilot.sessions) != 0 {
			t.Fatal("caller configuration accepted")
		}
	}
	if reply := selfConnectReply(t, server, "client"); reply.Accepted {
		t.Fatal("missing LAN gateway allowed")
	}
	delete(server.vpnPilot.selfRequests, "client")
	gateway := lanSession(server, "10.77.0.20")
	gateway.config.AdvertiseLAN = "192.168.1.0/24"
	gateway.status.AdvertiseLAN = gateway.config.AdvertiseLAN
	gateway.status.State = "connected"
	gateway.status.LANClients = []string{"denied"}
	server.vpnPilot.sessions["gateway"] = gateway
	if reply := selfConnectReply(t, server, "client"); reply.Accepted {
		t.Fatal("gateway whitelist bypassed")
	}
	delete(server.vpnPilot.selfRequests, "client")
	gateway.status.LANClients = []string{"client"}
	if reply := selfConnectReply(t, server, "client"); !reply.Accepted {
		t.Fatal(reply.Detail)
	}
	if routes := server.vpnPilot.sessions["client"].config.Routes; len(routes) != 1 || routes[0] != "192.168.1.0/24" {
		t.Fatal(routes)
	}
}

func TestVPNSelfServiceRechecksPolicyBeforeProvisioning(t *testing.T) {
	server, _ := lanServer(t)
	saveSelfPolicy(t, server, "client", "admin", `{"enabled":true}`, 200)
	if reply := selfConnectReply(t, server, "client"); !reply.Accepted {
		t.Fatal(reply.Detail)
	}
	session := server.vpnPilot.sessions["client"]
	if err := server.db.SetSystemSetting("vpn_self_service:client", "corrupt"); err != nil {
		t.Fatal(err)
	}
	_, public, _ := vpn.NewKey()
	raw, _ := json.Marshal(vpn.Status{ID: session.status.ID, State: "ready", PublicKey: public})
	server.receiveVPN("client", raw)
	if session.provisioned || session.status.State != "disconnecting" {
		t.Fatal("policy not rechecked")
	}
	server.receiveVPNSelfRequest("client", "disconnect", json.RawMessage(`{"id":"0123456789abcdef0123456789abcdef"}`))
	if session.status.State != "disconnecting" {
		t.Fatal("disconnect requires connect permission")
	}
	server.vpnPilot.selfRequests["denied"] = time.Now()
	if reply := selfConnectReply(t, server, "denied"); reply.Accepted || !strings.Contains(reply.Detail, "5 detik") {
		t.Fatal("request throttle missing")
	}
}
