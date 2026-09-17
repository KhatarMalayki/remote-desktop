//go:build windows

package agent

import "testing"

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
