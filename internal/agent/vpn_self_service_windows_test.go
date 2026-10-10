//go:build windows

package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
	"golang.org/x/sys/windows"
)

func TestVPNSelfServiceNativePipe(t *testing.T) {
	name := fmt.Sprintf(`\\.\pipe\RemoteDeskVPNTest-%d`, os.Getpid())
	pipe, err := createVPNSelfPipe(name)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(pipe)
	duplicate, err := createVPNSelfPipe(name)
	if err == nil {
		windows.CloseHandle(duplicate)
		t.Fatal("second server accepted")
	}
	completed := make(chan error, 1)
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for {
			err := windows.ConnectNamedPipe(pipe, nil)
			if errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
				break
			}
			if (err != nil && !errors.Is(err, windows.ERROR_PIPE_LISTENING)) || time.Now().After(deadline) {
				completed <- fmt.Errorf("connect: %w", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		defer windows.DisconnectNamedPipe(pipe)
		request, err := readVPNSelfPipe(pipe, 32, time.Second)
		if err != nil || string(request) != "connect" {
			completed <- fmt.Errorf("request %q: %v", request, err)
			return
		}
		data, _ := json.Marshal(vpn.SelfReply{Detail: "Ditolak policy admin"})
		var written uint32
		if err := windows.WriteFile(pipe, data, &written, nil); err != nil {
			completed <- err
			return
		}
		ack, err := readVPNSelfPipe(pipe, 8, time.Second)
		if err != nil || string(ack) != "ack" {
			completed <- fmt.Errorf("ack %q: %v", ack, err)
			return
		}
		completed <- nil
	}()
	reply, err := requestVPNFromTray(name, "connect")
	if err != nil {
		t.Fatal(err)
	}
	if reply.Accepted || reply.Detail != "Ditolak policy admin" {
		t.Fatalf("bad reply: %+v", reply)
	}
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
	if _, err := requestVPNFromTray(name, "configure"); err == nil {
		t.Fatal("arbitrary operation accepted")
	}
}
