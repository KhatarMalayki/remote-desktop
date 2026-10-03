package vpn

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

const Lease = 60 * time.Second
const SessionLimit = 15 * time.Minute
const TunnelName = "RemoteDeskPilot"

type Command struct {
	Operation string `json:"operation"`
	ID string `json:"id"`
	Expires int64 `json:"expires"`
	Config *Config `json:"config,omitempty"`
}

type Config struct {
	Address string `json:"address"`
	Network string `json:"network"`
	Endpoint string `json:"endpoint"`
	ServerKey string `json:"server_key"`
}

type Status struct {
	ID string `json:"id"`
	State string `json:"state"`
	Detail string `json:"detail"`
	PublicKey string `json:"public_key,omitempty"`
	Address string `json:"address,omitempty"`
	Updated int64 `json:"updated"`
}

func NewKey() (string, string, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil { return "", "", err }
	return base64.StdEncoding.EncodeToString(key.Bytes()), base64.StdEncoding.EncodeToString(key.PublicKey().Bytes()), nil
}

func ValidKey(value string) bool {
	decoded, err := base64.StdEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32 && value == base64.StdEncoding.EncodeToString(decoded) && strings.Trim(value, "A=") != ""
}

func Network(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(value)
	if err != nil || !prefix.Addr().Is4() || !prefix.Addr().IsPrivate() || prefix.Bits() != 24 || prefix != prefix.Masked() {
		return netip.Prefix{}, fmt.Errorf("pilot hanya menerima satu subnet IPv4 privat /24, bukan default route atau LAN")
	}
	return prefix, nil
}

func (config Config) Validate() error {
	prefix, err := Network(config.Network)
	if err != nil { return err }
	address, err := netip.ParseAddr(config.Address)
	if err != nil || !prefix.Contains(address) || address == prefix.Addr() || address == prefix.Addr().Next() || address.As4()[3] == 255 {
		return fmt.Errorf("IP client harus berada di subnet VPN, bukan alamat hub/network/broadcast")
	}
	if !ValidKey(config.ServerKey) { return fmt.Errorf("public key hub tidak valid") }
	host, port, err := net.SplitHostPort(config.Endpoint)
	portNumber, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || portNumber < 1 || portNumber > 65535 || host == "" || len(host) > 253 || strings.ContainsAny(host, " \t\r\n/\\#;=") {
		return fmt.Errorf("endpoint harus hostname/IP:port UDP yang valid")
	}
	if address, err := netip.ParseAddr(host); err == nil && (prefix.Contains(address) || address.IsUnspecified() || address.IsMulticast()) {
		return fmt.Errorf("endpoint tidak boleh melalui subnet VPN atau alamat multicast/unspecified")
	}
	return nil
}

func (config Config) Render(privateKey string) (string, error) {
	if err := config.Validate(); err != nil { return "", err }
	if !ValidKey(privateKey) { return "", fmt.Errorf("private key tidak valid") }
	return fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s/32\nMTU = 1380\n\n[Peer]\nPublicKey = %s\nEndpoint = %s\nAllowedIPs = %s\nPersistentKeepalive = 25\n", privateKey, config.Address, config.ServerKey, config.Endpoint, config.Network), nil
}

func CheckRoutes(network string, routes []string) error {
	prefix, err := Network(network)
	if err != nil { return err }
	for _, route := range routes {
		other, err := netip.ParsePrefix(route)
		if err != nil { return fmt.Errorf("gagal memvalidasi rute lokal") }
		if other.Addr().Is4() && other.Bits() != 0 && prefix.Overlaps(other) {
			return fmt.Errorf("subnet VPN bentrok dengan rute lokal/VPN lain %s; koneksi dibatalkan", other)
		}
	}
	return nil
}

func ValidCommand(command Command, now time.Time) bool {
	if len(command.ID) != 32 || strings.Trim(command.ID, "0123456789abcdef") != "" { return false }
	remaining := command.Expires-now.Unix()
	return remaining > 0 && remaining <= 120
}

func LeaseExpired(now, started, lastAck time.Time) bool {
	return now.Sub(lastAck) > Lease || now.Sub(started) > SessionLimit || now.Before(lastAck)
}
