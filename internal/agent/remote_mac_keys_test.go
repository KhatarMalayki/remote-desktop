package agent

import (
	"image"
	"math"
	"testing"
)

func TestMacInputMapping(t *testing.T) {
	for code, expected := range map[string]uint16{"KeyA": 0, "KeyE": 14, "MetaLeft": 55, "ControlLeft": 59, "ArrowUp": 126, "NumpadEnter": 76} {
		if actual, ok := macKeyCodes[code]; !ok || actual != expected {
			t.Fatalf("%s: %d != %d", code, actual, expected)
		}
	}
	if _, ok := macKeyCodes["Unknown"]; ok {
		t.Fatal("unknown key accepted")
	}
	pressed := map[string]bool{"MetaLeft": true, "MetaRight": true, "ShiftLeft": true, "KeyV": true}
	delete(pressed, "MetaLeft")
	if flags := macModifierFlags(pressed); flags != 1<<20|1<<17 {
		t.Fatalf("wrong modifiers after releasing one Command key: %d", flags)
	}
	posX, posY, err := macPointerPosition(1, 1, image.Rect(-1920, -1080, 0, 0))
	if err != nil || posX != -1 || posY != -1 {
		t.Fatalf("secondary display coordinates: %v %v %v", posX, posY, err)
	}
	for _, invalid := range []float64{-1, 1.1, math.NaN(), math.Inf(1)} {
		if _, _, err := macPointerPosition(invalid, 0, image.Rect(0, 0, 100, 100)); err == nil {
			t.Fatalf("accepted %v", invalid)
		}
	}
}
