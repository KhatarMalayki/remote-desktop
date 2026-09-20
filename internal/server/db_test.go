package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	// 2. Test CreateUser (ADH)
	adhPassHash := hashPassword("adh123")
	if err := db.CreateUser("adh_sby", adhPassHash, "adh", "Surabaya"); err != nil {
		t.Fatalf("failed creating adh user: %v", err)
	}
	_, _, kRole, kBranch, err := db.GetUser("adh_sby")
	if err != nil || kRole != "adh" || kBranch != "Surabaya" {
		t.Fatalf("failed getting adh user: role=%s branch=%s err=%v", kRole, kBranch, err)
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
		CreatedBy:  "adh_sby",
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
	if err := db.VerifyManualAsset(asset.ID, "verified", "good", "adh_sby", "Barang lengkap & berfungsi"); err != nil {
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

func TestBranchManagement(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	branches, err := db.ListBranches()
	if err != nil || len(branches) != 1 || branches[0].Name != "Pusat" {
		t.Fatalf("expected seeded Pusat location, got %#v (err=%v)", branches, err)
	}
	if err := db.CreateBranch("Surabaya", "cabang", "Operasional"); err != nil {
		t.Fatalf("create branch: %v", err)
	}
	if err := db.CreateBranch("Invalid", "", ""); err == nil {
		t.Fatal("expected empty type to fail")
	}

	branches, err = db.ListBranches()
	if err != nil || len(branches) != 2 {
		t.Fatalf("expected two locations, got %#v (err=%v)", branches, err)
	}
	var surabaya models.Branch
	for _, b := range branches {
		if b.Name == "Surabaya" {
			surabaya = b
		}
	}
	if surabaya.ID == 0 {
		t.Fatal("Surabaya location missing")
	}

	if err := db.CreateUser("adh_sby", hashPassword("password"), "adh", "Surabaya"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.UpdateBranch(surabaya.ID, "Surabaya Timur", "site", "Operasional"); err != nil {
		t.Fatalf("rename location: %v", err)
	}
	_, _, _, branch, err := db.GetUser("adh_sby")
	if err != nil || branch != "Surabaya Timur" {
		t.Fatalf("expected user assignment to follow rename, branch=%q err=%v", branch, err)
	}
	if err := db.DeleteBranch(surabaya.ID); err == nil {
		t.Fatal("expected deletion of used location to fail")
	}
	if err := db.DeleteBranch(branches[0].ID); err == nil {
		t.Fatal("expected Pusat deletion to fail")
	}
}

func TestTokenClaims(t *testing.T) {
	secret := "secret-jwt-key"
	tok := generateToken("adh_bdg", "adh", "Bandung", secret)
	claims, ok := parseToken(tok, secret)
	if !ok {
		t.Fatalf("failed to parse token")
	}
	if claims.Username != "adh_bdg" || claims.Role != "adh" || claims.Branch != "Bandung" {
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
	if !strings.Contains(w.Body.String(), `"version":"0.2.0"`) {
		t.Fatalf("expected configured server version, got %s", w.Body.String())
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
	if s.checkLoginRateLimit(ip) {
		t.Fatalf("expected rate limit to trigger after 5 failures")
	}
	recordLoginSuccess(ip)
	if !s.checkLoginRateLimit(ip) {
		t.Fatalf("expected rate limit cleared on success")
	}
}

func TestMFAWorkflow(t *testing.T) {
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
			JWTSecret: "test-secret-mfa",
		},
		db: db,
	}

	// 1. Check initial MFA status (should be disabled)
	en, _, err := db.GetUserMFA("admin")
	if err != nil || en {
		t.Fatalf("expected initial MFA disabled, got %v", en)
	}

	// 2. Setup MFA - generate secret
	secret := GenerateTOTPSecret()
	code, err := GenerateTOTPCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	// Enable MFA with valid code
	if !ValidateTOTPCode(secret, code) {
		t.Fatalf("expected valid code")
	}
	if err := db.SetUserMFA("admin", secret, true); err != nil {
		t.Fatalf("enable MFA in db: %v", err)
	}

	// 3. Test login when MFA is enabled
	// 3a. Login without code -> should require MFA
	bodyNoCode := strings.NewReader(`{"username":"admin","password":"admin123"}`)
	reqNoCode := httptest.NewRequest("POST", "/api/auth/login", bodyNoCode)
	wNoCode := httptest.NewRecorder()
	s.handleLogin(wNoCode, reqNoCode)
	if wNoCode.Code != 200 || !strings.Contains(wNoCode.Body.String(), "mfa_required") {
		t.Fatalf("expected mfa_required, got %s", wNoCode.Body.String())
	}

	// Extract mfa_ticket
	var resMFA map[string]interface{}
	_ = json.Unmarshal(wNoCode.Body.Bytes(), &resMFA)
	ticket, _ := resMFA["mfa_ticket"].(string)
	if ticket == "" {
		t.Fatalf("expected mfa_ticket in response")
	}

	// 3b. Verify second-step MFA login with code
	bodyMFA := strings.NewReader(fmt.Sprintf(`{"mfa_ticket":"%s","code":"%s"}`, ticket, code))
	reqMFA := httptest.NewRequest("POST", "/api/auth/login/mfa", bodyMFA)
	wMFA := httptest.NewRecorder()
	s.handleLoginMFA(wMFA, reqMFA)
	if wMFA.Code != 200 || !strings.Contains(wMFA.Body.String(), "token") {
		t.Fatalf("expected token after MFA login, got %s", wMFA.Body.String())
	}

	// 4. Test disable MFA
	if err := db.SetUserMFA("admin", "", false); err != nil {
		t.Fatalf("disable MFA: %v", err)
	}
	enAfter, _, _ := db.GetUserMFA("admin")
	if enAfter {
		t.Fatalf("expected MFA disabled")
	}
}

func TestChangeUsername(t *testing.T) {
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
			JWTSecret: "test-secret-un",
		},
		db: db,
	}

	// 1. Change username with wrong password -> fail
	bodyWrong := strings.NewReader(`{"new_username":"khatar_ops","password":"wrongpassword"}`)
	reqWrong := httptest.NewRequest("POST", "/api/auth/change-username", bodyWrong)
	ctx := context.WithValue(reqWrong.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"})
	wWrong := httptest.NewRecorder()
	s.handleChangeUsername(wWrong, reqWrong.WithContext(ctx))
	if wWrong.Code != 400 {
		t.Fatalf("expected 400, got %d", wWrong.Code)
	}

	// 2. Change username with correct password -> success
	bodyOK := strings.NewReader(`{"new_username":"khatar_ops","password":"admin123"}`)
	reqOK := httptest.NewRequest("POST", "/api/auth/change-username", bodyOK)
	wOK := httptest.NewRecorder()
	s.handleChangeUsername(wOK, reqOK.WithContext(ctx))
	if wOK.Code != 200 {
		t.Fatalf("expected 200, got %d body=%s", wOK.Code, wOK.Body.String())
	}

	// Verify old user doesn't exist and new user exists
	_, _, _, _, errOld := db.GetUser("admin")
	if errOld == nil {
		t.Fatalf("expected old username 'admin' to not exist")
	}
	_, _, role, _, errNew := db.GetUser("khatar_ops")
	if errNew != nil || role != "admin" {
		t.Fatalf("expected new username 'khatar_ops' to exist as admin, err=%v", errNew)
	}
}

func TestAuthLogs(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	// 1. Record failed attempt
	if err := db.RecordAuthLog("hacker", "1.2.3.4", "failed", "Password salah", "Mozilla/5.0"); err != nil {
		t.Fatalf("record auth log: %v", err)
	}

	// 2. Record success attempt
	if err := db.RecordAuthLog("admin", "192.168.1.10", "success", "Login berhasil", "Chrome/120"); err != nil {
		t.Fatalf("record auth log: %v", err)
	}

	// 3. Fetch logs
	logs, err := db.GetAuthLogs(10)
	if err != nil {
		t.Fatalf("get auth logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
	if logs[0].Username != "admin" || logs[0].Status != "success" {
		t.Fatalf("expected most recent log to be admin success, got %+v", logs[0])
	}
	if logs[1].Username != "hacker" || logs[1].Status != "failed" {
		t.Fatalf("expected older log to be hacker failed, got %+v", logs[1])
	}
}

func TestSecuritySettingsAndUnblock(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	// 1. Initial default settings
	defaults := db.GetSecuritySettings()
	if !defaults.RateLimitEnabled || defaults.MaxLoginAttempts != 5 || defaults.BlockDurationMin != 15 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	// 2. Save customized settings
	custom := models.SecuritySettings{
		RateLimitEnabled: true,
		MaxLoginAttempts: 3,
		BlockDurationMin: 30,
		IPWhitelist:      "192.168.1.100, 10.0.0.1",
	}
	if err := db.SaveSecuritySettings(custom); err != nil {
		t.Fatalf("save security settings: %v", err)
	}

	loaded := db.GetSecuritySettings()
	if loaded.MaxLoginAttempts != 3 || loaded.BlockDurationMin != 30 || loaded.IPWhitelist != "192.168.1.100, 10.0.0.1" {
		t.Fatalf("settings not persisted: %+v", loaded)
	}

	s := &Server{
		cfg: Config{
			Addr:      ":0",
			DBPath:    dbPath,
			JWTSecret: "test-secret-sec",
		},
		db: db,
	}

	// 3. Test whitelisted IP
	for i := 0; i < 10; i++ {
		recordLoginFail("192.168.1.100")
	}
	if !s.checkLoginRateLimit("192.168.1.100") {
		t.Fatalf("whitelisted IP should never be blocked")
	}

	// 4. Test non-whitelisted IP blocked after 3 attempts
	testIP := "203.0.113.50"
	for i := 0; i < 3; i++ {
		recordLoginFail(testIP)
	}
	if s.checkLoginRateLimit(testIP) {
		t.Fatalf("expected IP to be blocked after 3 attempts")
	}

	// Check getBlockedIPs
	blockedList := s.getBlockedIPs()
	found := false
	for _, b := range blockedList {
		if b.IP == testIP {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected blocked IP in list")
	}

	// 5. Unblock IP
	s.unblockIP(testIP)
	if !s.checkLoginRateLimit(testIP) {
		t.Fatalf("expected IP to be unblocked")
	}
}

func TestADHBranchIsolation(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	s := &Server{
		cfg: Config{Addr: ":0", DBPath: dbPath, JWTSecret: "sec-jwt"},
		db:  db,
	}

	// 1. Create asset for Surabaya
	sbyAsset := &models.ManualAsset{
		AssetTag: "SBY-01",
		Name:     "PC Kasir Surabaya",
		Branch:   "Surabaya",
	}
	_ = db.CreateManualAsset(sbyAsset)

	// 2. ADH Medan tries to edit Surabaya asset -> should be 403 Forbidden
	reqEdit := httptest.NewRequest("PUT", "/api/assets/manual/"+sbyAsset.ID, strings.NewReader(`{"name":"Hacked PC"}`))
	ctxMedan := context.WithValue(reqEdit.Context(), userClaimsKey, &UserClaims{Username: "adh_medan", Role: "adh", Branch: "Medan"})
	wEdit := httptest.NewRecorder()
	s.handleManualAsset(wEdit, reqEdit.WithContext(ctxMedan))
	if wEdit.Code != 403 {
		t.Fatalf("expected 403 when ADH Medan tries to edit Surabaya asset, got %d", wEdit.Code)
	}

	// 3. ADH Medan tries to delete Surabaya asset -> should be 403 Forbidden
	reqDel := httptest.NewRequest("DELETE", "/api/assets/manual/"+sbyAsset.ID, nil)
	wDel := httptest.NewRecorder()
	s.handleManualAsset(wDel, reqDel.WithContext(ctxMedan))
	if wDel.Code != 403 {
		t.Fatalf("expected 403 when ADH Medan tries to delete Surabaya asset, got %d", wDel.Code)
	}

	// 4. ADH Medan tries to verify Surabaya asset -> should be 403 Forbidden
	bodyVerify := strings.NewReader(fmt.Sprintf(`{"asset_id":"%s","asset_type":"manual","status":"verified"}`, sbyAsset.ID))
	reqVer := httptest.NewRequest("POST", "/api/assets/verify", bodyVerify)
	wVer := httptest.NewRecorder()
	s.handleVerifyAsset(wVer, reqVer.WithContext(ctxMedan))
	if wVer.Code != 403 {
		t.Fatalf("expected 403 when ADH Medan tries to verify Surabaya asset, got %d", wVer.Code)
	}

	// 5. ADH Medan creates new asset -> must be locked to Medan even if passing Surabaya
	bodyCreate := strings.NewReader(`{"asset_tag":"MDN-01","name":"Printer Medan","branch":"Surabaya"}`)
	reqCreate := httptest.NewRequest("POST", "/api/assets/manual", bodyCreate)
	wCreate := httptest.NewRecorder()
	s.handleManualAssets(wCreate, reqCreate.WithContext(ctxMedan))
	if wCreate.Code != 201 {
		t.Fatalf("expected 201 on create, got %d", wCreate.Code)
	}
	var created models.ManualAsset
	_ = json.Unmarshal(wCreate.Body.Bytes(), &created)
	if created.Branch != "Medan" {
		t.Fatalf("expected branch forced to 'Medan', got %s", created.Branch)
	}
}

func TestAgentPackageDownload(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed db: %v", err)
	}
	defer db.Close()

	// Write dummy agent
	dummyAgent := filepath.Join(tempDir, "rd-agent-windows-amd64.exe")
	_ = os.WriteFile(dummyAgent, []byte("MOCK_EXE_CONTENT"), 0755)

	s := &Server{
		cfg: Config{
			Addr:      ":0",
			DBPath:    dbPath,
			APIKey:    "my-test-api-key",
			AgentsDir: tempDir,
		},
		db: db,
	}

	// 1. ADH Medan requests package
	req := httptest.NewRequest("GET", "/api/agent/package?os=windows&arch=amd64", nil)
	req.Host = "testserver.example.com"
	ctx := context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "adh_medan", Role: "adh", Branch: "Medan"})
	w := httptest.NewRecorder()
	s.handleAgentPackageDownload(w, req.WithContext(ctx))

	if w.Code != 200 {
		t.Fatalf("expected 200 for zip download, got %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "RemoteDesk-Agent-Medan-vcurrent.zip") {
		t.Fatalf("expected versioned filename, got %s", w.Header().Get("Content-Disposition"))
	}
	if got := w.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("agent package must not be cacheable, got Cache-Control=%q", got)
	}

	// 2. Inspect ZIP contents
	zipBytes := w.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("read zip archive: %v", err)
	}

	foundExe := false
	foundCfg := false
	foundBat := false
	foundInstaller := false

	for _, f := range zr.File {
		if f.Name == "rd-agent.exe" {
			foundExe = true
		}
		if f.Name == "run-agent.bat" {
			foundBat = true
		}
		if f.Name == "install-service.ps1" {
			foundInstaller = true
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			script := string(data)
			if !strings.Contains(script, `sc.exe create`) || !strings.Contains(script, `RemoteDeskAgent`) || !strings.Contains(script, `LocalSystem`) || !strings.Contains(script, `Start-Service`) {
				t.Fatalf("invalid system service installer: %s", script)
			}
			if !strings.Contains(script, `Stop-Service -Name $serviceName`) || !strings.Contains(script, `WaitForStatus('Stopped'`) || !strings.Contains(script, `Get-Process -Name "rd-agent"`) || !strings.Contains(script, `Copy-Item`) {
				t.Fatalf("installer must stop the old service before replacing its binary: %s", script)
			}
			if !strings.Contains(script, `Add-Member -NotePropertyName "device_id"`) {
				t.Fatalf("installer must preserve the device identity for packages without device_id: %s", script)
			}
		}
		if f.Name == "pasang-otomatis.bat" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			bat := string(data)
			if !strings.Contains(bat, `-File "%~dp0install-service.ps1"`) || !strings.Contains(bat, "-Verb RunAs") || !strings.Contains(bat, "Unblock-File") || !strings.Contains(bat, "if errorlevel 1") {
				t.Fatalf("installer batch is missing elevated system-service setup: %s", bat)
			}
		}
		if f.Name == "agent.json" {
			foundCfg = true
			rc, _ := f.Open()
			cfgData, _ := io.ReadAll(rc)
			rc.Close()
			if !strings.Contains(string(cfgData), `"branch": "Medan"`) {
				t.Fatalf("expected agent.json to have branch Medan, got: %s", string(cfgData))
			}
			if !strings.Contains(string(cfgData), `"api_key": "my-test-api-key"`) {
				t.Fatalf("expected agent.json to have api key, got: %s", string(cfgData))
			}
			if !strings.Contains(string(cfgData), `"device_id": ""`) {
				t.Fatalf("expected agent.json to reserve device_id, got: %s", string(cfgData))
			}
		}
	}

	if !foundExe || !foundCfg || !foundBat || !foundInstaller {
		t.Fatalf("zip archive missing expected files: exe=%v cfg=%v bat=%v serviceInstaller=%v", foundExe, foundCfg, foundBat, foundInstaller)
	}
}
