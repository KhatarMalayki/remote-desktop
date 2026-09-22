package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestInitialAssignmentAttachmentAndTimeline(t *testing.T) {
	tmp := t.TempDir()
	db, err := NewDB(filepath.Join(tmp, "workflow.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.CreateUser("holder", "hash", "user", "Bandung"); err != nil {
		t.Fatal(err)
	}
	asset := &models.ManualAsset{AssetTag: "NEW-1", Name: "Laptop Baru", Category: "laptop", Branch: "Bandung"}
	if err := db.CreateManualAsset(asset); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, attachmentsDir: filepath.Join(tmp, "attachments")}
	if err := os.MkdirAll(s.attachmentsDir, 0700); err != nil {
		t.Fatal(err)
	}
	adh := &UserClaims{Username: "adh", Role: "adh", Branch: "Bandung"}
	r := httptest.NewRequest("POST", "/api/assets/switch-requests", strings.NewReader(fmt.Sprintf(`{"asset_id":%q,"asset_type":"manual","to_owner":"holder","reason":"Penyerahan perangkat baru"}`, asset.ID)))
	w := httptest.NewRecorder()
	s.handleSwitchRequests(w, r.WithContext(context.WithValue(r.Context(), userClaimsKey, adh)))
	if w.Code != 201 {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	var sw models.AssetSwitchRequest
	if err := json.Unmarshal(w.Body.Bytes(), &sw); err != nil {
		t.Fatal(err)
	}
	if sw.Operation != "assignment" || sw.Status != "approved" {
		t.Fatalf("unexpected workflow: %+v", sw)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("file", "berita-acara.pdf")
	part.Write([]byte("%PDF-1.4 test"))
	mw.Close()
	u := httptest.NewRequest("POST", fmt.Sprintf("/api/assets/switch-requests/%d/attachments", sw.ID), &body)
	u.Header.Set("Content-Type", mw.FormDataContentType())
	uw := httptest.NewRecorder()
	s.handleSwitchRequestSubroute(uw, u.WithContext(context.WithValue(u.Context(), userClaimsKey, adh)))
	if uw.Code != 201 {
		t.Fatalf("upload status=%d body=%s", uw.Code, uw.Body.String())
	}
	files, err := db.ListAttachments("handover", sw.ID)
	if err != nil || len(files) != 1 {
		t.Fatalf("attachments=%v err=%v", files, err)
	}
	activities, err := db.ListActivities("Bandung", asset.ID, "", "", "", "", 20)
	if err != nil || len(activities) < 3 {
		t.Fatalf("activities=%v err=%v", activities, err)
	}
	other := &UserClaims{Username: "outsider", Role: "user", Branch: "Medan"}
	download := httptest.NewRequest("GET", fmt.Sprintf("/api/assets/attachments/%d", files[0].ID), nil)
	dw := httptest.NewRecorder()
	s.handleAttachment(dw, download.WithContext(context.WithValue(download.Context(), userClaimsKey, other)))
	if dw.Code != 403 {
		t.Fatalf("outsider download=%d", dw.Code)
	}
	if _, err := db.db.Exec(`INSERT OR IGNORE INTO branches(name,type) VALUES('Jakarta','cabang')`); err != nil {
		t.Fatal(err)
	}
	move := httptest.NewRequest("POST", "/api/assets/relocate", strings.NewReader(fmt.Sprintf(`{"asset_id":%q,"asset_type":"manual","to_branch":"Jakarta","to_location":"Lantai 2","reason":"Mutasi perangkat antar kantor"}`, asset.ID)))
	movedResponse := httptest.NewRecorder()
	s.handleRelocateAsset(movedResponse, move.WithContext(context.WithValue(move.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"})))
	if movedResponse.Code != 200 {
		t.Fatalf("relocate=%d body=%s", movedResponse.Code, movedResponse.Body.String())
	}
	moved, _ := db.GetManualAsset(asset.ID)
	if moved.Branch != "Jakarta" || moved.Location != "Lantai 2" || moved.OwnerUsername != "" {
		t.Fatalf("unexpected relocated asset: %+v", moved)
	}
}

func TestBranchADHAndHeadOfficeSPVHolderWorkflow(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "adh-spv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db}

	if err := db.CreateUser("adh_surabaya", "hash", "adh", "Surabaya"); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateUser("spv_it", "hash", "spv", "Pusat"); err != nil {
		t.Fatal(err)
	}

	surabayaHolders, err := db.ListHolderOptions("Surabaya")
	if err != nil {
		t.Fatal(err)
	}
	if len(surabayaHolders) != 1 || surabayaHolders[0].Username != "adh_surabaya" || surabayaHolders[0].Role != "adh" {
		t.Fatalf("unexpected surabaya holders: %+v", surabayaHolders)
	}

	pusatHolders, err := db.ListHolderOptions("Pusat")
	if err != nil {
		t.Fatal(err)
	}
	if len(pusatHolders) != 1 || pusatHolders[0].Username != "spv_it" || pusatHolders[0].Role != "spv" {
		t.Fatalf("unexpected pusat holders: %+v", pusatHolders)
	}
	hoHolders, err := db.ListHolderOptions("HO-Bintaro")
	if err != nil {
		t.Fatal(err)
	}
	if len(hoHolders) != 1 || hoHolders[0].Username != "spv_it" {
		t.Fatalf("unexpected HO-Bintaro holders: %+v", hoHolders)
	}

	emptyHolders, err := db.ListHolderOptions("Medan")
	if err != nil {
		t.Fatal(err)
	}
	if emptyHolders == nil || len(emptyHolders) != 0 {
		t.Fatalf("expected empty non-nil slice, got: %#v", emptyHolders)
	}

	devSby := &models.Device{ID: "dev-sby", Hostname: "PC-SBY-01", Branch: "Surabaya", GroupName: "Surabaya"}
	if err := db.UpsertDevice(devSby); err != nil {
		t.Fatal(err)
	}
	adminClaims := &UserClaims{Username: "admin", Role: "admin"}
	r1 := httptest.NewRequest("POST", "/api/assets/switch-requests", strings.NewReader(`{"asset_id":"dev-sby","asset_type":"device","to_owner":"adh_surabaya","reason":"Penetapan PIC Cabang Surabaya"}`))
	w1 := httptest.NewRecorder()
	s.handleSwitchRequests(w1, r1.WithContext(context.WithValue(r1.Context(), userClaimsKey, adminClaims)))
	if w1.Code != 201 {
		t.Fatalf("assign adh failed: %d %s", w1.Code, w1.Body.String())
	}
	savedSby, _ := db.GetDevice("dev-sby")
	if savedSby.OwnerUsername != "adh_surabaya" {
		t.Fatalf("expected owner adh_surabaya, got: %s", savedSby.OwnerUsername)
	}

	devHO := &models.Device{ID: "dev-ho", Hostname: "SS-HO-ITAPSSPT", Branch: "Pusat", GroupName: "Pusat"}
	if err := db.UpsertDevice(devHO); err != nil {
		t.Fatal(err)
	}
	r2 := httptest.NewRequest("POST", "/api/assets/switch-requests", strings.NewReader(`{"asset_id":"dev-ho","asset_type":"device","to_owner":"spv_it","reason":"Penetapan PIC SPV IT HO"}`))
	w2 := httptest.NewRecorder()
	s.handleSwitchRequests(w2, r2.WithContext(context.WithValue(r2.Context(), userClaimsKey, adminClaims)))
	if w2.Code != 201 {
		t.Fatalf("assign spv failed: %d %s", w2.Code, w2.Body.String())
	}
	savedHO, _ := db.GetDevice("dev-ho")
	if savedHO.OwnerUsername != "spv_it" {
		t.Fatalf("expected owner spv_it, got: %s", savedHO.OwnerUsername)
	}
}
