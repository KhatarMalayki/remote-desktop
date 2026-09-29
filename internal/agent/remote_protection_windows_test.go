//go:build windows

package agent

import (
	"syscall"
	"testing"
)

func TestRemoteInputProtectionCleanup(t *testing.T) {
	original := blockRemoteInputCall
	defer func() { blockRemoteInputCall = original }()
	var calls []uintptr
	blockRemoteInputCall = func(args ...uintptr) (uintptr, uintptr, error) { calls = append(calls, args[0]); return 1, 0, nil }
	var protection remoteProtection
	if err := protection.block(true); err != nil {
		t.Fatal(err)
	}
	if !protection.active() {
		t.Fatal("protection not active")
	}
	if err := protection.block(true); err != nil {
		t.Fatal(err)
	}
	protection.close()
	protection.close()
	if protection.active() || len(calls) != 2 || calls[0] != 1 || calls[1] != 0 {
		t.Fatalf("incorrect cleanup: %v", calls)
	}
	blockRemoteInputCall = func(...uintptr) (uintptr, uintptr, error) { return 0, 0, syscall.Errno(5) }
	if err := protection.block(true); err == nil || protection.active() {
		t.Fatal("failed BlockInput advertised as active")
	}
}

func TestRemoteProtectionDesktopTransitionReassertsBlockInput(t *testing.T) {
	original := blockRemoteInputCall
	defer func() { blockRemoteInputCall = original }()
	var calls []uintptr
	blockRemoteInputCall = func(args ...uintptr) (uintptr, uintptr, error) { calls = append(calls, args[0]); return 1, 0, nil }
	var protection remoteProtection
	_ = protection.block(true)
	calls = nil
	protection.onDesktopChange("Winlogon")
	if len(calls) != 1 || calls[0] != 1 {
		t.Fatalf("expected BlockInput(1) on desktop change, got %v", calls)
	}
	if protection.desktop != "Winlogon" {
		t.Fatalf("expected desktop 'Winlogon', got %q", protection.desktop)
	}
}
