//go:build !windows

package agent

type remoteDesktop struct{}

func newRemoteDesktop() *remoteDesktop { return &remoteDesktop{} }
func (*remoteDesktop) prepare() error  { return nil }
func (*remoteDesktop) close()          {}
