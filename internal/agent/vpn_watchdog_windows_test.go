package agent

import (
	"errors"
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

func TestVPNLifecycleLockExcludesConcurrentSession(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	lock, err := vpnLockLifecycle(0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if other, err := vpnLockLifecycle(0); !errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		if other != nil {
			other.Close()
		}
		t.Fatalf("second lifecycle lock must be excluded: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := vpnLockLifecycle(0)
	if err != nil {
		t.Fatalf("next session cannot acquire released lock: %v", err)
	}
	next.Close()
}
