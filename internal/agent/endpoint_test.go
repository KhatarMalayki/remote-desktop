package agent

import (
	"errors"
	"testing"

	"github.com/user/remote-desktop/internal/models"
)

type mockLockStore struct {
	current  lockValue
	blocker  string
	readErr  error
	writeErr error
	state    lockPolicyState
	loadErr  error
	saveErr  error
	writes   []lockValue
}

func (m *mockLockStore) Read() (lockValue, string, error) {
	return m.current, m.blocker, m.readErr
}
func (m *mockLockStore) Write(v lockValue) error {
	m.writes = append(m.writes, v)
	m.current = v
	return m.writeErr
}
func (m *mockLockStore) Load() (lockPolicyState, error) {
	return m.state, m.loadErr
}
func (m *mockLockStore) Save(s lockPolicyState) error {
	m.state = s
	return m.saveErr
}

func TestExecuteLockPolicyAuditDoesNotMutate(t *testing.T) {
	store := &mockLockStore{current: lockValue{Present: true, Seconds: 300}, blocker: "ManageEngine terdeteksi"}
	request := models.LockPolicyRequest{ID: "0123456789abcdef0123456789abcdef", Mode: "audit", Seconds: 900}
	result := executeLockPolicy(store, request)
	if result.Status != "audited" || len(store.writes) != 0 {
		t.Fatalf("audit harus no-op: %+v writes=%d", result, len(store.writes))
	}
}

func TestExecuteLockPolicyRefusesWhenExternalManagerConflict(t *testing.T) {
	store := &mockLockStore{current: lockValue{Present: true, Seconds: 300}, blocker: "ManageEngine terdeteksi"}
	request := models.LockPolicyRequest{ID: "0123456789abcdef0123456789abcdef", Mode: "apply", Seconds: 900}
	result := executeLockPolicy(store, request)
	if result.Status != "conflict" || len(store.writes) != 0 {
		t.Fatalf("harus conflict saat ada pengelola lain: %+v", result)
	}
}

func TestExecuteLockPolicyApplyAndRestoreWithDrift(t *testing.T) {
	store := &mockLockStore{current: lockValue{Present: false, Seconds: 0}}
	applyReq := models.LockPolicyRequest{ID: "0123456789abcdef0123456789abcdef", Mode: "apply", Seconds: 600}
	resApply := executeLockPolicy(store, applyReq)
	if resApply.Status != "configured" || !store.current.Present || store.current.Seconds != 600 || !store.state.Owned {
		t.Fatalf("apply gagal: %+v store=%+v", resApply, store.current)
	}

	store.current = lockValue{Present: true, Seconds: 1200}
	resDrift := executeLockPolicy(store, models.LockPolicyRequest{ID: "1123456789abcdef0123456789abcdef", Mode: "apply", Seconds: 900})
	if resDrift.Status != "conflict" {
		t.Fatalf("harus conflict saat drift terdeteksi: %+v", resDrift)
	}

	store.current = lockValue{Present: true, Seconds: 600}
	resRestore := executeLockPolicy(store, models.LockPolicyRequest{ID: "2123456789abcdef0123456789abcdef", Mode: "restore", Seconds: 600})
	if resRestore.Status != "restored" || store.current.Present || store.state.Owned {
		t.Fatalf("restore gagal: %+v store=%+v state=%+v", resRestore, store.current, store.state)
	}
}

func TestExecuteLockPolicyIdempotentAndSaveFailure(t *testing.T) {
	store := &mockLockStore{current: lockValue{Present: false, Seconds: 0}, state: lockPolicyState{Last: models.LockPolicyResult{ID: "saved-id-00000000000000000000000", Status: "configured", Detail: "ok"}}}
	req := models.LockPolicyRequest{ID: "saved-id-00000000000000000000000", Mode: "apply", Seconds: 600}
	if res := executeLockPolicy(store, req); res.Status != "configured" {
		t.Fatalf("idempotent fail: %+v", res)
	}
	failStore := &mockLockStore{current: lockValue{Present: false, Seconds: 0}, saveErr: errors.New("disk full")}
	if res := executeLockPolicy(failStore, models.LockPolicyRequest{ID: "new-id-00000000000000000000000000", Mode: "apply", Seconds: 600}); res.Status != "failed" || len(failStore.writes) != 0 {
		t.Fatalf("harus gagal sebelum write bila backup gagal: %+v writes=%d", res, len(failStore.writes))
	}
}
