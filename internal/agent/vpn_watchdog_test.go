package agent

import (
	"strings"
	"testing"
	"time"
)

func TestVPNWatchdogLeaseOwnership(t *testing.T) {
	now := time.Now()
	initial := vpnLease{ID: strings.Repeat("a", 32), Started: now.Add(-time.Minute), Ack: now.Add(-time.Minute)}
	next := vpnLease{ID: strings.Repeat("b", 32), Started: now.Add(-time.Second), Ack: now}
	if !vpnLeaseSuperseded(initial, next, now) {
		t.Fatal("old watchdog must retire for a valid new session")
	}
	for _, bad := range []vpnLease{
		initial,
		{ID: "invalid", Started: now, Ack: now},
		{ID: next.ID, Started: now.Add(-2 * time.Minute), Ack: now},
		{ID: next.ID, Started: now.Add(time.Second), Ack: now},
		{ID: next.ID, Started: now, Ack: now.Add(time.Second)},
		{ID: next.ID, Started: now.Add(-2 * time.Minute), Ack: now.Add(-2 * time.Minute)},
	} {
		if vpnLeaseSuperseded(initial, bad, now) {
			t.Fatalf("invalid lease replaced watchdog: %+v", bad)
		}
	}
}
