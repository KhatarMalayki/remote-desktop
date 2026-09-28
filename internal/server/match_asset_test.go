package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

func TestMatchManualAssetPriorityAndDuplicateWarning(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test_match.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 1. Setup manual asset di Cabang Jakarta
	asset1 := &models.ManualAsset{
		ID:                 "ast-001",
		AssetTag:           "LAP-JKT-01",
		Name:               "PC-ADMIN",
		Category:           "laptop",
		Branch:             "Jakarta",
		OwnerUsername:      "budi",
		AssignedTo:         "Budi Santoso",
		VerificationStatus: "verified",
		AcquisitionYear:    2023,
	}
	if err := db.CreateManualAsset(asset1); err != nil {
		t.Fatal(err)
	}

	// Case 1: Prioritas 1 - match via asset_tag (device hostname sama dengan asset_tag)
	dev1 := &models.Device{
		ID:           "dev-001",
		Hostname:     "lap-jkt-01",
		Branch:       "Jakarta",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	if err := db.UpsertDevice(dev1); err != nil {
		t.Fatal(err)
	}
	loaded1, err := db.GetDevice("dev-001")
	if err != nil {
		t.Fatal(err)
	}
	if loaded1.ManualAssetID != "ast-001" || loaded1.OwnerUsername != "budi" || loaded1.AssignedTo != "Budi Santoso" {
		t.Fatalf("expected auto-link to ast-001, got: %#v", loaded1)
	}

	// Setup manual asset kedua untuk uji prioritas 2 (hostname)
	asset2 := &models.ManualAsset{
		ID:            "ast-002",
		AssetTag:      "TAG-XYZ-99",
		Name:          "SRV-BACKUP",
		Category:      "pc",
		Branch:        "Bandung",
		OwnerUsername: "siti",
		AssignedTo:    "Siti Rahma",
	}
	if err := db.CreateManualAsset(asset2); err != nil {
		t.Fatal(err)
	}

	// Case 2: Prioritas 2 - match via Name/Hostname saat tidak ada duplikat
	dev2 := &models.Device{
		ID:           "dev-002",
		Hostname:     "srv-backup",
		Branch:       "Bandung",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	if err := db.UpsertDevice(dev2); err != nil {
		t.Fatal(err)
	}
	loaded2, err := db.GetDevice("dev-002")
	if err != nil {
		t.Fatal(err)
	}
	if loaded2.ManualAssetID != "ast-002" || loaded2.OwnerUsername != "siti" {
		t.Fatalf("expected auto-link via hostname to ast-002, got: %#v", loaded2)
	}

	// Case 3: Prioritas 2 dengan hostname duplicate -> harus ditolak dan diberi log warning
	asset3 := &models.ManualAsset{
		ID:            "ast-003",
		AssetTag:      "TAG-DUP-01",
		Name:          "CASHIER-01",
		Branch:        "Surabaya",
		OwnerUsername: "dewi",
	}
	if err := db.CreateManualAsset(asset3); err != nil {
		t.Fatal(err)
	}

	// Device pertama sudah ada dengan hostname "cashier-01"
	devOld := &models.Device{
		ID:           "dev-old",
		Hostname:     "cashier-01",
		Branch:       "Surabaya",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	if err := db.UpsertDevice(devOld); err != nil {
		t.Fatal(err)
	}

	// Device baru mendaftar dengan hostname sama ("cashier-01") tapi beda ID
	devDup := &models.Device{
		ID:           "dev-new",
		Hostname:     "cashier-01",
		Branch:       "Surabaya",
		RegisteredAt: time.Now(),
		LastSeen:     time.Now(),
	}
	if err := db.UpsertDevice(devDup); err != nil {
		t.Fatal(err)
	}
	loadedDup, err := db.GetDevice("dev-new")
	if err != nil {
		t.Fatal(err)
	}
	if loadedDup.ManualAssetID != "" {
		t.Fatalf("duplicate hostname must NOT be linked automatically, got: %s", loadedDup.ManualAssetID)
	}

	logs, err := db.GetLogs("dev-new", 10)
	if err != nil {
		t.Fatal(err)
	}
	foundWarning := false
	for _, l := range logs {
		if l["action"] == "asset_match_warning" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected asset_match_warning log for duplicate hostname, logs: %#v", logs)
	}
}
func TestUnlinkDeviceManualAsset(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test_unlink.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db)}

	dev := &models.Device{
		ID:            "dev-linked-01",
		Hostname:      "pc-office",
		Branch:        "Jakarta",
		OwnerUsername: "budi",
		ManualAssetID: "ast-001",
	}
	if err := db.UpsertDevice(dev); err != nil {
		t.Fatal(err)
	}

	// Directly set manual_asset_id to test unlink
	if _, err := db.db.Exec(`UPDATE devices SET manual_asset_id='ast-001' WHERE id='dev-linked-01'`); err != nil {
		t.Fatal(err)
	}

	call := func(role string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/devices/dev-linked-01/unlink", nil)
		req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: role, Branch: "Jakarta"}))
		w := httptest.NewRecorder()
		s.handleDevice(w, req)
		return w
	}

	// 1. Viewer harus ditolak
	if w := call("viewer"); w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for viewer, got %d", w.Code)
	}

	// 2. Admin sukses unlink
	if w := call("admin"); w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin unlink, got %d body=%s", w.Code, w.Body.String())
	}

	loaded, err := db.GetDevice("dev-linked-01")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ManualAssetID != "" {
		t.Fatalf("expected empty manual_asset_id after unlink, got: %s", loaded.ManualAssetID)
	}
}

func TestRelinkManualAssetOnHostnameChange(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test_relink.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, asset := range []*models.ManualAsset{
		{ID: "asset-old", AssetTag: "TAG-OLD", Name: "PC-OLD", Branch: "Jakarta"},
		{ID: "asset-new", AssetTag: "TAG-NEW", Name: "PC-NEW", Branch: "Jakarta"},
	} {
		if err := db.CreateManualAsset(asset); err != nil {
			t.Fatal(err)
		}
	}
	dev := &models.Device{ID: "dev-relink", Hostname: "PC-OLD", Branch: "Jakarta", RegisteredAt: time.Now(), LastSeen: time.Now()}
	if err := db.UpsertDevice(dev); err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE devices SET hostname='PC-NEW' WHERE id=?`, dev.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.RelinkManualAssetOnHostnameChange(dev.ID); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.GetDevice(dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ManualAssetID != "asset-new" {
		t.Fatalf("expected asset-new after hostname change, got %q", loaded.ManualAssetID)
	}
}
