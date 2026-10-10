package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/user/remote-desktop/internal/versioncmp"
	"github.com/user/remote-desktop/internal/vpn"
)

type vpnHubCommand struct {
	name string
	args []string
}

func (s *Server) setupVPNFirewall() error {
	for _, rule := range [][]string{
		{"-I", "INPUT", "1", "-i", "rdpilot", "!", "-s", s.vpnPilot.network, "-j", "DROP"},
		{"-I", "FORWARD", "1", "-i", "rdpilot", "-j", "DROP"},
		{"-I", "FORWARD", "1", "-o", "rdpilot", "-j", "DROP"},
		{"-I", "FORWARD", "1", "-i", "rdpilot", "-o", "rdpilot", "-s", s.vpnPilot.network, "-d", s.vpnPilot.network, "-j", "ACCEPT"},
	} {
		if _, err := s.vpnPilot.run("iptables", rule...); err != nil {
			return err
		}
	}
	return nil
}

func vpnSessionActive(session *vpnSession) bool {
	return session.provisioned || len(session.lanCleanup) > 0 || (session.status.State != "disconnected" && session.status.State != "error")
}

func (s *Server) configureVPNLAN(deviceID, subnet string, clients []string, expectedRoute string, session *vpnSession) error {
	if subnet == "" && len(clients) > 0 {
		return fmt.Errorf("pilih subnet sebelum menentukan client LAN")
	}
	if subnet != "" && expectedRoute != "" {
		return fmt.Errorf("gateway tidak boleh sekaligus menjadi client LAN")
	}
	if subnet != "" {
		if _, err := vpn.LANPrefix(subnet); err != nil {
			return err
		}
		if len(clients) != 1 || clients[0] == deviceID {
			return fmt.Errorf("pilih satu perangkat lain sebagai client LAN pilot")
		}
		for _, id := range []string{deviceID, clients[0]} {
			device, err := s.db.GetDevice(id)
			if err != nil || device.OS != "windows" || versioncmp.IsNewer("0.2.69", device.Version) || !s.hub.IsOnline(id) {
				return fmt.Errorf("gateway dan client LAN memerlukan Windows agent online >=0.2.69")
			}
		}
		for id, other := range s.vpnPilot.sessions {
			if vpnSessionActive(other) && (other.config.AdvertiseLAN != "" || id == clients[0]) {
				return fmt.Errorf("disconnect gateway/client lama sebelum mengatur advertise LAN")
			}
		}
		session.config.AdvertiseLAN = subnet
		session.status.AdvertiseLAN = subnet
		session.status.LANClients = append([]string{}, clients...)
		if err := s.checkVPNLANHubRoutes(subnet); err != nil {
			return err
		}
	} else {
		for id, gateway := range s.vpnPilot.sessions {
			if !vpnSessionActive(gateway) || gateway.config.AdvertiseLAN == "" {
				continue
			}
			if len(gateway.status.LANClients) != 1 || gateway.status.LANClients[0] != deviceID {
				return fmt.Errorf("perangkat tidak diizinkan mengakses gateway LAN aktif")
			}
			if expectedRoute != gateway.config.AdvertiseLAN {
				return fmt.Errorf("konfirmasi subnet akses LAN tidak sesuai; refresh status dahulu")
			}
			if gateway.status.State != "connected" || time.Since(gateway.started) > vpn.SessionLimit || time.Since(time.Unix(gateway.status.Updated, 0)) > vpn.Lease || !s.hub.IsOnline(id) {
				return fmt.Errorf("tunggu gateway LAN terhubung dan terverifikasi")
			}
			device, err := s.db.GetDevice(deviceID)
			if err != nil || versioncmp.IsNewer("0.2.69", device.Version) {
				return fmt.Errorf("client LAN memerlukan agent >=0.2.69")
			}
			session.gateway = id
			session.config.Routes = []string{gateway.config.AdvertiseLAN}
			session.status.Routes = append([]string{}, session.config.Routes...)
		}
	}
	if expectedRoute != "" && session.gateway == "" {
		return fmt.Errorf("gateway LAN tidak tersedia; tidak tersambung otomatis")
	}
	return session.config.Validate()
}

func (s *Server) checkVPNLANHubRoutes(subnet string) error {
	raw, err := s.vpnPilot.run("ip", "-j", "-4", "route", "show", "table", "all")
	if err != nil {
		return err
	}
	var routes []struct {
		Dst string `json:"dst"`
	}
	if err = json.Unmarshal(raw, &routes); err != nil {
		return fmt.Errorf("rute hub tidak dapat divalidasi")
	}
	var prefixes []string
	for _, route := range routes {
		if route.Dst == "" || route.Dst == "default" {
			continue
		}
		prefix := route.Dst
		if !strings.Contains(prefix, "/") {
			prefix += "/32"
		}
		prefixes = append(prefixes, prefix)
	}
	return vpn.CheckLANRoutes(subnet, prefixes)
}

func (s *Server) setupVPNLAN(deviceID string, session *vpnSession) error {
	pilot := s.vpnPilot
	if subnet := session.config.AdvertiseLAN; subnet != "" {
		if err := s.checkVPNLANHubRoutes(subnet); err != nil {
			return err
		}
		if _, err := pilot.run("ip", "route", "add", subnet, "dev", "rdpilot"); err != nil {
			return err
		}
		session.lanCleanup = append(session.lanCleanup, vpnHubCommand{"ip", []string{"route", "del", subnet, "dev", "rdpilot"}})
	}
	if session.gateway == "" {
		return nil
	}
	gateway := pilot.sessions[session.gateway]
	if gateway == nil || gateway.status.State != "connected" || !gateway.provisioned || time.Since(gateway.started) > vpn.SessionLimit || time.Since(time.Unix(gateway.status.Updated, 0)) > vpn.Lease || !s.hub.IsOnline(session.gateway) {
		return fmt.Errorf("gateway LAN tidak lagi siap")
	}
	if len(gateway.status.LANClients) != 1 || gateway.status.LANClients[0] != deviceID || len(session.config.Routes) != 1 || gateway.config.AdvertiseLAN != session.config.Routes[0] {
		return fmt.Errorf("izin gateway berubah")
	}
	clientIP := session.config.Address + "/32"
	subnet := gateway.config.AdvertiseLAN
	for _, rule := range [][]string{
		{"-i", "rdpilot", "-o", "rdpilot", "-s", clientIP, "-d", subnet, "-j", "ACCEPT"},
		{"-i", "rdpilot", "-o", "rdpilot", "-s", subnet, "-d", clientIP, "-m", "conntrack", "--ctstate", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
	} {
		if _, err := pilot.run("iptables", append([]string{"-I", "FORWARD", "1"}, rule...)...); err != nil {
			return err
		}
		session.lanCleanup = append(session.lanCleanup, vpnHubCommand{"iptables", append([]string{"-D", "FORWARD"}, rule...)})
	}
	return nil
}

func (s *Server) cleanupVPNLAN(session *vpnSession) error {
	for len(session.lanCleanup) > 0 {
		last := len(session.lanCleanup) - 1
		command := session.lanCleanup[last]
		if _, err := s.vpnPilot.run(command.name, command.args...); err != nil {
			return err
		}
		session.lanCleanup = session.lanCleanup[:last]
	}
	return nil
}
