package agent

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func pasteRemoteText(text string, setClipboard func(string) error, hotkey func([]string) error) error {
	if len(text) > 16384 || !utf8.ValidString(text) || strings.ContainsRune(text, '\x00') {
		return fmt.Errorf("teks clipboard maksimal 16 KiB UTF-8, tanpa NUL")
	}
	if err := setClipboard(text); err != nil {
		return fmt.Errorf("clipboard target gagal diperbarui; paste dibatalkan")
	}
	return hotkey([]string{"ControlLeft", "KeyV"})
}
