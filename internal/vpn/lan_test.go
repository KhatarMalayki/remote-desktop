package vpn

import (
	"strings"
	"testing"
)

func TestLANConfigurationBoundaries(t *testing.T) {
	private, public, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	base := Config{Address: "10.77.0.2", Network: "10.77.0.0/24", Endpoint: "vpn.example.test:51820", ServerKey: public}
	for _, subnet := range []string{"0.0.0.0/0", "8.8.8.0/24", "192.168.1.1/24", "10.0.0.0/8", "192.168.0.0/15", "192.168.1.1/32", "::/0", "10.77.0.0/24", "192.168.1.0/24\nPostUp=bad"} {
		config := base
		config.AdvertiseLAN = subnet
		if config.Validate() == nil {
			t.Errorf("unsafe gateway accepted: %q", subnet)
		}
		config.AdvertiseLAN = ""
		config.Routes = []string{subnet}
		if config.Validate() == nil {
			t.Errorf("unsafe route accepted: %q", subnet)
		}
	}
	for _, routes := range [][]string{{""}, {"192.168.1.0/24", "192.168.2.0/24"}} {
		config := base
		config.Routes = routes
		if config.Validate() == nil {
			t.Fatal("invalid routes accepted")
		}
	}
	gateway := base
	gateway.AdvertiseLAN = "192.168.1.0/24"
	rendered, err := gateway.Render(private)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered, gateway.AdvertiseLAN) {
		t.Fatal("gateway routed its own LAN into VPN")
	}
	client := base
	client.Routes = []string{gateway.AdvertiseLAN}
	rendered, err = client.Render(private)
	if err != nil || !strings.Contains(rendered, "AllowedIPs = 10.77.0.0/24, 192.168.1.0/24\n") {
		t.Fatalf("missing client split route: %v %s", err, rendered)
	}
	client.AdvertiseLAN = gateway.AdvertiseLAN
	if client.Validate() == nil {
		t.Fatal("mixed roles accepted")
	}
	client.AdvertiseLAN = ""
	client.Endpoint = "192.168.1.20:51820"
	if client.Validate() == nil {
		t.Fatal("endpoint capture accepted")
	}
	for _, routes := range [][]string{{"192.168.0.0/16"}, {"192.168.1.99/32"}, {"broken"}} {
		if CheckLANRoutes("192.168.1.0/24", routes) == nil {
			t.Fatal("route collision accepted", routes)
		}
	}
	if err := CheckLANRoutes("192.168.1.0/24", []string{"0.0.0.0/0", "10.77.0.0/24"}); err != nil {
		t.Fatal(err)
	}
}
