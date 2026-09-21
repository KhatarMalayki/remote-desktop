//go:build windows

package main

import (
	"fmt"
	"os"
	"testing"
)

func TestSystemWorkerLockRejectsDuplicate(t *testing.T) {
	lockName := fmt.Sprintf(`Local\RemoteDeskAgentSystemWorker-test-%d`, os.Getpid())
	release, acquired, err := acquireNamedWorkerLock(lockName)
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("first worker should acquire the lock")
	}
	defer release()

	releaseDuplicate, duplicateAcquired, err := acquireNamedWorkerLock(lockName)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseDuplicate()
	if duplicateAcquired {
		t.Fatal("second worker should be rejected as a duplicate")
	}
}
