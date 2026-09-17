//go:build windows

package agent

import (
	"fmt"
	"image"
	"strings"
	"syscall"
)

var (
	user32DLL      = syscall.NewLazyDLL("user32.dll")
	setCursorPos   = user32DLL.NewProc("SetCursorPos")
	mouseEventProc = user32DLL.NewProc("mouse_event")
	keybdEventProc = user32DLL.NewProc("keybd_event")
)

func handleRemoteInput(command remoteCommand, bounds image.Rectangle) error {
	switch command.Type {
	case "mouse_move":
		setRemoteCursor(command.X, command.Y, bounds)
		return nil
	case "mouse_click":
		setRemoteCursor(command.X, command.Y, bounds)
		down, up := uintptr(0x0002), uintptr(0x0004)
		if command.Button == 2 {
			down, up = 0x0008, 0x0010
		} else if command.Button == 1 {
			down, up = 0x0020, 0x0040
		}
		mouseEventProc.Call(down, 0, 0, 0, 0)
		mouseEventProc.Call(up, 0, 0, 0, 0)
		return nil
	case "key_down":
		if command.Key == "Control" || command.Key == "Alt" || command.Key == "Shift" || command.Key == "Meta" {
			return nil
		}
		vk, ok := windowsVirtualKey(command.Code, command.Key)
		if !ok {
			return fmt.Errorf("unsupported key %q", command.Key)
		}
		var modifiers []uintptr
		if command.Ctrl {
			modifiers = append(modifiers, 0x11)
		}
		if command.Alt {
			modifiers = append(modifiers, 0x12)
		}
		if command.Shift {
			modifiers = append(modifiers, 0x10)
		}
		if command.Meta {
			modifiers = append(modifiers, 0x5B)
		}
		for _, modifier := range modifiers {
			keybdEventProc.Call(modifier, 0, 0, 0)
		}
		keybdEventProc.Call(vk, 0, 0, 0)
		keybdEventProc.Call(vk, 0, 0x0002, 0)
		for i := len(modifiers) - 1; i >= 0; i-- {
			keybdEventProc.Call(modifiers[i], 0, 0x0002, 0)
		}
		return nil
	default:
		return fmt.Errorf("unsupported command %q", command.Type)
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
	setCursorPos.Call(uintptr(px), uintptr(py))
}

func windowsVirtualKey(code, key string) (uintptr, bool) {
	if strings.HasPrefix(code, "Key") && len(code) == 4 {
		return uintptr(code[3]), true
	}
	if strings.HasPrefix(code, "Digit") && len(code) == 6 {
		return uintptr(code[5]), true
	}
	keys := map[string]uintptr{
		"Enter": 0x0D, "Escape": 0x1B, "Backspace": 0x08, "Tab": 0x09,
		" ": 0x20, "Space": 0x20, "ArrowLeft": 0x25, "ArrowUp": 0x26,
		"ArrowRight": 0x27, "ArrowDown": 0x28, "Delete": 0x2E, "Home": 0x24,
		"End": 0x23, "PageUp": 0x21, "PageDown": 0x22,
	}
	vk, ok := keys[key]
	return vk, ok
}
