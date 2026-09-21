//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// scheduleServiceManagedUpdate launches a helper outside rd-agent.exe. The
// helper is required because Windows will not rename the service image while
// either the service or its SYSTEM console worker still has it open.
func scheduleServiceManagedUpdate(exePath, newPath, oldPath string) error {
	scriptPath := filepath.Join(filepath.Dir(exePath), "rd-agent-update.ps1")
	if err := os.WriteFile(scriptPath, []byte(serviceUpdateScript(exePath, newPath, oldPath, scriptPath)), 0700); err != nil {
		return fmt.Errorf("write update helper: %w", err)
	}
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	cmd.Dir = filepath.Dir(exePath)
	if err := cmd.Start(); err != nil {
		_ = os.Remove(scriptPath)
		return fmt.Errorf("start update helper: %w", err)
	}
	return nil
}

func serviceUpdateScript(exePath, newPath, oldPath, scriptPath string) string {
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
	return `$ErrorActionPreference = 'Stop'
$serviceName = 'RemoteDeskAgent'
$exePath = ` + quote(exePath) + `
$newPath = ` + quote(newPath) + `
$oldPath = ` + quote(oldPath) + `
$scriptPath = ` + quote(scriptPath) + `
$logPath = Join-Path (Split-Path -Parent $exePath) 'update.log'
function Write-UpdateLog([string]$message) {
    Add-Content -LiteralPath $logPath -Value ((Get-Date -Format o) + ' ' + $message)
}
try {
    Write-UpdateLog 'dashboard update helper started'
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    $deadline = (Get-Date).AddSeconds(20)
    while ((Get-Date) -lt $deadline) {
        $running = @(Get-Process -Name 'rd-agent' -ErrorAction SilentlyContinue)
        if ($running.Count -eq 0) { break }
        $running | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 400
    }
    if (@(Get-Process -Name 'rd-agent' -ErrorAction SilentlyContinue).Count -gt 0) {
        throw 'Proses agent lama masih berjalan setelah service dihentikan.'
    }
    Remove-Item -LiteralPath $oldPath -Force -ErrorAction SilentlyContinue
    Move-Item -LiteralPath $exePath -Destination $oldPath -Force
    try {
        Move-Item -LiteralPath $newPath -Destination $exePath -Force
    } catch {
        Move-Item -LiteralPath $oldPath -Destination $exePath -Force -ErrorAction SilentlyContinue
        throw
    }
    Start-Service -Name $serviceName
    Write-UpdateLog 'dashboard update succeeded; service restarted'
} catch {
    Write-UpdateLog ('dashboard update FAILED: ' + $_.Exception.Message)
} finally {
    Remove-Item -LiteralPath $scriptPath -Force -ErrorAction SilentlyContinue
}
`
}
