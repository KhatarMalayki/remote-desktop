package agent

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoteFileNames(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../secret", "dir/file", "dir\\file", "C:secret", "file:stream", "a.", "a ", "CON", "con.txt", "LPT1.txt", "COM9", "COM¹.txt", "CON .txt", "bad\n.txt", "a|b"} {
		if validRemoteFilename(name) {
			t.Errorf("accepted unsafe name %q", name)
		}
	}
	for _, name := range []string{"report.csv", "company.pdf", "laporan aset.xlsx", "COM10.txt"} {
		if !validRemoteFilename(name) {
			t.Errorf("rejected safe name %q", name)
		}
	}
}

func TestRemoteFileTransfer(t *testing.T) {
	for _, data := range []string{"", "hello remote"} {
		t.Run(data, func(t *testing.T) {
			var transfer remoteFileTransfer
			defer transfer.abort()
			if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "report.txt", Size: int64(len(data))}); err != nil {
				t.Fatal(err)
			}
			dir := transfer.dir
			t.Cleanup(func() { _ = os.Remove(filepath.Join(dir, "report.txt")); _ = os.Remove(dir) })
			if len(data) > 0 {
				if _, err := transfer.handle(remoteCommand{Type: "file_chunk", Data: base64.StdEncoding.EncodeToString([]byte(data))}); err != nil {
					t.Fatal(err)
				}
			}
			digest := sha256.Sum256([]byte(data))
			response, err := transfer.handle(remoteCommand{Type: "file_end", Digest: hex.EncodeToString(digest[:])})
			if err != nil {
				t.Fatal(err)
			}
			transfer.abort()
			written, err := os.ReadFile(response["path"].(string))
			if err != nil || string(written) != data {
				t.Fatalf("saved data %q: %v", written, err)
			}
		})
	}
}

func TestRemoteFileRejectsInvalidTransfer(t *testing.T) {
	for _, command := range []remoteCommand{
		{Type: "file_start", Name: "other", Size: 1},
		{Type: "file_chunk", Offset: 1, Data: "YQ=="},
		{Type: "file_chunk", Data: "!"},
		{Type: "file_chunk", Data: "YWI="},
		{Type: "file_end"},
	} {
		var transfer remoteFileTransfer
		if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "test.txt", Size: 1}); err != nil {
			t.Fatal(err)
		}
		dir := transfer.dir
		if _, err := transfer.handle(command); err == nil {
			t.Errorf("accepted %#v", command)
		}
		transfer.abort()
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("partial directory remains: %v", err)
		}
	}
	var transfer remoteFileTransfer
	defer transfer.abort()
	for _, size := range []int64{-1, remoteFileLimit + 1} {
		if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "test", Size: size}); err == nil {
			t.Error("accepted invalid size")
		}
	}
	if _, err := transfer.handle(remoteCommand{Type: "file_start", Name: "test", Size: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := transfer.handle(remoteCommand{Type: "file_end", Digest: "wrong"}); err == nil {
		t.Error("accepted bad checksum")
	}
}
