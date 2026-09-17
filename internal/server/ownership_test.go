package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

func TestOwnershipAndSwitchWorkflow(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "ownership.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db}
	for _, u := range []string{"alice", "bob", "charlie"} {
		if err := db.CreateUser(u, "hash", "user", "Bandung"); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.CreateUser("outside", "hash", "user", "Medan"); err != nil {
		t.Fatal(err)
	}
	a := &models.ManualAsset{AssetTag: "A", Name: "Laptop A", Category: "laptop", Branch: "Bandung", OwnerUsername: "alice", AcquisitionYear: 2020}
	b := &models.ManualAsset{AssetTag: "B", Name: "Laptop B", Category: "laptop", Branch: "Bandung", OwnerUsername: "bob"}
	for _, asset := range []*models.ManualAsset{a, b} {
		if err := db.CreateManualAsset(asset); err != nil {
			t.Fatal(err)
		}
	}
	alice := &UserClaims{Username: "alice", Role: "user", Branch: "Bandung"}
	adh := &UserClaims{Username: "adh", Role: "adh", Branch: "Bandung"}
	call := func(handler http.HandlerFunc, method, url, body string, who *UserClaims) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, url, strings.NewReader(body))
		w := httptest.NewRecorder()
		handler(w, r.WithContext(context.WithValue(r.Context(), userClaimsKey, who)))
		return w
	}
	requireCode := func(w *httptest.ResponseRecorder, code int) {
		t.Helper()
		if w.Code != code {
			t.Fatalf("status=%d want=%d body=%s", w.Code, code, w.Body.String())
		}
	}
	list := call(s.handleManualAssets, "GET", "/api/assets/manual?branch=Medan", "", alice)
	requireCode(list, 200)
	var owned []models.ManualAsset
	if err := json.Unmarshal(list.Body.Bytes(), &owned); err != nil {
		t.Fatal(err)
	}
	if len(owned) != 1 || owned[0].ID != a.ID || owned[0].AcquisitionYear != 2020 {
		t.Fatalf("ownership filter: %+v", owned)
	}
	requireCode(call(s.handleManualAsset, "GET", "/api/assets/manual/"+b.ID, "", alice), 403)
	requireCode(call(s.handleManualAsset, "DELETE", "/api/assets/manual/"+a.ID, `{"reason":"test delete"}`, alice), 403)
	requireCode(call(s.handleManualAsset, "PUT", "/api/assets/manual/"+a.ID, `{"owner_username":"charlie","name":"changed","location":"Meja 2","condition":"fair"}`, alice), 200)
	saved, _ := db.GetManualAsset(a.ID)
	if saved.OwnerUsername != "alice" || saved.Name != "Laptop A" || saved.Location != "Meja 2" {
		t.Fatalf("editable field boundaries: %+v", saved)
	}
	request := func(to, swap string, who *UserClaims) *httptest.ResponseRecorder {
		return call(s.handleSwitchRequests, "POST", "/api/assets/switch-requests", fmt.Sprintf(`{"asset_id":%q,"asset_type":"manual","to_owner":%q,"swap_tag":%q,"reason":"Pergantian perangkat kerja"}`, a.ID, to, swap), who)
	}
	requireCode(request("outside", "", alice), 400)
	pending := request("bob", "B", alice)
	requireCode(pending, 201)
	var req models.AssetSwitchRequest
	if err := json.Unmarshal(pending.Body.Bytes(), &req); err != nil {
		t.Fatal(err)
	}
	if req.Status != "pending" || req.ID == 0 {
		t.Fatalf("invalid request: %+v", req)
	}
	saved, _ = db.GetManualAsset(a.ID)
	if saved.OwnerUsername != "alice" {
		t.Fatal("owner changed before approval")
	}
	approvalURL := fmt.Sprintf("/api/assets/switch-requests/%d/approve", req.ID)
	requireCode(call(s.handleSwitchRequestSubroute, "POST", approvalURL, `{}`, alice), 403)
	requireCode(call(s.handleSwitchRequestSubroute, "POST", approvalURL, `{}`, &UserClaims{Username: "otheradh", Role: "adh", Branch: "Medan"}), 403)
	requireCode(call(s.handleSwitchRequestSubroute, "POST", approvalURL, `{"note":"Sudah diperiksa"}`, adh), 200)
	saved, _ = db.GetManualAsset(a.ID)
	other, _ := db.GetManualAsset(b.ID)
	if saved.OwnerUsername != "bob" || other.OwnerUsername != "alice" {
		t.Fatal("swap did not update both owners")
	}
	if err := db.ReviewSwitchRequest(req.ID, "approved", "adh", ""); err == nil {
		t.Fatal("double approval accepted")
	}
	direct := request("charlie", "", adh)
	requireCode(direct, 201)
	saved, _ = db.GetManualAsset(a.ID)
	if saved.OwnerUsername != "charlie" {
		t.Fatal("ADH direct switch not applied")
	}
	// A stale request must never overwrite a newer handover.
	stale := &models.AssetSwitchRequest{AssetID: a.ID, AssetType: "manual", Branch: "Bandung", FromOwner: "bob", ToOwner: "alice", RequestedBy: "bob"}
	if err := db.CreateSwitchRequest(stale); err != nil {
		t.Fatal(err)
	}
	if err := db.ReviewSwitchRequest(stale.ID, "approved", "adh", ""); err == nil {
		t.Fatal("stale request accepted")
	}
	saved, _ = db.GetManualAsset(a.ID)
	if saved.OwnerUsername != "charlie" {
		t.Fatal("stale approval changed owner")
	}
	// Deletion/recreation of an account cannot acquire someone else's assignments.
	id, _, _, _, _ := db.GetUser("charlie")
	if err := db.DeleteUser(int64(id)); err == nil {
		t.Fatal("deleted assigned holder")
	}
}

