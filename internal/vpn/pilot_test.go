package vpn

import (
	"strings"
	"testing"
	"time"
)

func TestPilotRejectsUnsafeConfiguration(t *testing.T) {
	private, public, err := NewKey()
	if err != nil { t.Fatal(err) }
	config := Config{Address:"10.77.0.2", Network:"10.77.0.0/24", Endpoint:"vpn.example.test:51820", ServerKey:public}
	rendered, err := config.Render(private)
	if err != nil { t.Fatal(err) }
	for _, forbidden := range []string{"DNS", "0.0.0.0/0", "::/0", "PostUp", "PreUp", "Table", "SaveConfig"} {
		if strings.Contains(rendered, forbidden) { t.Fatal(forbidden) }
	}
	for _, subnet := range []string{"0.0.0.0/0", "0.0.0.0/1", "128.0.0.0/1", "::/0", "8.8.8.0/24", "10.77.0.1/24", "10.0.0.0/8"} {
		candidate := config; candidate.Network = subnet
		if candidate.Validate() == nil { t.Fatalf("accepted %s", subnet) }
	}
	for _, endpoint := range []string{"x:0", "x:99999", "10.77.0.1:51820", "x:51820\nPostUp = bad", "224.0.0.1:51820"} {
		candidate := config; candidate.Endpoint = endpoint
		if candidate.Validate() == nil { t.Fatalf("accepted %s", endpoint) }
	}
	for _, address := range []string{"10.77.0.0", "10.77.0.1", "10.77.0.255", "192.168.1.2"} {
		candidate := config; candidate.Address = address
		if candidate.Validate() == nil { t.Fatalf("accepted %s", address) }
	}
	if CheckRoutes(config.Network, []string{"0.0.0.0/0", "192.168.1.0/24"}) != nil { t.Fatal("safe routes rejected") }
	for _, route := range []string{"10.77.0.0/24", "10.0.0.0/8", "0.0.0.0/1", "10.77.0.50/32", "broken"} {
		if CheckRoutes(config.Network, []string{route}) == nil { t.Fatalf("route collision accepted: %s", route) }
	}
}

func TestPilotLeaseAndCommandExpiry(t *testing.T) {
	now := time.Now()
	command := Command{ID:strings.Repeat("a",32), Expires:now.Add(time.Minute).Unix()}
	if !ValidCommand(command,now) { t.Fatal("valid command rejected") }
	command.Expires=now.Add(-time.Second).Unix()
	if ValidCommand(command,now) { t.Fatal("expired command accepted") }
	if LeaseExpired(now,now,now) { t.Fatal("fresh lease rejected") }
	if !LeaseExpired(now,now.Add(-2*time.Minute),now.Add(-61*time.Second)) { t.Fatal("lost dashboard does not disconnect") }
	if !LeaseExpired(now,now.Add(-16*time.Minute),now) { t.Fatal("pilot exceeded 15 minutes") }
}
