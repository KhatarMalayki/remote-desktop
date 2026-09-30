package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoteDestinationNoOverwrite(t *testing.T) {
	directory := t.TempDir()
	for attempt := 0; attempt < 2; attempt++ {
		var transfer remoteFileTransfer
		if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "empty.txt", Destination: directory}); err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(nil)
		result, err := transfer.handle(remoteCommand{Type: "file_end", Digest: hex.EncodeToString(digest[:])})
		if attempt == 0 && (err != nil || result["path"] != filepath.Join(directory, "empty.txt")) {
			t.Fatalf("publish: %v %v", result, err)
		}
		if attempt == 1 && err == nil {
			t.Fatal("overwrote existing file")
		}
		transfer.abort()
	}
	if _, err := os.Stat(filepath.Join(directory, "empty.txt")); err != nil {
		t.Fatal(err)
	}
	var transfer remoteFileTransfer
	if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "file.txt", Destination: "relative"}); err == nil {
		t.Fatal("accepted relative folder")
	}
	result, err := transfer.handle(remoteCommand{Type: "file_list", Destination: directory})
	if err != nil || result["path"] != directory {
		t.Fatalf("listing: %v %v", result, err)
	}
}
