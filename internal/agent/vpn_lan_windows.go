//go:build windows

package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/remote-desktop/internal/vpn"
	"golang.org/x/sys/windows"
)

type vpnLANInterface struct {
	Index      int    `json:"index"`
	GUID       string `json:"guid"`
	Forwarding string `json:"forwarding"`
}

type vpnLANState struct {
	Name       string            `json:"name"`
	Network    string            `json:"network"`
	Subnet     string            `json:"subnet"`
	External   string            `json:"external"`
	Interfaces []vpnLANInterface `json:"interfaces"`
}

var vpnLANPowerShell = vpnPowershell
var vpnLANSecureDirectory = secureVPNDirectory

const vpnLANInspectScript = `
$request = [Console]::In.ReadToEnd() | ConvertFrom-Json
if (@(Get-NetNat -ErrorAction Stop).Count -ne 0) { throw 'NAT already in use' }
if (@(Get-Service SharedAccess -ErrorAction SilentlyContinue | Where-Object Status -eq Running).Count -ne 0) { throw 'Internet Connection Sharing in use' }
$tunnel = Get-NetIPInterface -InterfaceAlias RemoteDeskPilot -AddressFamily IPv4 -PolicyStore ActiveStore -ErrorAction Stop
$lan = Get-NetIPInterface -InterfaceIndex $request.index -AddressFamily IPv4 -PolicyStore ActiveStore -ErrorAction Stop
$interfaces = @($lan, $tunnel) | ForEach-Object {
  $adapter = Get-NetAdapter -InterfaceIndex $_.InterfaceIndex -IncludeHidden -ErrorAction Stop
  @{index=[int]$_.InterfaceIndex; guid=([guid]$adapter.InterfaceGuid).ToString('B'); forwarding=$_.Forwarding.ToString()}
}
ConvertTo-Json -InputObject @($interfaces) -Compress
`

const vpnLANApplyScript = `
$state = [Console]::In.ReadToEnd() | ConvertFrom-Json
if (@(Get-NetNat -ErrorAction Stop).Count -ne 0) { throw 'NAT already in use' }
foreach ($item in $state.interfaces) {
  $adapter = Get-NetAdapter -InterfaceIndex $item.index -IncludeHidden -ErrorAction Stop
  if (([guid]$adapter.InterfaceGuid).ToString('B') -ne $item.guid) { throw 'Adapter changed' }
  Set-NetIPInterface -InterfaceIndex $item.index -AddressFamily IPv4 -PolicyStore ActiveStore -Forwarding Enabled -ErrorAction Stop
}
New-NetNat -Name $state.name -InternalIPInterfaceAddressPrefix $state.network -ExternalIPInterfaceAddressPrefix $state.external -ErrorAction Stop | Out-Null
`

const vpnLANCleanupScript = `
$state = [Console]::In.ReadToEnd() | ConvertFrom-Json
$nat = @(Get-NetNat -ErrorAction Stop | Where-Object Name -eq $state.name)
foreach ($entry in $nat) {
  if ($entry.InternalIPInterfaceAddressPrefix -ne $state.network -or $entry.ExternalIPInterfaceAddressPrefix -ne $state.external) { throw 'NAT ownership mismatch' }
  $entry | Remove-NetNat -Confirm:$false -ErrorAction Stop
}
foreach ($item in $state.interfaces) {
  $adapter = @(Get-NetAdapter -IncludeHidden -ErrorAction Stop | Where-Object { $_.InterfaceIndex -eq $item.index })
  if ($adapter.Count -eq 0) { continue }
  if (([guid]$adapter[0].InterfaceGuid).ToString('B') -ne $item.guid) { throw 'Adapter ownership mismatch' }
  Set-NetIPInterface -InterfaceIndex $item.index -AddressFamily IPv4 -PolicyStore ActiveStore -Forwarding $item.forwarding -ErrorAction Stop
}
`

