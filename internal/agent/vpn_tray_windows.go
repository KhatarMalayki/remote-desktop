//go:build windows

package agent

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"github.com/user/remote-desktop/internal/vpn"
	"golang.org/x/sys/windows"
)

const (
	wmTrayMessage = win.WM_APP + 7
	cmdDisconnect = 1001
	cmdConnect    = 1003
	cmdAdminPanel = 1004
	wmTrayResult  = win.WM_APP + 8
)

var trayPanel string
var trayIcon win.NOTIFYICONDATA
var trayTaskbarCreated uint32
var trayRequestPending bool
var trayResults = make(chan vpn.SelfReply, 1)

func RunVPNTray(panel string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	if user.User.Sid.IsWellKnown(windows.WinLocalSystemSid) || windows.GetCurrentProcessToken().IsElevated() {
		return fmt.Errorf("VPN tray must run as an unelevated interactive user")
	}
	mutexName, _ := windows.UTF16PtrFromString(`Local\RemoteDeskVPNTray`)
	mutex, err := windows.CreateMutex(nil, false, mutexName)
	if mutex != 0 {
		defer windows.CloseHandle(mutex)
	}
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil
	}
	if err != nil {
		return err
	}
	trayPanel = vpnPanelURL(panel)
	taskbar, _ := windows.UTF16PtrFromString("TaskbarCreated")
	trayTaskbarCreated = win.RegisterWindowMessage(taskbar)
	instance := win.GetModuleHandle(nil)
	className, _ := windows.UTF16PtrFromString("RemoteDeskVPNTrayWindow")
	title, _ := windows.UTF16PtrFromString("RemoteDesk VPN Safety")
	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		HInstance:     instance,
		LpfnWndProc:   syscall.NewCallback(trayWndProc),
		LpszClassName: className,
	}
	if win.RegisterClassEx(&wndClass) == 0 {
		return fmt.Errorf("register window class failed")
	}
	hwnd := win.CreateWindowEx(0, className, title, 0, 0, 0, 0, 0, 0, 0, instance, nil)
	if hwnd == 0 {
		return fmt.Errorf("create tray window failed")
	}
	icon := win.LoadIcon(0, (*uint16)(unsafe.Pointer(uintptr(win.IDI_APPLICATION))))
	trayIcon = win.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(win.NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           win.NIF_MESSAGE | win.NIF_ICON | win.NIF_TIP,
		UCallbackMessage: wmTrayMessage,
		HIcon:            icon,
	}
	refreshVPNTray(true)
	defer win.Shell_NotifyIcon(win.NIM_DELETE, &trayIcon)
	win.SetTimer(hwnd, 1, 3000, 0)
	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
	return nil
}

func trayWndProc(hwnd win.HWND, msg uint32, wparam uintptr, lparam uintptr) uintptr {
	if trayTaskbarCreated != 0 && msg == trayTaskbarCreated {
		refreshVPNTray(true)
		return 0
	}
	switch msg {
	case wmTrayResult:
		result := <-trayResults
		trayRequestPending = false
		text, _ := windows.UTF16PtrFromString(result.Detail)
		title, _ := windows.UTF16PtrFromString("RemoteDesk VPN")
		win.MessageBox(hwnd, text, title, win.MB_OK|win.MB_ICONINFORMATION)
		refreshVPNTray(false)
		return 0
	case win.WM_TIMER:
		refreshVPNTray(false)
		return 0
	case wmTrayMessage:
		switch lparam {
		case win.WM_RBUTTONUP, win.WM_LBUTTONUP:
			menu := win.CreatePopupMenu()
			running, stateErr := vpnPlatformRunning()
			stateText := vpnTrayStatus(running, stateErr)
			appendMenuItem(menu, 0, stateText, true)
			appendMenuSeparator(menu)
			appendMenuItem(menu, cmdConnect, "Connect VPN (sesuai policy admin)", running || trayRequestPending)
			appendMenuItem(menu, cmdDisconnect, "Disconnect / Batalkan Koneksi", trayRequestPending)
			appendMenuItem(menu, cmdAdminPanel, "Panel Admin (opsional)", trayPanel == "")
			var point win.POINT
			win.GetCursorPos(&point)
			win.SetForegroundWindow(hwnd)
			chosen := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_NONOTIFY, point.X, point.Y, 0, hwnd, nil)
			win.DestroyMenu(menu)
			switch chosen {
			case cmdAdminPanel:
				operation, _ := windows.UTF16PtrFromString("open")
				target, _ := windows.UTF16PtrFromString(trayPanel)
				if err := windows.ShellExecute(windows.Handle(hwnd), operation, target, nil, nil, windows.SW_SHOWNORMAL); err != nil {
					trayError(hwnd, err)
				}
			case cmdConnect, cmdDisconnect:
				trayRequestPending = true
				operation := "connect"
				if chosen == cmdDisconnect {
					operation = "disconnect"
				}
				go func() {
					var stopErr error
					if operation == "disconnect" {
						stopErr = vpnEmergencyDisconnect()
					}
					reply, err := requestVPNFromTray(vpnSelfPipe, operation)
					if err != nil {
						reply.Detail = err.Error()
					}
					if stopErr != nil {
						reply.Detail += "\nPenghentian lokal belum terkonfirmasi: " + stopErr.Error()
					}
					trayResults <- reply
					win.PostMessage(hwnd, wmTrayResult, 0, 0)
				}()
			}
		}
		return 0
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	default:
		return win.DefWindowProc(hwnd, msg, wparam, lparam)
	}
}