func TestSwitchRollbackAndDeviceOwnership(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, u := range []string{"alice", "bob"} {
		if err := db.CreateUser(u, "hash", "user", "Bandung"); err != nil {
			t.Fatal(err)
		}
	}
	dev := &models.Device{ID: "dev-a", Hostname: "A", Branch: "Bandung", GroupName: "Bandung"}
	if err := db.UpsertDevice(dev); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateDeviceOwner(dev.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	list, total, err := db.ListDevicesForOwner("alice", "", 50, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("device ownership %d %v", total, err)
	}
	req := &models.AssetSwitchRequest{AssetID: dev.ID, AssetType: "device", FromOwner: "alice", ToOwner: "bob", Branch: "Bandung", SwapAssetID: "missing", SwapAssetType: "manual"}
	if err := db.CreateSwitchRequest(req); err != nil {
		t.Fatal(err)
	}
	if err := db.ReviewSwitchRequest(req.ID, "approved", "adh", ""); err == nil {
		t.Fatal("missing replacement accepted")
	}
	saved, _ := db.GetDevice(dev.ID)
	if saved.OwnerUsername != "alice" {
		t.Fatal("first transfer was not rolled back")
	}
	pending, _ := db.GetSwitchRequest(req.ID)
	if pending.Status != "pending" {
		t.Fatal("approval was not rolled back")
	}
	if !strings.Contains(deviceRecommendation(&models.Device{AcquisitionYear: time.Now().Year() - 6}), "5 tahun") {
		t.Fatal("missing age recommendation")
	}
	if strings.Contains(deviceRecommendation(&models.Device{RegisteredAt: time.Now().AddDate(-8, 0, 0)}), "5 tahun") {
		t.Fatal("registration mistaken for asset age")
	}
}

func TestHolderCannotAccessUnrelatedRoutes(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "access.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.CreateUser("holder", "hash", "user", "Bandung"); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, cfg: Config{APIKey: "private-agent-key", JWTSecret: "test-secret"}}
	for _, path := range []string{"/ws/viewer", "/api/logs/another-device", "/api/agent/package", "/api/users", "/api/network-scans"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+generateToken("holder", "user", "Bandung", s.cfg.JWTSecret))
		w := httptest.NewRecorder()
		s.authMiddleware(func(w http.ResponseWriter, r *http.Request) { t.Error("forbidden handler reached: " + path) })(w, r)
		if w.Code != 403 {
			t.Fatalf("%s: got %d", path, w.Code)
		}
	}
}
