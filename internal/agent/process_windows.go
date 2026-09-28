//go:build windows

package agent

import (
	"encoding/json"
	"os/exec"

	"github.com/user/remote-desktop/internal/models"
)

func listProcesses() ([]models.ProcessInfo, error) {
	command := `@(Get-Process | Sort-Object WorkingSet64 -Descending | Select-Object -First 50 @{N='pid';E={$_.Id}},@{N='name';E={$_.ProcessName}},@{N='memory_bytes';E={$_.WorkingSet64}}) | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command).Output()
	if err != nil {
		return nil, err
	}
	var processes []models.ProcessInfo
	if err := json.Unmarshal(out, &processes); err != nil {
		return nil, err
	}
	return enrichProcesses(processes), nil
}
