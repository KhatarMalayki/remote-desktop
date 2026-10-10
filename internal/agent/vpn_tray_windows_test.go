//go:build windows

package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestVPNTrayPanelURL(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"https://admin.example/path?api_key=secret#token", "https://admin.example/"},
		{"wss://user:secret@admin.example/ws", "https://admin.example/"},
		{"ws://localhost:8080/ws", "http://localhost:8080/"},
		{"https://[::1]:8443/path", "https://[::1]:8443/"},
		{"javascript:alert(1)", ""},
		{"file:///C:/Windows/system32/cmd.exe", ""},
		{"https:///missing-host", ""},
		{"https://example.com/\n", ""},
		{"", ""},
	} {
		if got := vpnPanelURL(test.input); got != test.want {
			t.Errorf("vpnPanelURL(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestVPNTrayStatus(t *testing.T) {
	if got := vpnTrayStatus(false, nil); got != "VPN: Nonaktif" {
		t.Fatal(got)
	}
	for _, running := range []bool{false, true} {
		if got := vpnTrayStatus(running, errors.New("access denied")); got != "VPN: Status tidak diketahui" {
			t.Fatal(got)
		}
	}
	if got := vpnTrayStatus(true, nil); !strings.HasPrefix(got, "VPN: Aktif - ") {
		t.Fatal(got)
	}
}
