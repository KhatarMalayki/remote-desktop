//go:build !windows

package main

import "log"

func runSystemService(_ string) {
	log.Fatal("RemoteDesk system service is only supported on Windows")
}
