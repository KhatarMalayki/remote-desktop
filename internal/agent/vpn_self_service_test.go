package agent

import "testing"

func TestVPNSelfServiceOfflineAndInvalidOperation(t *testing.T) {
	agent := &Agent{vpnPilot: &vpnClient{}}
	if _, err := agent.requestSelfVPN("connect"); err == nil {
		t.Fatal("offline request accepted")
	}
	if _, err := agent.requestSelfVPN("configure"); err == nil {
		t.Fatal("arbitrary operation accepted")
	}
}
