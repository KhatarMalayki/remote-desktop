package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

func TestDeviceIdentityPersistenceValidationAndPermissions(t *testing.T) {
	server := securityServer(t)
	device := &models.Device{ID: "identity-test", Hostname: "Laptop", LastSeen: time.Now(), RegisteredAt: time.Now()}
	if err := server.db.UpsertDevice(device); err != nil {
		t.Fatal(err)
	}
	if err := server.db.UpdateDeviceOwner(device.ID, "holder"); err != nil {
		t.Fatal(err)
	}
	put := func(role, username string, body map[string]interface{}, expected int) {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("PUT", "/api/devices/"+device.ID, strings.NewReader(string(raw)))
		request = request.WithContext(context.WithValue(request.Context(), userClaimsKey, &UserClaims{Role: role, Username: username}))
		response := httptest.NewRecorder()
		server.handleDevice(response, request)
		if response.Code != expected {
			t.Fatalf("PUT %s: %d %s", role, response.Code, response.Body.String())
		}
	}
	put("admin", "admin", map[string]interface{}{"serial_number": " SN-123 ", "product_id": " PROD-456 ", "acquisition_year": 2024}, 200)
	if err := server.db.UpsertDevice(device); err != nil {
		t.Fatal(err)
	}
	put("admin", "admin", map[string]interface{}{"note": "legacy client without identity fields"}, 200)
	for _, body := range []map[string]interface{}{
		{"serial_number": strings.Repeat("x", 129), "acquisition_year": 2020},
		{"serial_number": "bad\nserial"},
		{"product_id": "bad\x00product"},
		{"serial_number": "overwritten", "acquisition_year": 1960},
	} {
		put("admin", "admin", body, 400)
	}
	put("viewer", "viewer", map[string]interface{}{"serial_number": "overwritten"}, 403)
	put("user", "holder", map[string]interface{}{"serial_number": "overwritten", "product_id": "overwritten"}, 200)
	stored, err := server.db.GetDevice(device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SerialNumber != "SN-123" || stored.ProductID != "PROD-456" || stored.AcquisitionYear != 2024 {
		t.Fatalf("identity corrupted: %+v", stored)
	}
	for _, search := range []string{"SN-123", "PROD-456"} {
		rows, total, err := server.db.ListDevices("", search, 50, 0)
		if err != nil || total != 1 || len(rows) != 1 || rows[0].SerialNumber != "SN-123" {
			t.Fatalf("search %s: %d %v", search, total, err)
		}
		rows, total, err = server.db.ListDevicesForOwner("holder", search, 50, 0)
		if err != nil || total != 1 || len(rows) != 1 || rows[0].ProductID != "PROD-456" {
			t.Fatalf("owner search %s: %d %v", search, total, err)
		}
	}
	put("admin", "admin", map[string]interface{}{"serial_number": "", "product_id": ""}, 200)
	stored, err = server.db.GetDevice(device.ID)
	if err != nil || stored.SerialNumber != "" || stored.ProductID != "" {
		t.Fatalf("explicit clear failed: %v", err)
	}
}

func TestDeviceIdentityMigrationAndReopen(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"legacy.db")
	db,err:=NewDB(path)
	if err!=nil {t.Fatal(err)}
	for _,column:=range []string{"serial_number","product_id"} {
		if _,err=db.db.Exec("ALTER TABLE devices DROP COLUMN "+column);err!=nil {db.Close();t.Fatal(err)}
	}
	db.Close()
	db,err=NewDB(path)
	if err!=nil {t.Fatal(err)}
	device:=&models.Device{ID:"migrated",Hostname:"Laptop",LastSeen:time.Now(),RegisteredAt:time.Now()}
	if err=db.UpsertDevice(device);err!=nil {db.Close();t.Fatal(err)}
	device.SerialNumber,device.ProductID="SN-MIGRATED","PRODUCT-MIGRATED"
	if err=db.UpdateDeviceMeta(device);err!=nil {db.Close();t.Fatal(err)}
	db.Close()
	db,err=NewDB(path)
	if err!=nil {t.Fatal(err)}
	defer db.Close()
	stored,err:=db.GetDevice(device.ID)
	if err!=nil || stored.SerialNumber!=device.SerialNumber || stored.ProductID!=device.ProductID {t.Fatalf("reopen lost identity: %v",err)}
}
