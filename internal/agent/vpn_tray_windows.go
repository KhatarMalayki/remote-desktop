//go:build windows

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	wmTrayMessage = win.WM_APP + 7
	cmdDisconnect = 1001
	cmdExitTray   = 1002
)

func initVPNTray() {
	if os.Getenv("RD_VPN_NO_TRAY") == "true" {
		return
	}
	executable, err := os.Executable()
	if err != nil {
		return
	}
	script := fmt.Sprintf(`$path = [Console]::In.ReadToEnd(); $name = 'RemoteDeskVPNTray'; if (!(Get-ItemProperty -Path HKCU:\Software\Microsoft\Windows\CurrentVersion\Run -Name $name -ErrorAction SilentlyContinue)) { Set-ItemProperty -Path HKCU:\Software\Microsoft\Windows\CurrentVersion\Run -Name $name -Value ('"'+$path+'" --vpn-tray') }`)
	_, _ = vpnPowershell(script, executable)
}

func RunVPNTray() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
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
	tip, _ := windows.UTF16FromString("RemoteDesk VPN: Disconnect Darurat")
	data := win.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(win.NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           win.NIF_MESSAGE | win.NIF_ICON | win.NIF_TIP,
		UCallbackMessage: wmTrayMessage,
		HIcon:            icon,
	}
	copy(data.SzTip[:], tip)
	if !win.Shell_NotifyIcon(win.NIM_ADD, &data) {
		return fmt.Errorf("add tray icon failed")
	}
	defer win.Shell_NotifyIcon(win.NIM_DELETE, &data)
	win.SetTimer(hwnd, 1, 3000, 0)
	var msg win.MSG
	for win.GetMessage(&msg, 0, 0, 0) > 0 {
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}
	return nil
}

func trayWndProc(hwnd win.HWND, msg uint32, wparam uintptr, lparam uintptr) uintptr {
	switch msg {
	case wmTrayMessage:
		switch lparam {
		case win.WM_RBUTTONUP, win.WM_LBUTTONUP:
			menu := win.CreatePopupMenu()
			running, _ := vpnPlatformRunning()
			stateText := "VPN: Nonaktif"
			if running {
				stateText = "VPN: Aktif (Split Tunnel Pilot)"
			}
			disconnectText := "Disconnect VPN Sekarang (Pengaman)"
			if !running {
				disconnectText = "VPN Sudah Nonaktif"
			}
			appendMenuItem(menu, 0, stateText, true)
			appendMenuSeparator(menu)
			appendMenuItem(menu, cmdDisconnect, disconnectText, !running)
			appendMenuItem(menu, cmdExitTray, "Tutup Tray", false)
			var point win.POINT
			win.GetCursorPos(&point)
			win.SetForegroundWindow(hwnd)
			chosen := win.TrackPopupMenu(menu, win.TPM_RETURNCMD|win.TPM_NONOTIFY, point.X, point.Y, 0, hwnd, nil)
			win.DestroyMenu(menu)
			switch chosen {
			case cmdDisconnect:
				_ = vpnEmergencyDisconnect()
			case cmdExitTray:
				win.DestroyWindow(hwnd)
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
	manager, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(manager)
	name, _ := windows.UTF16PtrFromString(vpnService)
	handle, err := windows.OpenService(manager, name, windows.SERVICE_STOP|windows.SERVICE_QUERY_STATUS)
	if err == nil {
		defer windows.CloseServiceHandle(handle)
		var status windows.SERVICE_STATUS
		_ = windows.ControlService(handle, windows.SERVICE_CONTROL_STOP, &status)
	}
	leasePath := filepath.Join(vpnDirectory(), "lease.json")
	_ = os.Remove(leasePath)
	_ = exec.Command("net.exe", "stop", vpnService, "/y").Run()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if running, _ := vpnPlatformRunning(); !running {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("disconnect darurat belum terkonfirmasi")
}
