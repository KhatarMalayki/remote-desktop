//go:build windows

package agent

import (
	"strings"
	"testing"
)

func TestServiceUpdateHelperStopsServiceBeforeReplacingBinary(t *testing.T) {
	script := serviceUpdateScript(`C:\ProgramData\RemoteDesk\Agent\rd-agent.exe`, `C:\ProgramData\RemoteDesk\Agent\rd-agent.exe.new`, `C:\ProgramData\RemoteDesk\Agent\rd-agent.exe.old`, `C:\ProgramData\RemoteDesk\Agent\rd-agent-update.ps1`)
	for _, expected := range []string{"Stop-Service -Name $serviceName", "Get-Process -Name 'rd-agent'", "Move-Item -LiteralPath $exePath", "Move-Item -LiteralPath $newPath", "Start-Service -Name $serviceName"} {
		if !strings.Contains(script, expected) {
			t.Fatalf("update helper missing %q: %s", expected, script)
		}
	}
	if strings.Index(script, "Stop-Service -Name $serviceName") > strings.Index(script, "Move-Item -LiteralPath $exePath") {
		t.Fatal("helper must stop service before replacing the executable")
	}
}
