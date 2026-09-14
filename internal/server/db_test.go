package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"os"
	"path/filepath"
	"testing"

	"github.com/user/remote-desktop/internal/models"
)

func TestDBAndBranchVerification(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// 1. Test EnsureAdmin and GetUser
	adminPassHash := hashPassword("admin123")
	if err := db.EnsureAdmin("admin", adminPassHash); err != nil {
		t.Fatalf("ensure admin failed: %v", err)
	}
	id, hash, role, branch, err := db.GetUser("admin")
	if err != nil || role != "admin" || hash != adminPassHash {
		t.Fatalf("failed getting admin user: err=%v role=%s id=%d branch=%s", err, role, id, branch)
	}

	// 2. Test CreateUser (Kacab)
	kacabPassHash := hashPassword("kacab123")
	if err := db.CreateUser("kacab_sby", kacabPassHash, "kacab", "Surabaya"); err != nil {
		t.Fatalf("failed creating kacab user: %v", err)
	}
	_, _, kRole, kBranch, err := db.GetUser("kacab_sby")
	if err != nil || kRole != "kacab" || kBranch != "Surabaya" {
		t.Fatalf("failed getting kacab user: role=%s branch=%s err=%v", kRole, kBranch, err)
	}

	// 3. Test Manual Assets CRUD
	asset := &models.ManualAsset{
		AssetTag:   "AST-SBY-001",
		Name:       "Printer Kasir Thermal",
		Category:   "printer",
		Branch:     "Surabaya",
		Location:   "Meja Kasir 1",
		AssignedTo: "Budi",
		Condition:  "good",
		CreatedBy:  "kacab_sby",
	}
	if err := db.CreateManualAsset(asset); err != nil {
		t.Fatalf("create manual asset failed: %v", err)
	}
	if asset.ID == "" {
		t.Fatalf("expected asset ID to be generated")
	}

	assets, err := db.ListManualAssets("Surabaya", "", "", "")
	if err != nil || len(assets) != 1 {
		t.Fatalf("expected 1 asset in Surabaya, got %d, err=%v", len(assets), err)
	}

	// 4. Test Verification
	if err := db.VerifyManualAsset(asset.ID, "verified", "good", "kacab_sby", "Barang lengkap & berfungsi"); err != nil {
		t.Fatalf("verify manual asset failed: %v", err)
	}

	updatedAsset, err := db.GetManualAsset(asset.ID)
	if err != nil {
		t.Fatalf("get manual asset failed: %v", err)
	}
	if updatedAsset.VerificationStatus != "verified" || updatedAsset.VerifiedBy == "" {
		t.Fatalf("verification status not updated properly: %+v", updatedAsset)
	}

	// 5. Test Verification History
	vers, err := db.GetAssetVerifications(asset.ID, "Surabaya", 10)
	if err != nil || len(vers) != 1 {
		t.Fatalf("expected 1 verification log, got %d, err=%v", len(vers), err)
	}
	if vers[0].Status != "verified" {
		t.Fatalf("expected status verified, got %s", vers[0].Status)
	}

	// 6. Test Branch Stats
	stats, err := db.GetBranchStats("Surabaya")
	if err != nil {
		t.Fatalf("get branch stats failed: %v", err)
	}
	if stats["total_assets"] != 1 || stats["verified"] != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestTokenClaims(t *testing.T) {
	secret := "secret-jwt-key"
	tok := generateToken("kacab_bdg", "kacab", "Bandung", secret)
	claims, ok := parseToken(tok, secret)
	if !ok {
		t.Fatalf("failed to parse token")
	}
	if claims.Username != "kacab_bdg" || claims.Role != "kacab" || claims.Branch != "Bandung" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestAgentVersionEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	cfg := Config{
		Addr:      ":0",
		DBPath:    dbPath,
		APIKey:    "test-key",
		AdminUser: "admin",
		AdminPass: "admin123",
		Version:   "0.2.0",
		AgentsDir: tempDir,
	}

	// Create a dummy agent file
	dummyAgent := filepath.Join(tempDir, "rd-agent-windows-amd64.exe")
	if err := os.WriteFile(dummyAgent, []byte("fake-agent-binary-content"), 0755); err != nil {
		t.Fatalf("write dummy agent: %v", err)
	}

	s := &Server{
		cfg: cfg,
		db:  db,
		hub: NewHub(db),
	}

	// Test /api/agent/version
	req := httptest.NewRequest("GET", "/api/agent/version", nil)
	w := httptest.NewRecorder()
	s.handleAgentVersion(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Test /api/agent/download
	reqDl := httptest.NewRequest("GET", "/api/agent/download?os=windows&arch=amd64&key=test-key", nil)
	wDl := httptest.NewRecorder()
	s.handleAgentDownload(wDl, reqDl)

	if wDl.Code != 200 {
		t.Fatalf("expected 200 for download, got %d", wDl.Code)
	}
	if wDl.Body.String() != "fake-agent-binary-content" {
		t.Fatalf("unexpected content: %s", wDl.Body.String())
	}
}

func TestChangePasswordAndRateLimit(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	if err := db.EnsureAdmin("admin", hashPassword("admin123")); err != nil {
		t.Fatalf("ensure admin: %v", err)
	}

	s := &Server{
		cfg: Config{
			Addr:      ":0",
			DBPath:    dbPath,
			JWTSecret: "test-secret",
		},
		db: db,
	}

	// 1. Test change password with wrong old password
	bodyWrong := strings.NewReader(`{"old_password":"wrong","new_password":"newpassword123","confirm_password":"newpassword123"}`)
	reqWrong := httptest.NewRequest("POST", "/api/auth/change-password", bodyWrong)
	ctx := context.WithValue(reqWrong.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"})
	wWrong := httptest.NewRecorder()
	s.handleChangePassword(wWrong, reqWrong.WithContext(ctx))
	if wWrong.Code != 400 {
		t.Fatalf("expected 400 for wrong old pass, got %d", wWrong.Code)
	}

	// 2. Test change password with correct old password
	bodyOK := strings.NewReader(`{"old_password":"admin123","new_password":"newpassword123","confirm_password":"newpassword123"}`)
	reqOK := httptest.NewRequest("POST", "/api/auth/change-password", bodyOK)
	wOK := httptest.NewRecorder()
	s.handleChangePassword(wOK, reqOK.WithContext(ctx))
	if wOK.Code != 200 {
		t.Fatalf("expected 200 for valid change pass, got %d", wOK.Code)
	}

	// 3. Verify in database
	_, newHash, _, _, err := db.GetUser("admin")
	if err != nil || newHash != hashPassword("newpassword123") {
		t.Fatalf("password not updated in db: %v", err)
	}

	// 4. Test rate limiting
	ip := "192.168.99.1"
	for i := 0; i < 5; i++ {
		recordLoginFail(ip)
	}
	if checkLoginRateLimit(ip) {
		t.Fatalf("expected rate limit to trigger after 5 failures")
	}
	recordLoginSuccess(ip)
	if !checkLoginRateLimit(ip) {
		t.Fatalf("expected rate limit cleared on success")
	}
}
