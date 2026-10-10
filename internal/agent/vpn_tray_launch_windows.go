//go:build windows

package agent

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func vpnPanelURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" {
		return ""
	}
	switch parsed.Scheme {
	case "ws":
		parsed.Scheme = "http"
	case "wss":
		parsed.Scheme = "https"
	case "http", "https":
	default:
		return ""
	}
	parsed.User, parsed.RawQuery, parsed.Fragment = nil, "", ""
	parsed.Path, parsed.RawPath = "/", ""
	return parsed.String()
}

func initVPNTray(serverURL string) {
	if os.Getenv("RD_VPN_NO_TRAY") == "true" {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}
	if strings.EqualFold(filepath.Base(executable), "rd-agent-uiaccess.exe") {
		executable = filepath.Join(filepath.Dir(executable), "rd-agent.exe")
	}
	args := []string{"--vpn-tray", "--vpn-panel", vpnPanelURL(serverURL)}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		log.Printf("VPN tray: %v", err)
		return
	}
	if !user.User.Sid.IsWellKnown(windows.WinLocalSystemSid) {
		if windows.GetCurrentProcessToken().IsElevated() {
			log.Print("VPN tray: portable elevated process skipped; run --vpn-tray --vpn-panel as normal user")
			return
		}
		command := exec.Command(executable, args...)
		command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := command.Start(); err != nil {
			log.Printf("VPN tray: %v", err)
		} else {
			go command.Wait()
		}
		return
	}
	go func() {
		children := make(map[uint32]windows.Handle)
		for {
			var sessions *windows.WTS_SESSION_INFO
			var count uint32
			if err := windows.WTSEnumerateSessions(0, 0, 1, &sessions, &count); err == nil {
				for _, session := range unsafe.Slice(sessions, count) {
					if session.State != windows.WTSActive || session.SessionID == 0 {
						continue
					}
					if handle := children[session.SessionID]; handle != 0 {
						state, err := windows.WaitForSingleObject(handle, 0)
						if err != nil || state != windows.WAIT_OBJECT_0 {
							continue
						}
						windows.CloseHandle(handle)
						delete(children, session.SessionID)
					}
					handle, err := launchVPNTray(session.SessionID, executable, args)
					if err != nil {
						log.Printf("VPN tray session %d: %v", session.SessionID, err)
					} else {
						children[session.SessionID] = handle
					}
				}
				windows.WTSFreeMemory(uintptr(unsafe.Pointer(sessions)))
			}
			for session, handle := range children {
				if state, err := windows.WaitForSingleObject(handle, 0); err == nil && state == windows.WAIT_OBJECT_0 {
					windows.CloseHandle(handle)
					delete(children, session)
				}
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func launchVPNTray(session uint32, executable string, args []string) (windows.Handle, error) {
	var token windows.Token
	if err := windows.WTSQueryUserToken(session, &token); err != nil {
		return 0, err
	}
	defer token.Close()
	if token.IsElevated() {
		linked, err := token.GetLinkedToken()
		if err != nil {
			return 0, err
		}
		defer linked.Close()
		if linked.IsElevated() {
			return 0, fmt.Errorf("refusing elevated tray token")
		}
		token = linked
	}
	var environment *uint16
	if err := windows.CreateEnvironmentBlock(&environment, token, false); err != nil {
		return 0, err
	}
	defer windows.DestroyEnvironmentBlock(environment)
	application, err := windows.UTF16PtrFromString(executable)
	if err != nil {
		return 0, err
	}
	command, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{executable}, args...)))
	if err != nil {
		return 0, err
	}
	desktop, _ := windows.UTF16PtrFromString(`winsta0\Default`)
	directory, _ := windows.UTF16PtrFromString(filepath.Dir(executable))
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Desktop: desktop}
	var process windows.ProcessInformation
	if err := windows.CreateProcessAsUser(token, application, command, nil, nil, false, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, environment, directory, &startup, &process); err != nil {
		return 0, err
	}
	windows.CloseHandle(process.Thread)
	return process.Process, nil
}
