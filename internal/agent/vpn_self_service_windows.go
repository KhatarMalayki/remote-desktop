//go:build windows

package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/user/remote-desktop/internal/vpn"
	"golang.org/x/sys/windows"
)

const vpnSelfPipe = `\\.\pipe\RemoteDeskVPNSelfService`

// ponytail: one local request at a time; use overlapped IO if concurrent tray workflows are added.
func createVPNSelfPipe(name string) (windows.Handle, error) {
	security, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;0x100183;;;IU)S:(ML;;NW;;;ME)")
	if err != nil {
		return 0, err
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: security}
	path, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	return windows.CreateNamedPipe(path, windows.PIPE_ACCESS_DUPLEX|windows.FILE_FLAG_FIRST_PIPE_INSTANCE, windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_NOWAIT|windows.PIPE_REJECT_REMOTE_CLIENTS, 1, 4096, 128, 0, &attributes)
}

func (a *Agent) initVPNSelfService() {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || !user.User.Sid.IsWellKnown(windows.WinLocalSystemSid) {
		return
	}
	pipe, err := createVPNSelfPipe(vpnSelfPipe)
	if err != nil {
		log.Printf("VPN self-service: %v", err)
		return
	}
	go func() {
		defer windows.CloseHandle(pipe)
		for {
			err := windows.ConnectNamedPipe(pipe, nil)
			if err == nil || errors.Is(err, windows.ERROR_PIPE_LISTENING) {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
				windows.DisconnectNamedPipe(pipe)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			request, err := readVPNSelfPipe(pipe, 32, 2*time.Second)
			if err == nil && (string(request) == "connect" || string(request) == "disconnect") {
				reply, err := a.requestSelfVPN(string(request))
				if err != nil {
					reply.Detail = err.Error()
				}
				data, _ := json.Marshal(reply)
				var written uint32
				if len(data) <= 4096 && windows.WriteFile(pipe, data, &written, nil) == nil && int(written) == len(data) {
					_, _ = readVPNSelfPipe(pipe, 8, 2*time.Second)
				}
			}
			windows.DisconnectNamedPipe(pipe)
		}
	}()
}

func readVPNSelfPipe(pipe windows.Handle, size int, timeout time.Duration) ([]byte, error) {
	buffer := make([]byte, size)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var read uint32
		err := windows.ReadFile(pipe, buffer, &read, nil)
		if err == nil && read > 0 {
			return buffer[:read], nil
		}
		if err != nil && !errors.Is(err, windows.ERROR_NO_DATA) {
			return nil, err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("layanan VPN lokal tidak merespons")
}

func requestVPNFromTray(pipeName, operation string) (vpn.SelfReply, error) {
	if operation != "connect" && operation != "disconnect" {
		return vpn.SelfReply{}, fmt.Errorf("operasi tidak diizinkan")
	}
	path, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return vpn.SelfReply{}, err
	}
	pipe, err := windows.CreateFile(path, windows.FILE_READ_DATA|windows.FILE_WRITE_DATA|windows.FILE_WRITE_ATTRIBUTES|windows.SYNCHRONIZE, 0, nil, windows.OPEN_EXISTING, windows.SECURITY_SQOS_PRESENT|windows.SECURITY_IDENTIFICATION, 0)
	if err != nil {
		return vpn.SelfReply{}, fmt.Errorf("layanan RemoteDesk belum siap atau sibuk; pastikan agent service terbaru terpasang: %w", err)
	}
	defer windows.CloseHandle(pipe)
	mode := uint32(windows.PIPE_READMODE_MESSAGE | windows.PIPE_NOWAIT)
	if err := windows.SetNamedPipeHandleState(pipe, &mode, nil, nil); err != nil {
		return vpn.SelfReply{}, err
	}
	var written uint32
	if err := windows.WriteFile(pipe, []byte(operation), &written, nil); err != nil {
		return vpn.SelfReply{}, err
	}
	if written != uint32(len(operation)) {
		return vpn.SelfReply{}, fmt.Errorf("permintaan VPN tidak lengkap")
	}
	data, err := readVPNSelfPipe(pipe, 4096, 15*time.Second)
	if err != nil {
		return vpn.SelfReply{}, err
	}
	var reply vpn.SelfReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return reply, err
	}
	_ = windows.WriteFile(pipe, []byte("ack"), &written, nil)
	return reply, nil
}
