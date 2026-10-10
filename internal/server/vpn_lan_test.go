package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
	"github.com/user/remote-desktop/internal/vpn"
)

func lanServer(t *testing.T) (*Server, *[]string) {
	server := securityServer(t)
	server.hub = NewHub(server.db)
	_, public, err := vpn.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	commands := []string{}
	server.vpnPilot = &vpnServer{ready: true, network: "10.77.0.0/24", endpoint: "vpn.example.test:51820", publicKey: public, sessions: map[string]*vpnSession{}, run: func(name string, args ...string) ([]byte, error) {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "ip" && args[0] == "-j" {
			return []byte(`[{"dst":"default"},{"dst":"172.30.0.0/16"}]`), nil
		}
		return nil, nil
	}}
	for _, id := range []string{"gateway", "client", "denied"} {
		if err := server.db.UpsertDevice(&models.Device{ID: id, Hostname: id, OS: "windows", Version: "0.2.69"}); err != nil {
			t.Fatal(err)
		}
		server.hub.agents[id] = &Client{Send: make(chan []byte, 30)}
	}
	if _, err := server.db.db.Exec(`CREATE TABLE vpn_pilot_addresses(device TEXT PRIMARY KEY, slot INTEGER UNIQUE)`); err != nil {
		t.Fatal(err)
	}
	return server, &commands
}

func lanSession(server *Server, address string) *vpnSession {
	return &vpnSession{started: time.Now(), config: vpn.Config{Address: address, Network: server.vpnPilot.network, Endpoint: server.vpnPilot.endpoint, ServerKey: server.vpnPilot.publicKey}, status: vpn.Status{ID: strings.Repeat("a", 32), State: "preparing", Address: address, Updated: time.Now().Unix()}}
}

func TestVPNLANAuthorizationAndCollision(t *testing.T) {
	server, _ := lanServer(t)
	for _, input := range []struct {
		subnet  string
		clients []string
	}{
		{"0.0.0.0/0", []string{"client"}}, {"10.77.0.0/24", []string{"client"}}, {"172.30.1.0/24", []string{"client"}},
		{"192.168.1.0/24", nil}, {"192.168.1.0/24", []string{"gateway"}}, {"192.168.1.0/24", []string{"client", "denied"}}, {"", []string{"client"}},
	} {
		if server.configureVPNLAN("gateway", input.subnet, input.clients, "", lanSession(server, "10.77.0.2")) == nil {
			t.Fatalf("unsafe selection accepted: %+v", input)
		}
	}
	gateway := lanSession(server, "10.77.0.2")
	if err := server.configureVPNLAN("gateway", "192.168.1.0/24", []string{"client"}, "", gateway); err != nil {
		t.Fatal(err)
	}
	server.vpnPilot.sessions["gateway"] = gateway
	client := lanSession(server, "10.77.0.3")
	if server.configureVPNLAN("client", "", nil, "192.168.1.0/24", client) == nil {
		t.Fatal("client started before gateway ready")
	}
	gateway.status.State = "connected"
	gateway.provisioned = true
	if server.configureVPNLAN("denied", "", nil, "192.168.1.0/24", client) == nil {
		t.Fatal("unauthorized client accepted")
	}
	if server.configureVPNLAN("client", "", nil, "", client) == nil {
		t.Fatal("LAN route added without confirmation")
	}
	if err := server.configureVPNLAN("client", "", nil, "192.168.1.0/24", client); err != nil {
		t.Fatal(err)
	}
	if client.gateway != "gateway" || len(client.config.Routes) != 1 {
		t.Fatal("approved route missing")
	}
	gateway.status.Updated = time.Now().Add(-vpn.Lease - time.Second).Unix()
	if server.configureVPNLAN("client", "", nil, "192.168.1.0/24", client) == nil {
		t.Fatal("stale gateway accepted")
	}
	server.vpnPilot.sessions = map[string]*vpnSession{}
	if err := server.db.UpsertDevice(&models.Device{ID: "client", OS: "windows", Version: "0.2.68"}); err != nil {
		t.Fatal(err)
	}
	if server.configureVPNLAN("gateway", "192.168.1.0/24", []string{"client"}, "", lanSession(server, "10.77.0.2")) == nil {
		t.Fatal("old client accepted")
	}
}

func TestVPNLANBaselineFirewallIsDefaultDeny(t *testing.T) {
	server, commands := lanServer(t)
	if err := server.setupVPNFirewall(); err != nil {
		t.Fatal(err)
	}
	if len(*commands) != 4 {
		t.Fatal("incomplete baseline rules")
	}
	if (*commands)[0] != "iptables -I INPUT 1 -i rdpilot ! -s 10.77.0.0/24 -j DROP" {
		t.Fatal("LAN hosts can initiate traffic to hub")
	}
	if (*commands)[1] != "iptables -I FORWARD 1 -i rdpilot -j DROP" || (*commands)[2] != "iptables -I FORWARD 1 -o rdpilot -j DROP" {
		t.Fatal("unknown routes not denied")
	}
	if (*commands)[3] != "iptables -I FORWARD 1 -i rdpilot -o rdpilot -s 10.77.0.0/24 -d 10.77.0.0/24 -j ACCEPT" {
		t.Fatal("base rule exposes LAN without ACL")
	}
}

