//go:build !windows

package agent

import "fmt"

type remoteProtection struct{}

func (*remoteProtection) set(string, bool) error {
	return fmt.Errorf("proteksi input dan layar hanya tersedia pada Windows")
}
func (*remoteProtection) close()               {}
func (*remoteProtection) active() bool         { return false }
func (*remoteProtection) desktopChanged() bool { return false }
func (*remoteProtection) state() map[string]interface{} {
	return map[string]interface{}{"type": "protection_state", "blocked": false, "privacy": false}
}
