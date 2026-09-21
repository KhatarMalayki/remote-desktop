//go:build windows

package main

// This service is intentionally only a supervisor.  Windows services run in
// session 0, which cannot see or control a user's desktop.  The supervisor
// duplicates its LocalSystem token into the active console session and starts
// the normal agent there.  Unlike a per-user scheduled task, that worker keeps
// SYSTEM's access when the console switches to the Winlogon desktop.

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

const remoteDeskServiceName = "RemoteDeskAgent"

func runSystemService(configPath string) {
	// Services have no visible console. Persist supervisor output next to the
	// installed configuration so support can diagnose a console-worker failure
	// without asking an end user to reproduce it in a terminal.
	if logFile, openErr := os.OpenFile(filepath.Join(filepath.Dir(configPath), "service.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); openErr == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	}
	writeServiceDiagnostic(configPath, "service process starting")
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("cannot detect Windows service context: %v", err)
	}
	if !isService {
		log.Printf("[service] %s must be started by Windows Service Control Manager", remoteDeskServiceName)
		return
	}
	if err := svc.Run(remoteDeskServiceName, &remoteDeskService{configPath: configPath}); err != nil {
		log.Fatalf("service failed: %v", err)
	}
}

type remoteDeskService struct{ configPath string }

func (s *remoteDeskService) Execute(_ []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	stop := make(chan struct{})
	var once sync.Once
	writeServiceDiagnostic(s.configPath, "service accepted by SCM; starting console-worker supervisor")
	go superviseConsoleWorker(s.configPath, stop)
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for request := range requests {
		switch request.Cmd {
		case svc.Stop, svc.Shutdown:
			once.Do(func() { close(stop) })
			changes <- svc.Status{State: svc.StopPending}
			return false, 0
		}
	}
	return false, 0
}

func superviseConsoleWorker(configPath string, stop <-chan struct{}) {
	// A worker is intentionally restarted only after it exits.  It prevents the
	// service from creating duplicate agents after lock/unlock transitions.
	for {
		writeServiceDiagnostic(configPath, "console-worker supervisor cycle started")
		select {
		case <-stop:
			return
		default:
		}
		if err := startConsoleSystemWorker(configPath); err != nil {
			log.Printf("[service] console worker not started: %v", err)
			writeServiceDiagnostic(configPath, "console worker failed: "+err.Error())
		}
		select {
		case <-stop:
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func startConsoleSystemWorker(configPath string) error {
	sessionID := windows.WTSGetActiveConsoleSessionId()
	if sessionID == 0xFFFFFFFF {
		return fmt.Errorf("no active console session")
	}
	writeServiceDiagnostic(configPath, fmt.Sprintf("active console session is %d", sessionID))
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	if configPath == "" {
		configPath = filepath.Join(filepath.Dir(exe), "agent.json")
	}
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("agent config unavailable: %w", err)
	}
	workerExe := filepath.Join(filepath.Dir(exe), "rd-agent-uiaccess.exe")
	if _, err := os.Stat(workerExe); err != nil {
		return fmt.Errorf("UIAccess worker unavailable: %w", err)
	}

	// GetCurrentProcessToken is a pseudo-token handle. It is convenient for
	// inspection, but Windows rejects it for DuplicateTokenEx on some service
	// hosts with ERROR_INVALID_HANDLE. Open a real, inheritable-capable handle.
	var current windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ALL_ACCESS, &current); err != nil {
		return fmt.Errorf("open LocalSystem process token: %w", err)
	}
	defer current.Close()
	var token windows.Token
	if err := windows.DuplicateTokenEx(current, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityImpersonation, windows.TokenPrimary, &token); err != nil {
		return fmt.Errorf("duplicate LocalSystem token: %w", err)
	}
	defer token.Close()
	if err := windows.SetTokenInformation(token, windows.TokenSessionId, (*byte)(unsafe.Pointer(&sessionID)), uint32(unsafe.Sizeof(sessionID))); err != nil {
		return fmt.Errorf("assign token to console session: %w", err)
	}

	desktop, err := windows.UTF16PtrFromString("winsta0\\Default")
	if err != nil {
		return err
	}
	workerLog := filepath.Join(filepath.Dir(configPath), "worker.log")
	command, err := windows.UTF16PtrFromString(fmt.Sprintf("\"%s\" --system-worker --config \"%s\" --log-file \"%s\"", workerExe, configPath, workerLog))
	if err != nil {
		return err
	}
	workingDir, err := windows.UTF16PtrFromString(filepath.Dir(workerExe))
	if err != nil {
		return err
	}
	startup := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	if err := windows.CreateProcessAsUser(token, nil, command, nil, nil, false, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, nil, workingDir, startup, &process); err != nil {
		return fmt.Errorf("create SYSTEM console worker: %w", err)
	}
	defer windows.CloseHandle(process.Thread)
	defer windows.CloseHandle(process.Process)
	log.Printf("[service] SYSTEM worker started in console session %d (pid %d)", sessionID, process.ProcessId)
	writeServiceDiagnostic(configPath, fmt.Sprintf("SYSTEM worker started in session %d (pid %d)", sessionID, process.ProcessId))

	// Wait for the worker to end before the supervisor considers a replacement.
	_, _ = windows.WaitForSingleObject(process.Process, windows.INFINITE)
	return nil
}

// startSecureDesktopRelay creates a short-lived child directly on Winlogon.
// The normal worker is started on winsta0\\Default, so it cannot inject input
// on the protected desktop even after OpenInputDesktop/SetThreadDesktop.
func startSecureDesktopRelay(configPath, relayID string) error {
	if relayID == "" {
		return fmt.Errorf("missing relay ID")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	var current windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ALL_ACCESS, &current); err != nil {
		return fmt.Errorf("open LocalSystem worker token: %w", err)
	}
	defer current.Close()
	var token windows.Token
	if err := windows.DuplicateTokenEx(current, windows.TOKEN_ALL_ACCESS, nil, windows.SecurityImpersonation, windows.TokenPrimary, &token); err != nil {
		return fmt.Errorf("duplicate LocalSystem worker token: %w", err)
	}
	defer token.Close()
	desktop, err := windows.UTF16PtrFromString("winsta0\\Winlogon")
	if err != nil {
		return err
	}
	logPath := filepath.Join(filepath.Dir(configPath), "secure-relay.log")
	command, err := windows.UTF16PtrFromString(fmt.Sprintf("\"%s\" --system-worker --secure-relay %s --config \"%s\" --log-file \"%s\"", exe, relayID, configPath, logPath))
	if err != nil {
		return err
	}
	workingDir, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil {
		return err
	}
	startup := &windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	if err := windows.CreateProcessAsUser(token, exeToUTF16(exe), command, nil, nil, false, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, nil, workingDir, startup, &process); err != nil {
		return fmt.Errorf("create Winlogon relay worker: %w", err)
	}
	defer windows.CloseHandle(process.Thread)
	defer windows.CloseHandle(process.Process)
	writeServiceDiagnostic(configPath, fmt.Sprintf("Winlogon relay worker started for %s (pid %d)", relayID, process.ProcessId))
	return nil
}

func exeToUTF16(value string) *uint16 {
	result, _ := windows.UTF16PtrFromString(value)
	return result
}

func writeServiceDiagnostic(configPath, message string) {
	path := filepath.Join(filepath.Dir(configPath), "service.log")
	if configPath == "" {
		path = filepath.Join(os.Getenv("ProgramData"), "RemoteDesk", "Agent", "service.log")
	}
	line := fmt.Sprintf("%s %s\r\n", time.Now().Format(time.RFC3339), message)
	if file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		_, _ = file.WriteString(line)
		_ = file.Close()
	}
}