func TestVPNLANRouteACLAndCascadeCleanup(t *testing.T) {
	server, commands := lanServer(t)
	gateway := lanSession(server, "10.77.0.2")
	if err := server.configureVPNLAN("gateway", "192.168.1.0/24", []string{"client"}, "", gateway); err != nil {
		t.Fatal(err)
	}
	gateway.status.State = "connected"
	gateway.status.PublicKey = "gateway-key"
	gateway.provisioned = true
	server.vpnPilot.sessions["gateway"] = gateway
	if err := server.setupVPNLAN("gateway", gateway); err != nil {
		t.Fatal(err)
	}
	client := lanSession(server, "10.77.0.3")
	if err := server.configureVPNLAN("client", "", nil, "192.168.1.0/24", client); err != nil {
		t.Fatal(err)
	}
	client.provisioned = true
	client.status.PublicKey = "client-key"
	client.status.State = "connected"
	server.vpnPilot.sessions["client"] = client
	if err := server.setupVPNLAN("client", client); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(*commands, "\n")
	for _, expected := range []string{"ip route add 192.168.1.0/24 dev rdpilot", "-s 10.77.0.3/32 -d 192.168.1.0/24 -j ACCEPT", "-s 192.168.1.0/24 -d 10.77.0.3/32 -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT"} {
		if !strings.Contains(joined, expected) {
			t.Fatal("missing scoped route/ACL", expected)
		}
	}
	server.stopVPNSession("gateway", gateway, "admin disconnect")
	if gateway.provisioned || client.provisioned || len(gateway.lanCleanup) > 0 || len(client.lanCleanup) > 0 {
		t.Fatal("gateway stop left access behind")
	}
	if client.status.State != "disconnecting" {
		t.Fatal("client route not revoked by disconnect")
	}
	joined = strings.Join(*commands, "\n")
	if strings.Index(joined, "peer client-key remove") > strings.Index(joined, "peer gateway-key remove") {
		t.Fatal("gateway stopped before client access was revoked")
	}
	if !strings.Contains(joined, "ip route del 192.168.1.0/24 dev rdpilot") {
		t.Fatal("LAN route leaked")
	}
	before := len(*commands)
	server.stopVPNSession("gateway", gateway, "retry")
	if len(*commands) != before {
		t.Fatal("already-clean resources removed again")
	}
}

func TestVPNLANPartialSetupAndCleanupFailure(t *testing.T) {
	server, _ := lanServer(t)
	gateway := lanSession(server, "10.77.0.2")
	gateway.config.AdvertiseLAN = "192.168.1.0/24"
	gateway.status.State = "connected"
	gateway.provisioned = true
	gateway.status.LANClients = []string{"client"}
	server.vpnPilot.sessions["gateway"] = gateway
	client := lanSession(server, "10.77.0.3")
	client.gateway = "gateway"
	client.config.Routes = []string{gateway.config.AdvertiseLAN}
	client.provisioned = true
	server.vpnPilot.sessions["client"] = client
	original := server.vpnPilot.run
	insertions := 0
	cleanupFails := true
	server.vpnPilot.run = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" && args[0] == "-I" {
			insertions++
			if insertions == 2 {
				return nil, errors.New("insertion failed")
			}
		}
		if name == "iptables" && args[0] == "-D" && cleanupFails {
			return nil, errors.New("cleanup failed")
		}
		return original(name, args...)
	}
	if server.setupVPNLAN("client", client) == nil {
		t.Fatal("partial setup succeeded")
	}
	server.stopVPNSession("client", client, "setup failed")
	if client.provisioned || len(client.lanCleanup) != 1 || client.status.State != "error" {
		t.Fatal("unsafe cleanup state")
	}
	cleanupFails = false
	server.stopVPNSession("client", client, "retry")
	if len(client.lanCleanup) != 0 || client.status.State != "disconnecting" {
		t.Fatal("cleanup did not recover")
	}
}

func TestVPNLANAPIRequiresAdminAndSendsApprovedConfiguration(t *testing.T) {
	server, _ := lanServer(t)
	body := `{"device":"gateway","operation":"connect","advertise_lan":"192.168.1.0/24","lan_clients":["client"]}`
	for _, role := range []string{"viewer", "it_support", "admin"} {
		request := httptest.NewRequest("POST", "/api/vpn/pilot", strings.NewReader(body))
		request = request.WithContext(context.WithValue(request.Context(), userClaimsKey, &UserClaims{Username: role, Role: role}))
		response := httptest.NewRecorder()
		server.handleVPN(response, request)
		expected := 403
		if role == "admin" {
			expected = 202
		}
		if response.Code != expected {
			t.Fatalf("role %s: %d %s", role, response.Code, response.Body.String())
		}
	}
	session := server.vpnPilot.sessions["gateway"]
	_, public, _ := vpn.NewKey()
	report, _ := json.Marshal(vpn.Status{ID: session.status.ID, State: "ready", PublicKey: public})
	server.receiveVPN("gateway", report)
	<-server.hub.agents["gateway"].Send
	var sent struct {
		Data vpn.Command `json:"data"`
	}
	if err := json.Unmarshal(<-server.hub.agents["gateway"].Send, &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Data.Operation != "connect" || sent.Data.Config.AdvertiseLAN != "192.168.1.0/24" {
		t.Fatalf("missing LAN config: %+v", sent.Data)
	}
	server.vpnPilot.run = func(string, ...string) ([]byte, error) {
		return []byte(fmt.Sprintf("%s\t%d\n", public, time.Now().Unix())), nil
	}
	report, _ = json.Marshal(vpn.Status{ID: session.status.ID, State: "running"})
	server.receiveVPN("gateway", report)
	if session.status.State != "connected" {
		t.Fatal("gateway not connected")
	}
}