func (state vpnLANState) validate() error {
	id := strings.TrimPrefix(state.Name, "RemoteDeskPilotLAN-")
	if !strings.HasPrefix(state.Name, "RemoteDeskPilotLAN-") || !vpn.ValidCommand(vpn.Command{ID: id, Expires: time.Now().Add(time.Minute).Unix()}, time.Now()) {
		return fmt.Errorf("identitas cleanup LAN tidak valid")
	}
	network, err := vpn.Network(state.Network)
	if err != nil {
		return err
	}
	subnet, err := vpn.LANPrefix(state.Subnet)
	if err != nil || subnet.Overlaps(network) {
		return fmt.Errorf("subnet cleanup LAN tidak valid")
	}
	external, err := netip.ParsePrefix(state.External)
	if err != nil || external.Bits() != 32 || !subnet.Contains(external.Addr()) {
		return fmt.Errorf("alamat NAT tidak valid")
	}
	if len(state.Interfaces) != 2 || state.Interfaces[0].Index == state.Interfaces[1].Index {
		return fmt.Errorf("interface cleanup tidak valid")
	}
	for _, adapter := range state.Interfaces {
		if _, err := windows.GUIDFromString(adapter.GUID); err != nil {
			return fmt.Errorf("identitas adapter tidak valid")
		}
		if adapter.Index <= 0 || (adapter.Forwarding != "Enabled" && adapter.Forwarding != "Disabled") {
			return fmt.Errorf("status forwarding tidak valid")
		}
	}
	return nil
}

func vpnSetupLAN(config vpn.Config, id string) error {
	if config.AdvertiseLAN == "" {
		return nil
	}
	if err := config.Validate(); err != nil {
		return err
	}
	prefix, _ := vpn.LANPrefix(config.AdvertiseLAN)
	adapters, err := net.Interfaces()
	if err != nil {
		return err
	}
	index := 0
	external := ""
	for _, adapter := range adapters {
		if adapter.Flags&net.FlagUp == 0 || adapter.Flags&net.FlagLoopback != 0 || adapter.Name == vpn.TunnelName {
			continue
		}
		addresses, err := adapter.Addrs()
		if err != nil {
			return err
		}
		for _, address := range addresses {
			local, err := netip.ParsePrefix(address.String())
			if err == nil && local.Masked() == prefix {
				if index != 0 && index != adapter.Index {
					return fmt.Errorf("subnet LAN ada pada beberapa adapter; ditolak")
				}
				index = adapter.Index
				external = local.Addr().String() + "/32"
			}
		}
	}
	if index == 0 {
		return fmt.Errorf("subnet advertise harus persis subnet interface LAN aktif pada gateway")
	}
	request, _ := json.Marshal(map[string]int{"index": index})
	raw, err := vpnLANPowerShell(vpnLANInspectScript, string(request))
	if err != nil {
		return fmt.Errorf("gateway belum siap: NetNat/forwarding tidak tersedia, NAT/ICS sedang dipakai, atau adapter belum siap: %w", err)
	}
	state := vpnLANState{Name: "RemoteDeskPilotLAN-" + id, Network: config.Network, Subnet: config.AdvertiseLAN, External: external}
	if err = json.Unmarshal(raw, &state.Interfaces); err != nil {
		return fmt.Errorf("status adapter LAN tidak dapat dibaca")
	}
	return vpnApplyLANState(state)
}

func vpnApplyLANState(state vpnLANState) error {
	if err := state.validate(); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(vpnDirectory(), "lan.json")); !os.IsNotExist(err) {
		return fmt.Errorf("cleanup LAN sebelumnya belum selesai; tidak ditimpa")
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err = vpnAtomicFile("lan.json", raw); err != nil {
		return err
	}
	if _, err = vpnLANPowerShell(vpnLANApplyScript, string(raw)); err != nil {
		return errors.Join(fmt.Errorf("aktivasi forwarding/NAT LAN gagal: %w", err), vpnCleanupLAN())
	}
	return nil
}

func vpnCleanupLAN() error {
	path := filepath.Join(vpnDirectory(), "lan.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := vpnLANSecureDirectory(); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var state vpnLANState
	if err = json.Unmarshal(raw, &state); err != nil {
		return err
	}
	if err = state.validate(); err != nil {
		return err
	}
	if _, err = vpnLANPowerShell(vpnLANCleanupScript, string(raw)); err != nil {
		return fmt.Errorf("cleanup NAT/forwarding LAN belum selesai: %w", err)
	}
	if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
