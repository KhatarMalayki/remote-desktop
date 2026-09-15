package agent

import (
	"bufio"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

// ScanLocalNetwork discovers neighbors only in the agent's local /24 subnet.
// UDP probes prime the OS ARP/neighbor cache; ARP data provides the MAC address.
func ScanLocalNetwork() (string, []models.NetworkScanHost, error) {
	ip := net.ParseIP(getLocalIP()).To4()
	if ip == nil {
		return "", nil, &net.ParseError{Type: "IP", Text: "local IPv4 unavailable"}
	}
	subnet := net.IPv4(ip[0], ip[1], ip[2], 0).String() + "/24"
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)
	for host := 1; host < 255; host++ {
		if byte(host) == ip[3] {
			continue
		}
		target := net.IPv4(ip[0], ip[1], ip[2], byte(host)).String()
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c, err := net.DialTimeout("udp", net.JoinHostPort(target, "9"), 150*time.Millisecond)
			if err == nil {
				c.Close()
			}
		}()
	}
	wg.Wait()
	return subnet, arpNeighbors(ip), nil
}

func arpNeighbors(localIP net.IP) []models.NetworkScanHost {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("arp", "-a")
	} else {
		cmd = exec.Command("arp", "-an")
	}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	prefix := strconv.Itoa(int(localIP[0])) + "." + strconv.Itoa(int(localIP[1])) + "." + strconv.Itoa(int(localIP[2])) + "."
	seen := map[string]bool{}
	var hosts []models.NetworkScanHost
	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 2 {
			continue
		}
		candidate := strings.Trim(f[0], "()")
		if !strings.HasPrefix(candidate, prefix) || net.ParseIP(candidate) == nil {
			continue
		}
		mac := ""
		for _, part := range f[1:] {
			if strings.Count(part, "-") == 5 || strings.Count(part, ":") == 5 {
				mac = strings.ToUpper(strings.ReplaceAll(part, "-", ":"))
				break
			}
		}
		if mac == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		hosts = append(hosts, models.NetworkScanHost{IP: candidate, MAC: mac})
	}
	return hosts
}
