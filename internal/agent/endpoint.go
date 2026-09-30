package agent

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/user/remote-desktop/internal/models"
)

var endpointMu sync.Mutex

type lockValue struct {
	Present bool
	Seconds uint32
}

type lockPolicyState struct {
	Owned   bool
	Before  lockValue
	Applied lockValue
	Last    models.LockPolicyResult
}

type lockPolicyStore interface {
	Read() (lockValue, string, error)
	Write(lockValue) error
	Load() (lockPolicyState, error)
	Save(lockPolicyState) error
}

func executeLockPolicy(store lockPolicyStore, request models.LockPolicyRequest) models.LockPolicyResult {
	result := models.LockPolicyResult{ID: request.ID, Status: "failed"}
	if !request.Valid() {
		result.Detail = "Permintaan policy tidak valid"
		return result
	}
	current, blocker, err := store.Read()
	if err != nil {
		result.Detail = "Gagal membaca policy: " + err.Error()
		return result
	}
	if request.Mode == "audit" {
		result.Status = "audited"
		result.Detail = fmt.Sprintf("Registry: ada=%t, %d detik; target %d detik. Tidak mengubah perangkat. %s", current.Present, current.Seconds, request.Seconds, blocker)
		return result
	}
	state, err := store.Load()
	if err != nil {
		result.Detail = "Backup/receipt tidak dapat dibaca: " + err.Error()
		return result
	}
	if state.Last.ID == request.ID {
		return state.Last
	}
	if blocker != "" {
		result.Status = "conflict"
		result.Detail = blocker
		return result
	}
	if state.Owned && current != state.Applied {
		result.Status = "conflict"
		result.Detail = "Pengaturan berubah di luar RemoteDesk; tidak ditimpa, backup dipertahankan"
		return result
	}
	target := lockValue{Present: true, Seconds: request.Seconds}
	if request.Mode == "restore" {
		if !state.Owned {
			result.Status = "conflict"
			result.Detail = "Tidak ada backup policy milik RemoteDesk"
			return result
		}
		target = state.Before
	} else {
		if !state.Owned && current.Present && current.Seconds != 0 {
			result.Status = "conflict"
			result.Detail = "Policy sudah diatur pengelola lain; gunakan audit, tidak ditimpa"
			return result
		}
		if !state.Owned {
			state.Before = current
		}
		state.Owned, state.Applied = true, target
	}
	state.Last = models.LockPolicyResult{ID: request.ID, Status: "unknown", Detail: "Operasi dimulai; hasil belum pasti. Audit sebelum mencoba ulang"}
	if err := store.Save(state); err != nil {
		result.Detail = "Backup gagal; policy tidak diubah: " + err.Error()
		return result
	}
	if err := store.Write(target); err != nil {
		result.Status = "unknown"
		result.Detail = "Penulisan policy gagal; audit sebelum mencoba ulang: " + err.Error()
	} else if observed, _, err := store.Read(); err != nil || observed != target {
		result.Status = "unknown"
		result.Detail = "Nilai setelah penulisan tidak dapat diverifikasi; tidak dicoba ulang otomatis"
	} else {
		result.Status = "configured"
		result.Detail = fmt.Sprintf("Registry terverifikasi %d detik. Restart Windows diperlukan; penguncian sesi belum diuji", target.Seconds)
		if request.Mode == "restore" {
			state.Owned = false
			result.Status = "restored"
			result.Detail = "Nilai registry awal dipulihkan dan diverifikasi. Restart Windows diperlukan"
		}
	}
	state.Last = result
	if err := store.Save(state); err != nil {
		result.Status = "unknown"
		result.Detail = "Policy mungkin berubah tetapi receipt gagal disimpan; audit diperlukan"
	}
	return result
}

func (a *Agent) reportEndpoint() {
	endpointMu.Lock()
	report := collectEndpointReport()
	endpointMu.Unlock()
	message, _ := json.Marshal(map[string]interface{}{"action": "endpoint_report", "data": report})
	_ = a.writeTextMessage(message)
}

func (a *Agent) handleLockPolicy(request models.LockPolicyRequest) {
	endpointMu.Lock()
	result := runLockPolicy(request)
	endpointMu.Unlock()
	message, _ := json.Marshal(map[string]interface{}{"action": "lock_policy_result", "data": result})
	_ = a.writeTextMessage(message)
	a.reportEndpoint()
}
