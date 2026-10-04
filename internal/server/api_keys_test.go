package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/user/remote-desktop/internal/models"
)

func TestRetainShortLegacyKeyWithoutWeakeningNewKeys(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/keys.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db), cfg: Config{APIKey: "current-key-for-tests"}}
	if err := s.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	client := &Client{Send: make(chan []byte, 1)}
	s.hub.agents["pilot"] = client
	call := func(operation, key string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"operation": operation, "api_key": key})
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Role: "admin"}))
		response := httptest.NewRecorder()
		s.handleKeyRotation(response, req)
		return response
	}
	legacy := "legacy#1234"
	if response := call("prepare", legacy); response.Code != 400 {
		t.Fatal("short new key accepted")
	}
	if response := call("retain", legacy); response.Code != 200 {
		t.Fatalf("legacy recovery rejected: %s", response.Body.String())
	}
	if !s.acceptsAPIKey(legacy) || s.currentAPIKey() != "current-key-for-tests" || len(client.Send) != 0 {
		t.Fatal("recovery changed current key or broadcast")
	}
	restarted := &Server{db: db, cfg: Config{APIKey: "different-bootstrap"}}
	if err := restarted.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	if !restarted.acceptsAPIKey(legacy) {
		t.Fatal("recovered key lost after restart")
	}
	for _, key := range []string{"", "   ", "bad\nkey", strings.Repeat("x", 513)} {
		if response := call("retain", key); response.Code != 400 {
			t.Fatal("invalid recovery key accepted")
		}
	}
}

func TestAPIKeyRotationIsExplicitAndPersistent(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/keys.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db), cfg: Config{APIKey: "original-key-for-tests"}}
	if err := s.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pilot", "offline"} {
		if err := db.UpsertDevice(&models.Device{ID: id, OS: "windows", Version: "0.2.62"}); err != nil {
			t.Fatal(err)
		}
	}
	pilot := &Client{DeviceID: "pilot", Send: make(chan []byte, 2)}
	s.hub.agents["pilot"] = pilot
	call := func(operation, key, id string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]string{"operation": operation, "api_key": key, "device_id": id})
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(body)))
		req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Role: "admin"}))
		response := httptest.NewRecorder()
		s.handleKeyRotation(response, req)
		return response
	}
	key := "special-#&+%?= key-中文"
	if response := call("prepare", key, ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if len(pilot.Send) != 0 {
		t.Fatal("prepare broadcast a key")
	}
	if !s.acceptsAPIKey("original-key-for-tests") || !s.acceptsAPIKey(key) {
		t.Fatal("rotation lost an accepted key")
	}
	if s.deviceKeyStatus("pilot") != "unknown" {
		t.Fatal("unverified device marked migrated")
	}
	if response := call("migrate", "", "offline"); response.Code != 409 {
		t.Fatal("offline migration accepted")
	}
	if response := call("migrate", "", "pilot"); response.Code != 202 {
		t.Fatal(response.Body.String())
	}
	if len(pilot.Send) != 1 || s.deviceKeyStatus("pilot") != "unknown" {
		t.Fatal("delivery is not verification")
	}
	<-pilot.Send
	if err := s.recordDeviceKey("pilot", key); err != nil {
		t.Fatal(err)
	}
	if err := s.recordDeviceKey("offline", "original-key-for-tests"); err != nil {
		t.Fatal(err)
	}
	if s.deviceKeyStatus("pilot") != "current" || s.deviceKeyStatus("offline") != "old" {
		t.Fatal("incorrect key status")
	}
	if response := call("revoke", "", ""); response.Code != 409 {
		t.Fatal("revoked key needed by offline device")
	}
	restarted := &Server{db: db, cfg: Config{APIKey: "different-yaml-key"}}
	if err := restarted.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	if restarted.currentAPIKey() != key || !restarted.acceptsAPIKey("original-key-for-tests") || restarted.acceptsAPIKey("different-yaml-key") {
		t.Fatal("restart changed authoritative key state")
	}
	if restarted.deviceKeyStatus("pilot") != "current" {
		t.Fatal("restart lost verification")
	}
	if response := call("prepare", "second-special-#&+key", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if !s.acceptsAPIKey(key) || !s.acceptsAPIKey("original-key-for-tests") {
		t.Fatal("second rotation lost old keys")
	}
	for _, id := range []string{"pilot", "offline"} {
		if err := s.recordDeviceKey(id, s.currentAPIKey()); err != nil {
			t.Fatal(err)
		}
	}
	if response := call("revoke", "", ""); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if s.acceptsAPIKey(key) || s.acceptsAPIKey("original-key-for-tests") {
		t.Fatal("revoked keys still accepted")
	}
	if err := restarted.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	if restarted.acceptsAPIKey("original-key-for-tests") {
		t.Fatal("restart revived revoked key")
	}
	if err := db.UpsertDevice(&models.Device{ID: "pilot", OS: "windows", Version: "0.2.61"}); err != nil {
		t.Fatal(err)
	}
	if response := call("migrate", "", "pilot"); response.Code != 409 {
		t.Fatal("unsafe old agent migration allowed")
	}
	if response := call("", "unsafe-broadcast-key", ""); response.Code != 400 {
		t.Fatal("legacy broadcast allowed")
	}
}

func TestOldKeyWebSocketDoesNotAutoMigrate(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/keys.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db), cfg: Config{APIKey: "new-key-for-testing"}, upgrader: websocket.Upgrader{}}
	if err := s.loadAPIKeys(); err != nil {
		t.Fatal(err)
	}
	s.keyMu.Lock()
	state := s.keys
	state.Old = []string{"old-#&+%?= key-testing"}
	err = s.saveAPIKeysLocked(state)
	s.keyMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(s.handleAgentWS))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"?"+url.Values{"key": {state.Old[0]}, "id": {"pilot"}}.Encode(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Second)
	for s.deviceKeyStatus("pilot") != "old" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.deviceKeyStatus("pilot") != "old" {
		t.Fatal("old key connection not recorded")
	}
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if _, raw, err := conn.ReadMessage(); err == nil {
		t.Fatalf("unexpected auto-migration message: %d bytes", len(raw))
	}
}

func TestAPIKeyRotationAuthorizationAndValidation(t *testing.T) {
	s := &Server{}
	for _, role := range []string{"", "user", "it_support"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if role != "" {
			req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Role: role}))
		}
		response := httptest.NewRecorder()
		s.handleKeyRotation(response, req)
		if response.Code != 403 {
			t.Fatalf("role %q accepted", role)
		}
	}
	for _, key := range []string{"", "short", strings.Repeat("x", 513), "control-key-123456\n"} {
		if validAPIKey(key) {
			t.Fatal("invalid key accepted")
		}
	}
	if !validAPIKey("special-#&+%?= key") {
		t.Fatal("special characters rejected")
	}
}
