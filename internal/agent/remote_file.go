package agent

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const remoteFileLimit = 100 * 1024 * 1024
const remoteChunkLimit = 48 * 1024

type remoteFileTransfer struct {
	dir     string
	file    *os.File
	name    string
	size    int64
	written int64
	digest  hash.Hash
	updated time.Time
}

func validRemoteFilename(name string) bool {
	if name == "" || len(name) > 180 || strings.ContainsAny(name, "/\\:<>\"|?*\x00") || strings.TrimRight(name, ". ") != name || name == "." || name == ".." {
		return false
	}
	for _, char := range name {
		if char < 32 {
			return false
		}
	}
	stem := strings.TrimRight(strings.ToUpper(strings.SplitN(name, ".", 2)[0]), " ")
	if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || stem == "CONIN$" || stem == "CONOUT$" {
		return false
	}
	if strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT") {
		suffix := []rune(stem[3:])
		if len(suffix) == 1 && strings.ContainsRune("123456789¹²³", suffix[0]) {
			return false
		}
	}
	if !filepath.IsLocal(name) {
		return false
	}
	return true
}

func (transfer *remoteFileTransfer) abort() {
	if transfer.file != nil {
		_ = transfer.file.Close()
	}
	if transfer.dir != "" {
		_ = os.Remove(filepath.Join(transfer.dir, transfer.name))
		_ = os.Remove(transfer.dir)
	}
	*transfer = remoteFileTransfer{}
}

func (transfer *remoteFileTransfer) handle(command remoteCommand) (map[string]interface{}, error) {
	switch command.Type {
	case "file_start":
		if transfer.file != nil {
			return nil, fmt.Errorf("transfer masih berlangsung")
		}
		if !validRemoteFilename(command.Name) || command.Size < 0 || command.Size > remoteFileLimit {
			return nil, fmt.Errorf("nama file tidak valid atau ukuran melebihi 100 MiB")
		}
		dir, err := os.MkdirTemp("", "RemoteDesk-received-")
		if err != nil {
			return nil, err
		}
		transfer.dir, transfer.name = dir, command.Name
		root, err := os.OpenRoot(dir)
		if err != nil {
			transfer.abort()
			return nil, err
		}
		defer root.Close()
		transfer.file, err = root.OpenFile(command.Name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			transfer.abort()
			return nil, err
		}
		transfer.size, transfer.digest = command.Size, sha256.New()
	case "file_chunk":
		if transfer.file == nil {
			return nil, fmt.Errorf("transfer belum dimulai")
		}
		if command.Offset != transfer.written || len(command.Data) > base64.StdEncoding.EncodedLen(remoteChunkLimit) {
			return nil, fmt.Errorf("offset atau ukuran chunk tidak valid")
		}
		data, err := base64.StdEncoding.Strict().DecodeString(command.Data)
		if err != nil || len(data) == 0 || transfer.written+int64(len(data)) > transfer.size {
			return nil, fmt.Errorf("data transfer tidak valid")
		}
		written, err := transfer.file.Write(data)
		if err != nil {
			return nil, err
		}
		if written != len(data) {
			return nil, fmt.Errorf("file write tidak lengkap")
		}
		_, _ = transfer.digest.Write(data)
		transfer.written += int64(written)
	case "file_end":
		if transfer.file == nil || transfer.written != transfer.size {
			return nil, fmt.Errorf("transfer belum lengkap")
		}
		if command.Digest != hex.EncodeToString(transfer.digest.Sum(nil)) {
			return nil, fmt.Errorf("checksum file tidak cocok")
		}
		if err := transfer.file.Sync(); err != nil {
			return nil, err
		}
		if err := transfer.file.Close(); err != nil {
			return nil, err
		}
		path := filepath.Join(transfer.dir, transfer.name)
		*transfer = remoteFileTransfer{}
		return map[string]interface{}{"type": "file_complete", "path": path}, nil
	case "file_cancel":
		transfer.abort()
	default:
		return nil, fmt.Errorf("perintah transfer tidak dikenal")
	}
	transfer.updated = time.Now()
	return map[string]interface{}{"type": "file_ack", "offset": transfer.written}, nil
}
