//go:build darwin && cgo

package agent

/*
#cgo LDFLAGS: -framework ApplicationServices -framework Cocoa -framework Carbon
#include <stdlib.h>
int rdMacCaptureAllowed(void);
int rdMacInputAllowed(void);
int rdMacKey(unsigned short key, int down, unsigned long long flags);
int rdMacMouse(int action, int button, double x, double y, int dx, int dy, unsigned long long flags, int clicks);
int rdMacClipboardSet(const char *text);
char *rdMacClipboardGet(int *length);
*/
import "C"

import (
	"fmt"
	"image"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
	"unsafe"
)

var macInput = struct {
	sync.Mutex
	keys               map[string]bool
	buttons            map[int]bool
	x, y               float64
	lastClick          time.Time
	lastButton, clicks int
	clickX, clickY     float64
}{keys: map[string]bool{}, buttons: map[int]bool{}}

type remoteDesktop struct{}

func newRemoteDesktop() *remoteDesktop { return &remoteDesktop{} }
func (*remoteDesktop) close()          {}
func (*remoteDesktop) prepare() error {
	if C.rdMacCaptureAllowed() == 0 {
		return fmt.Errorf("macOS: izinkan Screen Recording untuk rd-agent di System Settings > Privacy & Security, lalu restart agent")
	}
	return nil
}

func activeInputDesktopName() string { return "macOS user session" }
func isSecureInputDesktop() bool     { return false }
func setInteractiveInputMode(bool)   {}

func macInputPermission() error {
	switch C.rdMacInputAllowed() {
	case 1:
		return nil
	case -1:
		return fmt.Errorf("macOS Secure Input aktif; kontrol keyboard ditolak, tutup input terlindungi di target")
	default:
		return fmt.Errorf("macOS: izinkan Accessibility untuk rd-agent di System Settings > Privacy & Security")
	}
}

func macSendKey(code string, down bool) error {
	key, ok := macKeyCodes[code]
	if !ok {
		return fmt.Errorf("tombol macOS tidak didukung: %q", code)
	}
	previous := macInput.keys[code]
	if down {
		macInput.keys[code] = true
	} else {
		delete(macInput.keys, code)
	}
	pressed := C.int(0)
	if down {
		pressed = 1
	}
	if C.rdMacKey(C.ushort(key), pressed, C.ulonglong(macModifierFlags(macInput.keys))) == 0 {
		if previous {
			macInput.keys[code] = true
		} else {
			delete(macInput.keys, code)
		}
		return fmt.Errorf("gagal membuat event keyboard macOS")
	}
	return nil
}

func handleRemoteInput(command remoteCommand, bounds image.Rectangle) error {
	macInput.Lock()
	defer macInput.Unlock()
	if err := macInputPermission(); err != nil {
		return err
	}
	if command.Type == "key_down" || command.Type == "key_up" {
		return macSendKey(command.Code, command.Type == "key_down")
	}
	posX, posY, err := macPointerPosition(command.X, command.Y, bounds)
	if err != nil {
		return err
	}
	action, button := 0, -1
	switch command.Type {
	case "mouse_move":
		for _, held := range []int{0, 2, 1} {
			if macInput.buttons[held] {
				button = held
				break
			}
		}
	case "mouse_down", "mouse_up":
		if command.Button < 0 || command.Button > 2 {
			return fmt.Errorf("tombol mouse tidak valid")
		}
		button = command.Button
		action = 1
		if command.Type == "mouse_up" {
			action = 2
		}
	case "mouse_wheel":
		if command.DeltaX < -32767 || command.DeltaX > 32767 || command.DeltaY < -32767 || command.DeltaY > 32767 {
			return fmt.Errorf("scroll di luar batas")
		}
		action = 3
	default:
		return fmt.Errorf("perintah remote tidak didukung: %q", command.Type)
	}
	if action == 1 {
		if button == macInput.lastButton && time.Since(macInput.lastClick) < 500*time.Millisecond && posX-macInput.clickX < 4 && posX-macInput.clickX > -4 && posY-macInput.clickY < 4 && posY-macInput.clickY > -4 {
			macInput.clicks = macInput.clicks%3 + 1
		} else {
			macInput.clicks = 1
		}
		macInput.lastClick, macInput.lastButton = time.Now(), button
		macInput.clickX, macInput.clickY = posX, posY
	}
	if C.rdMacMouse(C.int(action), C.int(button), C.double(posX), C.double(posY), C.int(command.DeltaX), C.int(command.DeltaY), C.ulonglong(macModifierFlags(macInput.keys)), C.int(macInput.clicks)) == 0 {
		return fmt.Errorf("gagal membuat event mouse macOS")
	}
	macInput.x, macInput.y = posX, posY
	if action == 1 {
		macInput.buttons[button] = true
	}
	if action == 2 {
		delete(macInput.buttons, button)
	}
	return nil
}

func releaseRemoteInputs() {
	macInput.Lock()
	defer macInput.Unlock()
	for code := range macInput.keys {
		_ = macSendKey(code, false)
	}
	for button := range macInput.buttons {
		C.rdMacMouse(2, C.int(button), C.double(macInput.x), C.double(macInput.y), 0, 0, 0, 1)
	}
	clear(macInput.keys)
	clear(macInput.buttons)
}

func sendRemoteHotkey(keys []string) error {
	macInput.Lock()
	defer macInput.Unlock()
	if len(keys) == 0 || len(keys) > 8 {
		return fmt.Errorf("hotkey harus berisi 1-8 tombol")
	}
	if err := macInputPermission(); err != nil {
		return err
	}
	for _, code := range keys {
		if _, ok := macKeyCodes[code]; !ok {
			return fmt.Errorf("tombol macOS tidak didukung: %q", code)
		}
	}
	pressed := []string{}
	defer func() {
		for index := len(pressed) - 1; index >= 0; index-- {
			_ = macSendKey(pressed[index], false)
		}
	}()
	for _, code := range keys {
		if macInput.keys[code] {
			continue
		}
		if err := macSendKey(code, true); err != nil {
			return err
		}
		pressed = append(pressed, code)
	}
	return nil
}

func setRemoteClipboard(text string) error {
	if len(text) > 16384 || !utf8.ValidString(text) || strings.ContainsRune(text, '\x00') {
		return fmt.Errorf("clipboard maksimal 16 KiB UTF-8 tanpa NUL")
	}
	value := C.CString(text)
	defer C.free(unsafe.Pointer(value))
	if C.rdMacClipboardSet(value) == 0 {
		return fmt.Errorf("gagal menulis clipboard macOS")
	}
	return nil
}

func getRemoteClipboard() (string, error) {
	var length C.int
	value := C.rdMacClipboardGet(&length)
	if value == nil {
		return "", fmt.Errorf("clipboard bukan teks UTF-8 atau melebihi 16 KiB")
	}
	defer C.free(unsafe.Pointer(value))
	text := C.GoStringN(value, length)
	if !utf8.ValidString(text) || strings.ContainsRune(text, '\x00') {
		return "", fmt.Errorf("clipboard bukan teks UTF-8 tanpa NUL")
	}
	return text, nil
}
