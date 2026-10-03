//go:build !windows && (!darwin || !cgo)

package agent

type remoteDesktop struct{}

func newRemoteDesktop() *remoteDesktop { return &remoteDesktop{} }
func (*remoteDesktop) prepare() error  { return nil }
func (*remoteDesktop) close()          {}
