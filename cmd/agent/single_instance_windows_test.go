//go:build windows

package main

import "testing"

func TestSystemWorkerLockRejectsDuplicate(t *testing.T) {
	release, acquired, err := acquireSystemWorkerLock()
	if err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("first worker should acquire the lock")
	}
	defer release()

	releaseDuplicate, duplicateAcquired, err := acquireSystemWorkerLock()
	if err != nil {
		t.Fatal(err)
	}
	defer releaseDuplicate()
	if duplicateAcquired {
		t.Fatal("second worker should be rejected as a duplicate")
	}
}
