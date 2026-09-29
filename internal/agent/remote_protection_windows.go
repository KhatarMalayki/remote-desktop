//go:build windows

package agent

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var blockRemoteInputCall = user32DLL.NewProc("BlockInput").Call
var privacyClassOnce sync.Once
var privacyClassError error
var privacyWindowProc = syscall.NewCallback(func(window win.HWND, message uint32, word, param uintptr) uintptr {
	return win.DefWindowProc(window, message, word, param)
})

type remotePrivacyWindow struct {
	stop     chan struct{}
	done     chan struct{}
	deadline atomic.Int64
}

type remoteProtection struct {
	blocked bool
	desktop string
	privacy *remotePrivacyWindow
}

func (protection *remoteProtection) active() bool {
	return protection.blocked || protection.privacy != nil
}

func (protection *remoteProtection) state() map[string]interface{} {
	return map[string]interface{}{"type": "protection_state", "blocked": protection.blocked, "privacy": protection.privacy != nil}
}

func (protection *remoteProtection) desktopChanged() bool {
	if protection.privacy != nil {
		select {
		case <-protection.privacy.done:
			return true
		default:
		}
		protection.privacy.deadline.Store(time.Now().Add(20 * time.Second).UnixNano())
	}
	return protection.active() && !strings.EqualFold(activeInputDesktopName(), protection.desktop)
}

func (protection *remoteProtection) set(kind string, enabled bool) error {
	if enabled {
		desktop := activeInputDesktopName()
		if !strings.EqualFold(desktop, "Default") {
			return fmt.Errorf("proteksi hanya tersedia pada desktop Windows normal, bukan lock screen/UAC")
		}
		protection.desktop = desktop
	}
	switch kind {
	case "block_input":
		if !enabled && protection.privacy != nil {
			return fmt.Errorf("matikan privacy terlebih dahulu")
		}
		return protection.block(enabled)
	case "privacy_mode":
		if enabled && protection.privacy == nil {
			privacy, err := startRemotePrivacy()
			if err != nil {
				return err
			}
			if err := protection.block(true); err != nil {
				close(privacy.stop)
				<-privacy.done
				return err
			}
			protection.privacy = privacy
		} else if !enabled && protection.privacy != nil {
			close(protection.privacy.stop)
			<-protection.privacy.done
			protection.privacy = nil
		}
		return nil
	default:
		return fmt.Errorf("proteksi tidak dikenal")
	}
}

func (protection *remoteProtection) block(enabled bool) error {
	if protection.blocked == enabled {
		return nil
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	if result, _, err := blockRemoteInputCall(value); result == 0 {
		return fmt.Errorf("BlockInput gagal: %v", err)
	}
	protection.blocked = enabled
	return nil
}

func (protection *remoteProtection) close() {
	if protection.privacy != nil {
		close(protection.privacy.stop)
		<-protection.privacy.done
		protection.privacy = nil
	}
	if err := protection.block(false); err != nil {
		log.Printf("[remote] input release failed (Ctrl+Alt+Del locally releases input): %v", err)
	}
}

func startRemotePrivacy() (*remotePrivacyWindow, error) {
	if windows.RtlGetVersion().BuildNumber < 19041 {
		return nil, fmt.Errorf("privacy membutuhkan Windows 10 2004 atau lebih baru")
	}
	var composed int32
	result, _, _ := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmIsCompositionEnabled").Call(uintptr(unsafe.Pointer(&composed)))
	if result != 0 || composed == 0 {
		return nil, fmt.Errorf("privacy membutuhkan desktop composition")
	}
	privacy := &remotePrivacyWindow{stop: make(chan struct{}), done: make(chan struct{})}
	privacy.deadline.Store(time.Now().Add(20 * time.Second).UnixNano())
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(privacy.done)
		desktop := newRemoteDesktop()
		defer desktop.close()
		if err := desktop.prepare(); err != nil {
			ready <- err
			return
		}
		className := syscall.StringToUTF16Ptr("RemoteDeskPrivacy")
		privacyClassOnce.Do(func() {
			class := win.WNDCLASSEX{LpfnWndProc: privacyWindowProc, HInstance: win.GetModuleHandle(nil), HbrBackground: win.HBRUSH(win.GetStockObject(win.BLACK_BRUSH)), LpszClassName: className}
			class.CbSize = uint32(unsafe.Sizeof(class))
			if win.RegisterClassEx(&class) == 0 {
				privacyClassError = fmt.Errorf("gagal mendaftarkan privacy window")
			}
		})
		if privacyClassError != nil {
			ready <- privacyClassError
			return
		}
		window := win.CreateWindowEx(win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE|win.WS_EX_TRANSPARENT|win.WS_EX_LAYERED, className, syscall.StringToUTF16Ptr("RemoteDesk privacy"), win.WS_POPUP,
			win.GetSystemMetrics(win.SM_XVIRTUALSCREEN), win.GetSystemMetrics(win.SM_YVIRTUALSCREEN), win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN), win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN), 0, 0, win.GetModuleHandle(nil), nil)
		if window == 0 {
			ready <- fmt.Errorf("gagal membuat privacy window")
			return
		}
		defer win.DestroyWindow(window)
		if ok, _, err := user32DLL.NewProc("SetLayeredWindowAttributes").Call(uintptr(window), 0, 255, 2); ok == 0 {
			ready <- fmt.Errorf("privacy opacity: %v", err)
			return
		}
		if ok, _, err := user32DLL.NewProc("SetWindowDisplayAffinity").Call(uintptr(window), 0x11); ok == 0 {
			ready <- fmt.Errorf("privacy capture exclusion: %v", err)
			return
		}
		win.ShowWindow(window, win.SW_SHOWNOACTIVATE)
		ready <- nil
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-privacy.stop:
				return
			case <-ticker.C:
				if time.Now().UnixNano() > privacy.deadline.Load() || !strings.EqualFold(activeInputDesktopName(), "Default") {
					return
				}
				var message win.MSG
				for win.PeekMessage(&message, 0, 0, 0, win.PM_REMOVE) {
					win.TranslateMessage(&message)
					win.DispatchMessage(&message)
				}
				if !win.SetWindowPos(window, win.HWND_TOPMOST, win.GetSystemMetrics(win.SM_XVIRTUALSCREEN), win.GetSystemMetrics(win.SM_YVIRTUALSCREEN), win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN), win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN), win.SWP_NOACTIVATE|win.SWP_SHOWWINDOW) {
					return
				}
			}
		}
	}()
	if err := <-ready; err != nil {
		<-privacy.done
		return nil, err
	}
	return privacy, nil
}
