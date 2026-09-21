package agent

import "testing"

func TestParseRustDeskID(t *testing.T) {
	if got := parseRustDeskID("338317347\r\n"); got != "338317347" {
		t.Fatalf("parseRustDeskID() = %q", got)
	}
	if got := parseRustDeskID("RustDesk unavailable"); got != "" {
		t.Fatalf("invalid output returned %q", got)
	}
}