func vpnTrayStatus(running bool, err error) string {
	if err != nil {
		return "VPN: Status tidak diketahui"
	}
	if !running {
		return "VPN: Nonaktif"
	}
	adapter, err := net.InterfaceByName(vpn.TunnelName)
	if err == nil {
		addresses, err := adapter.Addrs()
		if err == nil {
			for _, address := range addresses {
				if network, ok := address.(*net.IPNet); ok && network.IP.To4() != nil {
					return "VPN: Aktif - " + network.IP.String()
				}
			}
		}
	}
	return "VPN: Aktif - IP belum tersedia"
}

func refreshVPNTray(add bool) {
	running, err := vpnPlatformRunning()
	tip, _ := windows.UTF16FromString("RemoteDesk | " + vpnTrayStatus(running, err))
	clear(trayIcon.SzTip[:])
	copy(trayIcon.SzTip[:len(trayIcon.SzTip)-1], tip)
	if add || !win.Shell_NotifyIcon(win.NIM_MODIFY, &trayIcon) {
		win.Shell_NotifyIcon(win.NIM_ADD, &trayIcon)
	}
}

func trayError(hwnd win.HWND, err error) {
	text, _ := windows.UTF16PtrFromString(err.Error())
	title, _ := windows.UTF16PtrFromString("RemoteDesk VPN")
	win.MessageBox(hwnd, text, title, win.MB_OK|win.MB_ICONERROR)
}

func appendMenuItem(menu win.HMENU, id uint32, text string, disabled bool) {
	title, _ := windows.UTF16FromString(text)
	item := win.MENUITEMINFO{
		CbSize:     uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
		FMask:      win.MIIM_FTYPE | win.MIIM_STRING | win.MIIM_ID | win.MIIM_STATE,
		FType:      win.MFT_STRING,
		WID:        id,
		DwTypeData: &title[0],
		Cch:        uint32(len(title)),
	}
	if disabled {
		item.FState = win.MFS_DISABLED | win.MFS_GRAYED
	}
	win.InsertMenuItem(menu, id, false, &item)
}

func appendMenuSeparator(menu win.HMENU) {
	item := win.MENUITEMINFO{
		CbSize: uint32(unsafe.Sizeof(win.MENUITEMINFO{})),
		FMask:  win.MIIM_FTYPE,
		FType:  win.MFT_SEPARATOR,
	}
	win.InsertMenuItem(menu, 0, false, &item)
}

func vpnEmergencyDisconnect() error {
	service, err := openVPNService(windows.SERVICE_STOP | windows.SERVICE_QUERY_STATUS)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return nil
	}
	if err != nil {
		return err
	}
	defer service.Close()
	handle := service.Handle
	var status windows.SERVICE_STATUS
	if err := windows.ControlService(handle, windows.SERVICE_CONTROL_STOP, &status); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return err
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := windows.QueryServiceStatus(handle, &status); err != nil {
			return err
		}
		if status.CurrentState == windows.SERVICE_STOPPED {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("disconnect darurat belum terkonfirmasi")
}
