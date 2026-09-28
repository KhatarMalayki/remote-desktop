//go:build !windows

package agent

import (
	"errors"

	"github.com/user/remote-desktop/internal/models"
)

func listProcesses() ([]models.ProcessInfo, error) {
	return nil, errors.New("process inspection is only supported on Windows")
}
