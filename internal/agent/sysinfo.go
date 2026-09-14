package agent

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type SystemInfo struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	CPUModel    string `json:"cpu_model"`
	CPUCores    int    `json:"cpu_cores"`
	MemoryTotal uint64 `json:"memory_total"`
	MemoryUsed  uint64 `json:"memory_used"`
	DiskTotal   uint64 `json:"disk_total"`
	DiskUsed    uint64 `json:"disk_used"`
	LocalIP     string `json:"local_ip"`
	Version     string `json:"version"`
	Branch      string `json:"branch"`
}

func CollectSystemInfo() SystemInfo {
	info := SystemInfo{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	info.Hostname, _ = os.Hostname()
	info.CPUModel = getCPUModel()
	info.MemoryTotal, info.MemoryUsed = getMemoryInfo()
	info.DiskTotal, info.DiskUsed = getDiskInfo()
	info.LocalIP = getLocalIP()

	return info
}

func getCPUModel() string {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return "unknown"
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	case "windows":
		out, err := exec.Command("wmic", "cpu", "get", "name", "/format:list").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.HasPrefix(line, "Name=") {
					return strings.TrimSpace(strings.TrimPrefix(line, "Name="))
				}
			}
		}
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return "unknown"
}

func getMemoryInfo() (total, used uint64) {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0, 0
		}
		var memTotal, memAvail uint64
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			val, _ := strconv.ParseUint(fields[1], 10, 64)
			val *= 1024
			switch fields[0] {
			case "MemTotal:":
				memTotal = val
			case "MemAvailable:":
				memAvail = val
			}
		}
		return memTotal, memTotal - memAvail

	case "windows":
		out, err := exec.Command("wmic", "OS", "get", "FreePhysicalMemory,TotalVisibleMemorySize", "/format:list").Output()
		if err != nil {
			return 0, 0
		}
		var totalKB, freeKB uint64
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
				totalKB, _ = strconv.ParseUint(strings.TrimPrefix(line, "TotalVisibleMemorySize="), 10, 64)
			}
			if strings.HasPrefix(line, "FreePhysicalMemory=") {
				freeKB, _ = strconv.ParseUint(strings.TrimPrefix(line, "FreePhysicalMemory="), 10, 64)
			}
		}
		return totalKB * 1024, (totalKB - freeKB) * 1024

	case "darwin":
		out, _ := exec.Command("sysctl", "-n", "hw.memsize").Output()
		total, _ = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		return total, 0
	}
	return 0, 0
}

func getDiskInfo() (total, used uint64) {
	switch runtime.GOOS {
	case "linux", "darwin":
		out, err := exec.Command("df", "-B1", "/").Output()
		if err != nil {
			return 0, 0
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				total, _ = strconv.ParseUint(fields[1], 10, 64)
				used, _ = strconv.ParseUint(fields[2], 10, 64)
			}
		}
	case "windows":
		out, err := exec.Command("wmic", "logicaldisk", "where", "DeviceID='C:'",
			"get", "Size,FreeSpace", "/format:list").Output()
		if err != nil {
			return 0, 0
		}
		var size, free uint64
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Size=") {
				size, _ = strconv.ParseUint(strings.TrimPrefix(line, "Size="), 10, 64)
			}
			if strings.HasPrefix(line, "FreeSpace=") {
				free, _ = strconv.ParseUint(strings.TrimPrefix(line, "FreeSpace="), 10, 64)
			}
		}
		return size, size - free
	}
	return 0, 0
}

func getLocalIP() string {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("hostname", "-I").Output()
		if err == nil {
			fields := strings.Fields(string(out))
			if len(fields) > 0 {
				return fields[0]
			}
		}
	case "windows":
		out, err := exec.Command("powershell", "-Command",
			"(Get-NetIPAddress -AddressFamily IPv4 | Where-Object { $_.InterfaceAlias -notlike '*Loopback*' } | Select-Object -First 1).IPAddress").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "darwin":
		out, err := exec.Command("ipconfig", "getifaddr", "en0").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return "127.0.0.1"
}

func GetCPUUsage() float64 {
	switch runtime.GOOS {
	case "linux":
		data1, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0
		}
		idle1, total1 := parseProcStat(string(data1))
		// sleep handled by caller
		data2, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0
		}
		idle2, total2 := parseProcStat(string(data2))
		idleDelta := float64(idle2 - idle1)
		totalDelta := float64(total2 - total1)
		if totalDelta == 0 {
			return 0
		}
		return (1.0 - idleDelta/totalDelta) * 100.0
	}
	return 0
}

func parseProcStat(data string) (idle, total uint64) {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return
			}
			for i := 1; i < len(fields); i++ {
				val, _ := strconv.ParseUint(fields[i], 10, 64)
				total += val
				if i == 4 {
					idle = val
				}
			}
			return
		}
	}
	return
}

func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
