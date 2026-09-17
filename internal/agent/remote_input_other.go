//go:build !windows

package agent

import (
	"fmt"
	"image"
)

func handleRemoteInput(command remoteCommand, bounds image.Rectangle) error {
	return fmt.Errorf("remote input is only supported on Windows")
}
