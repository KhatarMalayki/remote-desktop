//go:build windows

package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/user/remote-desktop/internal/models"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func ManageRustDesk(command models.RustDeskCommand) models.RustDeskResult {
	result := models.RustDeskResult{RequestID: command.RequestID, Operation: command.Operation}
	var binary string
	var err error
	if command.Operation == "install" {
		binary, err = installRustDesk(command.DownloadURL, command.SHA256)
	} else {
		binary = findRustDeskBinary()
		if binary == "" {
			err = fmt.Errorf("RustDesk belum terpasang")
		}
	}
	if err == nil && command.Config != "" {
		var cliConfig string
		cliConfig, err = rustDeskCLIConfig(command.Config)
		if err == nil {
			err = runRustDeskAdmin(binary, "--config", cliConfig)
		}
		if err == nil {
			err = runRustDeskAsConsoleUser(binary, "--config", cliConfig)
		}
		if err == nil {
			err = restartRustDeskService()
		}
	}
	if err == nil && command.Password != "" {
		err = runRustDeskAdmin(binary, "--password", command.Password)
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.RustDeskID = getRustDeskID()
	return result
}

func runRustDeskAsConsoleUser(binary string, args ...string) error {
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == 0xFFFFFFFF {
		return nil
	}
	var token windows.Token
	if err := windows.WTSQueryUserToken(sessionID, &token); err != nil {
		return fmt.Errorf("mengambil token pengguna aktif gagal: %w", err)
	}
	defer token.Close()
	var environment *uint16
	if err := windows.CreateEnvironmentBlock(&environment, token, false); err != nil {
		return fmt.Errorf("membuat environment pengguna aktif gagal: %w", err)
	}
	defer windows.DestroyEnvironmentBlock(environment)

	parts := []string{syscall.EscapeArg(binary)}
	for _, arg := range args {
		parts = append(parts, syscall.EscapeArg(arg))
	}
	command, err := windows.UTF16PtrFromString(strings.Join(parts, " "))
	if err != nil {
		return err
	}
	application, err := windows.UTF16PtrFromString(binary)
	if err != nil {
		return err
	}
	desktop, _ := windows.UTF16PtrFromString("winsta0\\default")
	workingDirectory, _ := windows.UTF16PtrFromString(filepath.Dir(binary))
	startup := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	if err := windows.CreateProcessAsUser(token, application, command, nil, nil, false, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, environment, workingDirectory, startup, &process); err != nil {
		return fmt.Errorf("menerapkan config ke profil pengguna aktif gagal: %w", err)
	}
	defer windows.CloseHandle(process.Thread)
	defer windows.CloseHandle(process.Process)
	if result, err := windows.WaitForSingleObject(process.Process, 45*1000); err != nil || result == uint32(windows.WAIT_TIMEOUT) {
		return fmt.Errorf("RustDesk profil pengguna tidak selesai menerapkan config")
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(process.Process, &exitCode); err != nil || exitCode != 0 {
		return fmt.Errorf("RustDesk profil pengguna gagal menerapkan config (exit %d)", exitCode)
	}
	return nil
}

func restartRustDeskService() error {
	manager, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("membuka Windows service manager gagal: %w", err)
	}
	defer manager.Disconnect()
	service, err := manager.OpenService("RustDesk")
	if err != nil {
		return fmt.Errorf("Windows service RustDesk tidak ditemukan: %w", err)
	}
	defer service.Close()

	status, err := service.Query()
	if err != nil {
		return fmt.Errorf("membaca status service RustDesk gagal: %w", err)
	}
	if status.State != svc.Stopped {
		_, _ = service.Control(svc.Stop)
		if err := waitForRustDeskService(service, svc.Stopped, 30*time.Second); err != nil {
			return err
		}
	}
	if err := service.Start(); err != nil {
		return fmt.Errorf("menyalakan service RustDesk gagal: %w", err)
	}
	if err := waitForRustDeskService(service, svc.Running, 30*time.Second); err != nil {
		return err
	}
	// Give the mediator time to register the ID with the self-hosted hbbs.
	time.Sleep(5 * time.Second)
	return nil
}

func waitForRustDeskService(service *mgr.Service, expected svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := service.Query()
		if err == nil && status.State == expected {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("service RustDesk tidak mencapai status %d", expected)
}

func findRustDeskBinary() string {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "RustDesk", "RustDesk.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "RustDesk", "RustDesk.exe"),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return ""
}

func installRustDesk(downloadURL, expectedSHA256 string) (string, error) {
	if binary := findRustDeskBinary(); binary != "" {
		return binary, nil
	}
	parsed, err := url.Parse(downloadURL)
	if err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", fmt.Errorf("URL installer RustDesk tidak diizinkan")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download RustDesk gagal: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download RustDesk gagal: HTTP %d", resp.StatusCode)
	}
	tempFile, err := os.CreateTemp("", "rustdesk-installer-*.exe")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	hasher := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(tempFile, hasher), io.LimitReader(resp.Body, 100*1024*1024))
	closeErr := tempFile.Close()
	if copyErr != nil || closeErr != nil {
		return "", fmt.Errorf("menyimpan installer RustDesk gagal")
	}
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHash, expectedSHA256) {
		return "", fmt.Errorf("checksum installer RustDesk tidak cocok")
	}
	installCtx, installCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer installCancel()
	if output, err := exec.CommandContext(installCtx, tempPath, "--silent-install").CombinedOutput(); err != nil {
		return "", fmt.Errorf("instalasi RustDesk gagal: %s", strings.TrimSpace(string(output)))
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if binary := findRustDeskBinary(); binary != "" {
			_ = runRustDeskAdmin(binary, "--install-service")
			return binary, nil
		}
		time.Sleep(2 * time.Second)
	}
	return "", fmt.Errorf("RustDesk selesai dijalankan tetapi file instalasi belum ditemukan")
}

func runRustDeskAdmin(binary string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("perintah RustDesk gagal: %s", strings.TrimSpace(string(output)))
	}
	if strings.Contains(strings.ToLower(string(output)), "settings are disabled") {
		return fmt.Errorf("pengaturan RustDesk dikunci oleh client")
	}
	return nil
}
