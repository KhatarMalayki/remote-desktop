//go:build windows

package agent

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/lxn/win"
)

func TestNativeInputDesktopProbe(t *testing.T) {
	if os.Getenv("RD_NATIVE_INPUT_PROBE") != "1" {
		t.Skip("set RD_NATIVE_INPUT_PROBE=1 for a zero-distance native mouse-move probe")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	binding := newRemoteDesktop()
	defer binding.close()
	oldDesktop, _, openErr := openInputDesktopProc.Call(0, 0, 0x181)
	if oldDesktop == 0 {
		t.Fatal(openErr)
	}
	if ok, _, setErr := setThreadDesktopProc.Call(oldDesktop); ok == 0 {
		closeDesktopProc.Call(oldDesktop)
		t.Fatal(setErr)
	}
	binding.current = oldDesktop
	input := win.MOUSE_INPUT{Type: win.INPUT_MOUSE}
	input.Mi.DwFlags = win.MOUSEEVENTF_MOVE
	oldErr := sendNativeInput("mouse", unsafe.Pointer(&input), unsafe.Sizeof(input))
	t.Logf("old desktop access 0x181: %v", oldErr)
	if err := binding.prepare(); err != nil {
		t.Fatal(err)
	}
	newErr := sendNativeInput("mouse", unsafe.Pointer(&input), unsafe.Sizeof(input))
	t.Logf("upstream desktop access %#x: %v", remoteDesktopAccess, newErr)
	if newErr != nil {
		t.Fatal(newErr)
	}
}

func TestNativeInputResultAndLayout(t *testing.T) {
	wantSize := uintptr(28)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		wantSize = 40
	}
	if unsafe.Sizeof(win.MOUSE_INPUT{}) != wantSize || unsafe.Sizeof(win.KEYBD_INPUT{}) != wantSize {
		t.Fatal("SendInput structures do not match the Windows INPUT ABI")
	}
	originalCall := sendInputCall
	t.Cleanup(func() { sendInputCall = originalCall })
	for _, test := range []struct {
		name      string
		sent      uintptr
		callErr   error
		wantError bool
	}{
		{"accepted", 1, syscall.Errno(5), false},
		{"rejected without reason", 0, syscall.Errno(0), true},
		{"access denied", 0, syscall.Errno(5), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			sendInputCall = func(args ...uintptr) (uintptr, uintptr, error) {
				if len(args) != 3 || args[0] != 1 || args[1] == 0 || args[2] != wantSize {
					t.Fatalf("unexpected SendInput arguments: %v", args)
				}
				return test.sent, 0, test.callErr
			}
			input := win.MOUSE_INPUT{Type: win.INPUT_MOUSE}
			err := sendNativeInput("mouse", unsafe.Pointer(&input), unsafe.Sizeof(input))
			if (err != nil) != test.wantError {
				t.Fatalf("error = %v, wantError = %v", err, test.wantError)
			}
			if err != nil && !strings.Contains(err.Error(), test.callErr.Error()) {
				t.Fatalf("lost native error: %v", err)
			}
		})
	}
}

func TestRemoteDesktopHandleLifecycle(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	binding := newRemoteDesktop()
	defer binding.close()
	if err := binding.prepare(); err != nil {
		t.Skipf("interactive Windows desktop unavailable: %v", err)
	}
	for count := 0; count < 20; count++ {
		previous := binding.current
		if err := binding.prepare(); err != nil {
			t.Fatal(err)
		}
		var needed uint32
		if ok, _, _ := getUserObjectInfoProc.Call(previous, 2, 0, 0, uintptr(unsafe.Pointer(&needed))); ok != 0 || needed != 0 {
			t.Fatal("previous desktop handle was not closed")
		}
	}
	binding.close()
	threadID, _, _ := getCurrentThreadIDProc.Call()
	current, _, _ := getThreadDesktopProc.Call(threadID)
	if binding.current != 0 || current != binding.original {
		t.Fatal("original thread desktop was not restored")
	}
}

func TestWindowsVirtualKeyCoverage(t *testing.T) {
	tests := []struct {
		code string
		key  string
		want uintptr
	}{
		{code: "KeyA", key: "a", want: 0x41},
		{code: "Digit9", key: "9", want: 0x39},
		{code: "ControlLeft", key: "Control", want: 0xA2},
		{code: "ArrowDown", key: "ArrowDown", want: 0x28},
		{code: "F12", key: "F12", want: 0x7B},
		{code: "Semicolon", key: ";", want: 0xBA},
		{code: "Numpad7", key: "7", want: 0x67},
	}
	for _, test := range tests {
		got, ok := windowsVirtualKey(test.code, test.key)
		if !ok || got != test.want {
			t.Fatalf("windowsVirtualKey(%q, %q) = %#x, %v; want %#x, true", test.code, test.key, got, ok, test.want)
		}
	}
}

func TestMouseButtonFlags(t *testing.T) {
	for button, want := range map[int][2]uintptr{0: {0x0002, 0x0004}, 1: {0x0020, 0x0040}, 2: {0x0008, 0x0010}} {
		down, up := mouseFlags(button)
		if down != want[0] || up != want[1] {
			t.Fatalf("button %d flags = %#x/%#x, want %#x/%#x", button, down, up, want[0], want[1])
		}
	}
}

func TestLegacyInputFallbackOnlyOnNormalDesktop(t *testing.T) {
	if !shouldUseLegacyInputFallback("Default", false) {
		t.Fatal("normal desktop must allow legacy input fallback")
	}
	if shouldUseLegacyInputFallback("Winlogon", false) || shouldUseLegacyInputFallback("", false) {
		t.Fatal("secure or unknown desktop must not use legacy input fallback")
	}
	if shouldUseLegacyInputFallback("Default", true) {
		t.Fatal("interactive user relay must use SendInput")
	}
}
