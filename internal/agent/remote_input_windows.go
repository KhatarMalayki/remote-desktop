//go:build windows

package agent

import (
	"fmt"
	"image"
	"os/exec"
	"strings"
	"syscall"
)

var (
	user32DLL      = syscall.NewLazyDLL("user32.dll")
	setCursorPos   = user32DLL.NewProc("SetCursorPos")
	mouseEventProc = user32DLL.NewProc("mouse_event")
	keybdEventProc = user32DLL.NewProc("keybd_event")
)

const (
	keyEventKeyUp = 0x0002
)

func handleRemoteInput(command remoteCommand, bounds image.Rectangle) error {
	switch command.Type {
	case "mouse_move":
		setRemoteCursor(command.X, command.Y, bounds)
		return nil
	case "mouse_down", "mouse_up":
		setRemoteCursor(command.X, command.Y, bounds)
		down, up := mouseFlags(command.Button)
		flag := down
		if command.Type == "mouse_up" {
			flag = up
		}
		mouseEventProc.Call(flag, 0, 0, 0, 0)
		return nil
	case "mouse_wheel":
		setRemoteCursor(command.X, command.Y, bounds)
		if command.DeltaY != 0 {
			mouseEventProc.Call(0x0800, 0, 0, uintptr(int32(-command.DeltaY)), 0)
		}
		if command.DeltaX != 0 {
			mouseEventProc.Call(0x01000, 0, 0, uintptr(int32(command.DeltaX)), 0)
		}
		return nil
	case "key_down", "key_up":
		vk, ok := windowsVirtualKey(command.Code, command.Key)
		if !ok {
			return fmt.Errorf("unsupported key %q (%s)", command.Key, command.Code)
		}
		flags := uintptr(0)
		if command.Type == "key_up" {
			flags = keyEventKeyUp
		}
		keybdEventProc.Call(vk, 0, flags, 0)
		return nil
	default:
		return fmt.Errorf("unsupported command %q", command.Type)
	}
}

func mouseFlags(button int) (uintptr, uintptr) {
	switch button {
	case 1:
		return 0x0020, 0x0040
	case 2:
		return 0x0008, 0x0010
	default:
		return 0x0002, 0x0004
	}
}

func releaseRemoteInputs() {
	for _, vk := range []uintptr{0x10, 0x11, 0x12, 0x5B, 0x5C} {
		keybdEventProc.Call(vk, 0, keyEventKeyUp, 0)
	}
	for _, flag := range []uintptr{0x0004, 0x0010, 0x0040} {
		mouseEventProc.Call(flag, 0, 0, 0, 0)
	}
}

func setRemoteCursor(x, y float64, bounds image.Rectangle) {
	if x < 0 {
		x = 0
	}
	if x > 1 {
		x = 1
	}
	if y < 0 {
		y = 0
	}
	if y > 1 {
		y = 1
	}
	px := bounds.Min.X + int(x*float64(bounds.Dx()-1))
	py := bounds.Min.Y + int(y*float64(bounds.Dy()-1))
	setCursorPos.Call(uintptr(int32(px)), uintptr(int32(py)))
}

func windowsVirtualKey(code, key string) (uintptr, bool) {
	if strings.HasPrefix(code, "Key") && len(code) == 4 {
		return uintptr(code[3]), true
	}
	if strings.HasPrefix(code, "Digit") && len(code) == 6 {
		return uintptr(code[5]), true
	}
	if strings.HasPrefix(code, "F") && len(code) <= 3 {
		var n int
		if _, err := fmt.Sscanf(code, "F%d", &n); err == nil && n >= 1 && n <= 24 {
			return uintptr(0x70 + n - 1), true
		}
	}
	keys := map[string]uintptr{
		"Enter": 0x0D, "NumpadEnter": 0x0D, "Escape": 0x1B, "Backspace": 0x08, "Tab": 0x09,
		"Space": 0x20, "ArrowLeft": 0x25, "ArrowUp": 0x26, "ArrowRight": 0x27, "ArrowDown": 0x28,
		"Delete": 0x2E, "Insert": 0x2D, "Home": 0x24, "End": 0x23, "PageUp": 0x21, "PageDown": 0x22,
		"ShiftLeft": 0xA0, "ShiftRight": 0xA1, "ControlLeft": 0xA2, "ControlRight": 0xA3,
		"AltLeft": 0xA4, "AltRight": 0xA5, "MetaLeft": 0x5B, "MetaRight": 0x5C,
		"CapsLock": 0x14, "NumLock": 0x90, "ScrollLock": 0x91, "PrintScreen": 0x2C, "Pause": 0x13,
		"Semicolon": 0xBA, "Equal": 0xBB, "Comma": 0xBC, "Minus": 0xBD, "Period": 0xBE,
		"Slash": 0xBF, "Backquote": 0xC0, "BracketLeft": 0xDB, "Backslash": 0xDC,
		"BracketRight": 0xDD, "Quote": 0xDE,
		"Numpad0": 0x60, "Numpad1": 0x61, "Numpad2": 0x62, "Numpad3": 0x63, "Numpad4": 0x64,
		"Numpad5": 0x65, "Numpad6": 0x66, "Numpad7": 0x67, "Numpad8": 0x68, "Numpad9": 0x69,
		"NumpadMultiply": 0x6A, "NumpadAdd": 0x6B, "NumpadSubtract": 0x6D, "NumpadDecimal": 0x6E, "NumpadDivide": 0x6F,
	}
	vk, ok := keys[code]
	if !ok && key == " " {
		return 0x20, true
	}
	return vk, ok
}

func setRemoteClipboard(text string) error {
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", "$input | Set-Clipboard")
	command.Stdin = strings.NewReader(text)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("gagal menulis clipboard: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func getRemoteClipboard() (string, error) {
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", "Get-Clipboard -Raw")
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gagal membaca clipboard: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r"), nil
}
