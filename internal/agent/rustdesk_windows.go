//go:build windows

package agent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func getRustDeskID() string {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "RustDesk", "RustDesk.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "RustDesk", "RustDesk.exe"),
	}
	if path, err := exec.LookPath("rustdesk.exe"); err == nil {
		candidates = append(candidates, path)
	}
	for _, binary := range candidates {
		if binary == "" {
			continue
		}
		if _, err := os.Stat(binary); err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		output, err := exec.CommandContext(ctx, binary, "--get-id").CombinedOutput()
		cancel()
		if err == nil {
			if id := parseRustDeskID(string(output)); id != "" {
				return id
			}
		}
	}
	return ""
}
