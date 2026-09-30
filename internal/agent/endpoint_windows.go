package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"github.com/user/remote-desktop/internal/models"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const inactivityKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System`
const endpointBackupKey = `SOFTWARE\RemoteDesk\EndpointPolicy`

type windowsLockStore struct{}

func installedMachineApplications() ([]models.InstalledApplication, error) {
	apps := []models.InstalledApplication{}
	var failures []string
	for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
		root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`, registry.READ|view)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			failures = append(failures, "registry uninstall tidak dapat dibaca")
			continue
		}
		names, err := root.ReadSubKeyNames(-1)
		root.Close()
		if err != nil {
			failures = append(failures, "enumerasi uninstall gagal")
			continue
		}
		for _, name := range names {
			key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\`+name, registry.QUERY_VALUE|view)
			if err != nil {
				failures = append(failures, "sebagian entri tidak terbaca")
				continue
			}
			display, _, displayErr := key.GetStringValue("DisplayName")
			version, _, _ := key.GetStringValue("DisplayVersion")
			key.Close()
			if displayErr != nil && !errors.Is(displayErr, registry.ErrNotExist) {
				failures = append(failures, "nama aplikasi tidak terbaca")
			}
			if display != "" {
				apps = append(apps, models.InstalledApplication{Name: boundedEndpointText(display, 512), Version: boundedEndpointText(version, 128)})
			}
		}
	}
	manager, err := mgr.Connect()
	if err != nil {
		manager, err = openMachineSCM()
	}
	if err != nil {
		failures = append(failures, "service manager tidak dapat dibaca")
	} else {
		defer manager.Disconnect()
		names, err := manager.ListServices()
		if err != nil {
			failures = append(failures, "enumerasi service gagal")
		}
		for _, name := range names {
			entry := models.InstalledApplication{Name: boundedEndpointText(name, 512), Service: "unknown"}
			service, err := manager.OpenService(name)
			if err == nil {
				if config, err := service.Config(); err == nil {
					entry.Name = boundedEndpointText(name+" / "+config.DisplayName, 512)
				}
				if status, err := service.Query(); err == nil {
					switch status.State {
					case svc.Running:
						entry.Service = "running"
					case svc.Stopped:
						entry.Service = "stopped"
					default:
						entry.Service = "transitioning"
					}
				}
				service.Close()
			}
			apps = append(apps, entry)
		}
	}
	if len(apps) > 2000 {
		apps = apps[:2000]
		failures = append(failures, "inventaris melebihi batas")
	}
	if len(failures) > 0 {
		return apps, fmt.Errorf("%s", failures[0])
	}
	return apps, nil
}

func boundedEndpointText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for len(value) > limit {
		value = string([]rune(value)[:len([]rune(value))-1])
	}
	return value
}

func readLockValue() (lockValue, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, inactivityKey, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if errors.Is(err, registry.ErrNotExist) {
		return lockValue{}, nil
	}
	if err != nil {
		return lockValue{}, err
	}
	defer key.Close()
	value, kind, err := key.GetIntegerValue("InactivityTimeoutSecs")
	if errors.Is(err, registry.ErrNotExist) {
		return lockValue{}, nil
	}
	if err != nil {
		return lockValue{}, err
	}
	if kind != registry.DWORD {
		return lockValue{}, fmt.Errorf("tipe policy bukan DWORD")
	}
	return lockValue{Present: true, Seconds: uint32(value)}, nil
}

func managementBlocker(apps []models.InstalledApplication, scanErr error) (string, error) {
	var name *uint16
	var joinStatus uint32
	if err := windows.NetGetJoinInformation(nil, &name, &joinStatus); err != nil {
		return "", err
	}
	if name != nil {
		windows.NetApiBufferFree((*byte)(unsafe.Pointer(name)))
	}
	if joinStatus == windows.NetSetupDomainName {
		return "Perangkat domain: audit saja; kelola policy melalui pengelola domain", nil
	}
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Enrollments`, registry.READ|registry.WOW64_64KEY)
	if err == nil {
		names, readErr := key.ReadSubKeyNames(-1)
		key.Close()
		if readErr != nil {
			return "", readErr
		}
		for _, entry := range names {
			if len(entry) == 36 {
				return "Indikasi enrollment MDM: audit saja; konfirmasi dengan pengelola endpoint", nil
			}
		}
	} else if !errors.Is(err, registry.ErrNotExist) {
		return "", err
	}
	if scanErr != nil {
		return "Inventaris tidak lengkap: penerapan ditolak untuk mencegah konflik", nil
	}
	for _, app := range apps {
		name := strings.ToLower(app.Name)
		if strings.Contains(name, "manageengine") || strings.Contains(name, "desktop central") || strings.Contains(name, "endpoint central") || strings.Contains(name, "dcagent") {
			return "ManageEngine terdeteksi: audit saja; koordinasikan pemilik policy sebelum penerapan", nil
		}
	}
	return "", nil
}

func (windowsLockStore) Read() (lockValue, string, error) {
	value, err := readLockValue()
	if err != nil {
		return value, "", err
	}
	apps, scanErr := installedMachineApplications()
	blocker, err := managementBlocker(apps, scanErr)
	return value, blocker, err
}

func (windowsLockStore) Write(value lockValue) error {
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, inactivityKey, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return err
	}
	defer key.Close()
	if value.Present {
		return key.SetDWordValue("InactivityTimeoutSecs", value.Seconds)
	}
	err = key.DeleteValue("InactivityTimeoutSecs")
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

func (windowsLockStore) Load() (lockPolicyState, error) {
	var state lockPolicyState
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, endpointBackupKey, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if errors.Is(err, registry.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	defer key.Close()
	raw, _, err := key.GetStringValue("State")
	if errors.Is(err, registry.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	err = json.Unmarshal([]byte(raw), &state)
	return state, err
}

func (windowsLockStore) Save(state lockPolicyState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.LOCAL_MACHINE, endpointBackupKey, registry.SET_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return err
	}
	defer key.Close()
	if err = key.SetStringValue("State", string(raw)); err != nil {
		return err
	}
	return nil
}

func openMachineSCM() (*mgr.Mgr, error) {
	handle, err := windows.OpenSCManager(nil, nil, windows.GENERIC_READ)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: handle}, nil
}

func collectEndpointReport() models.EndpointReport {
	report := models.EndpointReport{Status: "ok"}
	apps, scanErr := installedMachineApplications()
	report.Applications = apps
	if scanErr != nil {
		report.Status = "unknown"
		report.Detail = scanErr.Error()
	}
	value, err := readLockValue()
	if err != nil {
		report.PolicyError = err.Error()
		return report
	}
	report.LockSeconds, report.LockPresent = value.Seconds, value.Present
	blocker, err := managementBlocker(apps, scanErr)
	if err != nil {
		report.PolicyError = err.Error()
	} else {
		report.PolicyError = blocker
	}
	state, err := (windowsLockStore{}).Load()
	if err != nil {
		report.PolicyError = "Backup tidak terbaca: " + err.Error()
	} else {
		report.Managed = state.Owned
		report.Drift = state.Owned && value != state.Applied
	}
	return report
}

func runLockPolicy(request models.LockPolicyRequest) models.LockPolicyResult {
	return executeLockPolicy(windowsLockStore{}, request)
}
