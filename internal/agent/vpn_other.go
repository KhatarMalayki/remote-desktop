//go:build !windows

package agent

import (
	"fmt"
	"github.com/user/remote-desktop/internal/vpn"
)

func vpnPlatformPrepare(serverURL, apiKey string)(string,string,error){return "","",fmt.Errorf("pilot VPN client hanya tersedia di Windows")}
func vpnPlatformConnect(vpn.Config,string,string,vpnLease)error{return fmt.Errorf("pilot VPN client hanya tersedia di Windows")}
func vpnPlatformDisconnect()error{return nil}
func vpnPlatformRunning()(bool,error){return false,nil}
func vpnWriteLease(vpnLease)error{return fmt.Errorf("platform tidak didukung")}
func initVPNTray(){}
func RunVPNTray()error{return fmt.Errorf("tray pilot hanya tersedia di Windows")}
func RunVPNWatchdog()error{return fmt.Errorf("watchdog pilot hanya tersedia di Windows")}
