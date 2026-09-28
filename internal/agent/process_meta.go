package agent

import (
	"sort"
	"strings"

	"github.com/user/remote-desktop/internal/models"
)

func enrichProcesses(processes []models.ProcessInfo) []models.ProcessInfo {
	for i := range processes {
		name := strings.ToLower(processes[i].Name)
		processes[i].IsSystem = isSystemProcess(name)
		processes[i].Icon = processIcon(name)
	}
	return processes
}

func activeApplications(processes []models.ProcessInfo) []string {
	seen := map[string]string{}
	for _, process := range processes {
		if !process.IsSystem && process.Name != "" {
			seen[strings.ToLower(process.Name)] = process.Name
		}
	}
	apps := make([]string, 0, len(seen))
	for _, name := range seen {
		apps = append(apps, name)
	}
	sort.Strings(apps)
	if len(apps) > 30 {
		apps = apps[:30]
	}
	return apps
}

func isSystemProcess(name string) bool {
	system := map[string]bool{"system": true, "registry": true, "memory compression": true, "smss": true, "csrss": true, "wininit": true, "services": true, "lsass": true, "svchost": true, "fontdrvhost": true, "dwm": true, "winlogon": true, "sihost": true, "searchhost": true, "searchapp": true, "runtimebroker": true, "startmenuexperiencehost": true, "shellexperiencehost": true, "msmpeng": true, "securityhealthservice": true, "spoolsv": true, "conhost": true}
	return system[name]
}

func processIcon(name string) string {
	for key, icon := range map[string]string{"chrome": "🌐", "msedge": "🌐", "firefox": "🦊", "excel": "📗", "winword": "📘", "powerpnt": "📙", "outlook": "✉️", "teams": "💬", "whatsapp": "💬", "onedrive": "☁️", "photos": "🖼️", "acrobat": "📄", "code": "💻", "rustdesk": "🖥️", "anydesk": "🖥️"} {
		if strings.Contains(name, key) {
			return icon
		}
	}
	if isSystemProcess(name) {
		return "⚙️"
	}
	return "▣"
}
