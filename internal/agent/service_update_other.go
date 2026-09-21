//go:build !windows

package agent

func scheduleServiceManagedUpdate(_, _, _ string) error { return nil }
