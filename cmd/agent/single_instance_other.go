//go:build !windows

package main

func acquireSystemWorkerLock() (release func(), acquired bool, err error) {
	return func() {}, true, nil
}
