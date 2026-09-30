package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var applicationDeployMu sync.Mutex

type applicationDeployment struct {
	ID        string   `json:"id"`
	Extension string   `json:"extension"`
	Digest    string   `json:"digest"`
	Args      []string `json:"args"`
}

func (a *Agent) deployApplication(request applicationDeployment) {
	report := func(status, detail string) {
		message, _ := json.Marshal(map[string]interface{}{"action": "deployment_result", "data": map[string]string{"id": request.ID, "status": status, "detail": detail}})
		_ = a.writeTextMessage(message)
	}
	if !applicationDeployMu.TryLock() {
		report("failed", "Instalasi lain sedang berjalan; tidak dicoba ulang otomatis.")
		return
	}
	defer applicationDeployMu.Unlock()
	if runtime.GOOS != "windows" || len(request.ID) != 64 || len(request.Digest) != 64 || (request.Extension != ".msi" && request.Extension != ".exe") || len(request.Args) > 64 {
		report("failed", "Permintaan deployment tidak valid")
		return
	}
	if request.Extension == ".msi" && len(request.Args) > 0 || request.Extension == ".exe" && len(request.Args) == 0 {
		report("failed", "Argumen installer tidak valid")
		return
	}
	for _, arg := range request.Args {
		if len(arg) > 2048 || strings.ContainsRune(arg, 0) {
			report("failed", "Argumen tidak valid")
			return
		}
	}
	report("running", "Mengunduh dan memverifikasi installer")
	dir, err := os.MkdirTemp("", "RemoteDesk-deploy-")
	if err != nil {
		report("failed", err.Error())
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "installer"+request.Extension)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.cfg.ServerURL, "/")+"/api/deployment-package", nil)
	if err != nil {
		report("failed", "URL server tidak valid")
		return
	}
	req.Header.Set("Authorization", "Bearer "+request.ID)
	client := &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect download ditolak") }}
	response, err := client.Do(req)
	if err != nil {
		report("failed", "Download gagal")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		report("failed", fmt.Sprintf("Download HTTP %d", response.StatusCode))
		return
	}
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		report("failed", err.Error())
		return
	}
	digest := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(output, digest), io.LimitReader(response.Body, (100<<20)+1))
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil || size == 0 || size > 100<<20 || hex.EncodeToString(digest.Sum(nil)) != request.Digest {
		report("failed", "Ukuran/checksum installer tidak valid")
		return
	}
	report("running", "Installer berjalan; jangan kirim ulang")
	var command *exec.Cmd
	if request.Extension == ".msi" {
		command = exec.CommandContext(ctx, "msiexec.exe", "/i", path, "/qn", "/norestart")
	} else {
		command = exec.CommandContext(ctx, path, request.Args...)
	}
	command.Dir = dir
	err = command.Run()
	if ctx.Err() != nil {
		report("unknown", "Timeout; proses turunan mungkin masih berjalan. Periksa target sebelum mencoba ulang.")
		return
	}
	code := command.ProcessState
	if code == nil {
		report("failed", "Installer tidak dapat dijalankan")
		return
	}
	status, detail := deploymentExitStatus(code.ExitCode())
	report(status, detail)
}

func deploymentExitStatus(code int) (string, string) {
	switch code {
	case 0:
		return "succeeded", "Installer exit code 0"
	case 3010:
		return "reboot_required", "Installer berhasil; restart diperlukan, tidak dijalankan otomatis"
	case 1641:
		return "reboot_required", "Installer memulai restart (exit code 1641)"
	default:
		return "failed", fmt.Sprintf("Installer exit code %d", code)
	}
}
