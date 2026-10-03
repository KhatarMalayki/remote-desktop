package agent

import (
	"fmt"
	"image"
	"math"
)

var macKeyCodes = map[string]uint16{
	"KeyA": 0, "KeyS": 1, "KeyD": 2, "KeyF": 3, "KeyH": 4, "KeyG": 5,
	"KeyZ": 6, "KeyX": 7, "KeyC": 8, "KeyV": 9, "KeyB": 11, "KeyQ": 12,
	"KeyW": 13, "KeyE": 14, "KeyR": 15, "KeyY": 16, "KeyT": 17,
	"Digit1": 18, "Digit2": 19, "Digit3": 20, "Digit4": 21, "Digit6": 22,
	"Digit5": 23, "Equal": 24, "Digit9": 25, "Digit7": 26, "Minus": 27,
	"Digit8": 28, "Digit0": 29, "BracketRight": 30, "KeyO": 31, "KeyU": 32,
	"BracketLeft": 33, "KeyI": 34, "KeyP": 35, "Enter": 36, "KeyL": 37,
	"KeyJ": 38, "Quote": 39, "KeyK": 40, "Semicolon": 41, "Backslash": 42,
	"Comma": 43, "Slash": 44, "KeyN": 45, "KeyM": 46, "Period": 47,
	"Tab": 48, "Space": 49, "Backquote": 50, "Backspace": 51, "Escape": 53,
	"MetaRight": 54, "MetaLeft": 55, "ShiftLeft": 56, "AltLeft": 58,
	"ControlLeft": 59, "ShiftRight": 60, "AltRight": 61, "ControlRight": 62,
	"NumpadDecimal": 65, "NumpadMultiply": 67, "NumpadAdd": 69,
	"NumLock": 71, "NumpadDivide": 75, "NumpadEnter": 76, "NumpadSubtract": 78,
	"NumpadEqual": 81, "Numpad0": 82, "Numpad1": 83, "Numpad2": 84,
	"Numpad3": 85, "Numpad4": 86, "Numpad5": 87, "Numpad6": 88,
	"Numpad7": 89, "Numpad8": 91, "Numpad9": 92,
	"F1": 122, "F2": 120, "F3": 99, "F4": 118, "F5": 96, "F6": 97,
	"F7": 98, "F8": 100, "F9": 101, "F10": 109, "F11": 103, "F12": 111,
	"Home": 115, "PageUp": 116, "Delete": 117, "End": 119, "PageDown": 121,
	"ArrowLeft": 123, "ArrowRight": 124, "ArrowDown": 125, "ArrowUp": 126,
}

func macModifierFlags(pressed map[string]bool) uint64 {
	var flags uint64
	for code := range pressed {
		switch code {
		case "ShiftLeft", "ShiftRight":
			flags |= 1 << 17
		case "ControlLeft", "ControlRight":
			flags |= 1 << 18
		case "AltLeft", "AltRight":
			flags |= 1 << 19
		case "MetaLeft", "MetaRight":
			flags |= 1 << 20
		}
	}
	return flags
}

func macPointerPosition(x, y float64, bounds image.Rectangle) (float64, float64, error) {
	if bounds.Empty() || math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) || x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, fmt.Errorf("koordinat remote tidak valid")
	}
	return float64(bounds.Min.X) + x*float64(bounds.Dx()-1), float64(bounds.Min.Y) + y*float64(bounds.Dy()-1), nil
}
