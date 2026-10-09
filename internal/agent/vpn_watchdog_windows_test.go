package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestVPNLeaseRefreshRetriesTransientReader(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(vpnDirectory(), "lease.json")
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	done := make(chan struct{})
	go func() { time.Sleep(100 * time.Millisecond); reader.Close(); close(done) }()
	err = vpnAtomicFile("lease.json", []byte("renewed"))
	<-done
	if err != nil {
		t.Fatalf("transient watchdog reader stopped lease renewal: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "renewed" {
		t.Fatalf("lease not renewed: %s %v", data, err)
	}
}

func TestVPNLeasePersistentLockPreservesPreviousFile(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(vpnDirectory(), "lease.json")
	if err := os.WriteFile(path, []byte("previous"), 0600); err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	started := time.Now()
	err = vpnAtomicFile("lease.json", []byte("renewed"))
	if err == nil {
		t.Fatal("persistent reader lock must not be ignored")
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("retry exceeded bounded deadline")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || string(data) != "previous" {
		t.Fatalf("old lease was lost: %s %v", data, readErr)
	}
	files, err := filepath.Glob(filepath.Join(vpnDirectory(), ".vpn-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("temporary file leak: %v %v", files, err)
	}
}

func TestVPNWatchdogRetriesFailedStop(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	originalRunning, originalDisconnect := vpnRunningCheck, vpnDisconnectFunc
	t.Cleanup(func() { vpnRunningCheck, vpnDisconnectFunc = originalRunning, originalDisconnect })
	vpnRunningCheck = func() (bool, error) { return true, nil }
	stopErr := errors.New("temporary service stop failure")
	vpnDisconnectFunc = func() error { return stopErr }
	now := time.Now()
	initial := vpnLease{ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Started: now.Add(-2 * time.Minute), Ack: now.Add(-65 * time.Second)}
	done, _, _, err := vpnWatchdogStep(initial, true, initial.Ack)
	if done || !errors.Is(err, stopErr) {
		t.Fatalf("watchdog abandoned live tunnel: done=%v err=%v", done, err)
	}
	vpnDisconnectFunc = func() error { return nil }
	done, _, _, err = vpnWatchdogStep(initial, true, initial.Ack)
	if !done || err != nil {
		t.Fatalf("successful retry did not finish: done=%v err=%v", done, err)
	}
}

func TestVPNWatchdogToleratesTransientUnreadableLease(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	origRunning := vpnRunningCheck
	origDisconnect := vpnDisconnectFunc
	disconnected := false
	vpnRunningCheck = func() (bool, error) { return true, nil }
	vpnDisconnectFunc = func() error { disconnected = true; return nil }
	defer func() {
		vpnRunningCheck = origRunning
		vpnDisconnectFunc = origDisconnect
	}()
	now := time.Now()
	initial := vpnLease{ID: "testsessionid123456789012345678", Started: now, Ack: now}
	// lease.json does not exist yet (simulating transient write lock or atomic replace in flight)
	done, running, ack, err := vpnWatchdogStep(initial, true, now.Add(-5*time.Second))
	if done || disconnected {
		t.Fatalf("watchdog disconnected prematurely during transient unreadable lease: done=%v err=%v", done, err)
	}
	if ack.IsZero() {
		t.Fatal("fresh ack was discarded")
	}
	// When ack is genuinely older than Lease, it must disconnect
	done, running, ack, err = vpnWatchdogStep(initial, true, now.Add(-65*time.Second))
	if !done || !disconnected {
		t.Fatalf("watchdog failed to disconnect expired lease: done=%v running=%v", done, running)
	}
}
