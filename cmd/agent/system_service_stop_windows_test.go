//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestStoppedSupervisorTerminatesOwnedWorker(t *testing.T) {
	command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 60")
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer command.Process.Kill()
	process, err := windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_TERMINATE, false, uint32(command.Process.Pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(process)
	stop := make(chan struct{})
	close(stop)
	started := time.Now()
	if err := waitConsoleWorker(process, stop); err != nil {
		t.Fatal(err)
	}
	_ = command.Wait()
	if time.Since(started) > 10*time.Second {
		t.Fatal("worker did not stop promptly")
	}
}
