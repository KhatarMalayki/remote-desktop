//go:build !windows

package agent

import (
	"runtime"

	"github.com/user/remote-desktop/internal/models"
)

func ManageRustDesk(command models.RustDeskCommand) models.RustDeskResult {
	return models.RustDeskResult{RequestID: command.RequestID, Operation: command.Operation, Error: "RustDesk management is not supported on " + runtime.GOOS}
}
