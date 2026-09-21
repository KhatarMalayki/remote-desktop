package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/remote-desktop/internal/models"
)

func TestRustDeskSettingsAndManage(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	hub := NewHub(db)
	hub.agents["pc-1"] = &Client{DeviceID: "pc-1", Send: make(chan []byte, 1)}
	s := &Server{db: db, hub: hub}

	call := func(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
		w := httptest.NewRecorder()
		handler(w, req)
		return w
	}

	config := "=" + strings.Repeat("a", 60)
	settingsBody, _ := json.Marshal(map[string]string{"config": `\` + config})
	if w := call(s.handleRustDeskSettings, string(settingsBody)); w.Code != http.StatusOK {
		t.Fatalf("save config status=%d body=%s", w.Code, w.Body.String())
	}
	if got := db.GetSystemSetting("rustdesk_config"); got != config {
		t.Fatalf("stored config=%q", got)
	}
	w := call(s.handleRustDeskManage, `{"device_id":"pc-1","operation":"install","password":"StrongPass123!"}`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("manage status=%d body=%s", w.Code, w.Body.String())
	}
	raw := <-hub.agents["pc-1"].Send
	var envelope struct {
		Action string                 `json:"action"`
		Data   models.RustDeskCommand `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Action != "rustdesk_manage" || envelope.Data.Config != config || envelope.Data.Password != "StrongPass123!" || envelope.Data.SHA256 == "" {
		t.Fatalf("unexpected command: %#v", envelope)
	}
}

func TestRustDeskManageRejectsWeakPassword(t *testing.T) {
	db, err := NewDB(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db)}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"device_id":"pc-1","operation":"set_password","password":"short"}`))
	req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
	w := httptest.NewRecorder()
	s.handleRustDeskManage(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
