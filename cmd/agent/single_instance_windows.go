//go:build windows

package main

import (
	"errors"

	"golang.org/x/sys/windows"
)

// acquireSystemWorkerLock prevents an updater-started worker and the Windows
// service supervisor from running simultaneously during an upgrade. The lock
// lives in the active console session, which is exactly where both workers run.
func acquireSystemWorkerLock() (release func(), acquired bool, err error) {
	return acquireNamedWorkerLock(`Local\RemoteDeskAgentSystemWorker`)
}

func acquireNamedWorkerLock(lockName string) (release func(), acquired bool, err error) {
	name, err := windows.UTF16PtrFromString(lockName)
	if err != nil {
		return nil, false, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(handle)
		return func() {}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return func() { _ = windows.CloseHandle(handle) }, true, nil
}
