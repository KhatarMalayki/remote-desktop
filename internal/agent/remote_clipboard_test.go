package agent

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestPasteRemoteTextValidationAndSequence(t *testing.T) {
	for _, invalid := range []string{strings.Repeat("a", 16385), "with\x00null", "broken \xff utf8"} {
		if err := pasteRemoteText(invalid, func(string) error { return nil }, func([]string) error { return nil }); err == nil {
			t.Fatalf("accepted invalid clipboard payload %q", invalid)
		}
	}
	setCalled := false
	expectedErr := errors.New("write failed")
	err := pasteRemoteText("valid phrase", func(string) error {
		setCalled = true
		return expectedErr
	}, func([]string) error {
		t.Fatal("hotkey sent after clipboard write failure")
		return nil
	})
	if !setCalled || err == nil || !strings.Contains(err.Error(), "clipboard target gagal") {
		t.Fatalf("expected write failure, got %v", err)
	}
	hotkeyKeys := []string{}
	written := ""
	err = pasteRemoteText("teks paste sukses", func(text string) error {
		written = text
		return nil
	}, func(keys []string) error {
		hotkeyKeys = append(hotkeyKeys, keys...)
		return nil
	})
	if err != nil || written != "teks paste sukses" || strings.Join(hotkeyKeys, ",") != strings.Join(remotePasteKeys(runtime.GOOS), ",") {
		t.Fatalf("unexpected success sequence: written=%q keys=%v err=%v", written, hotkeyKeys, err)
	}
}

func TestRemotePasteKeys(t *testing.T) {
	for platform, expected := range map[string]string{"windows": "ControlLeft,KeyV", "darwin": "MetaLeft,KeyV"} {
		if actual := strings.Join(remotePasteKeys(platform), ","); actual != expected {
			t.Fatalf("%s: %s != %s", platform, actual, expected)
		}
	}
}
