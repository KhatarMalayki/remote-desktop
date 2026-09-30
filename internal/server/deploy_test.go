package server

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/user/remote-desktop/internal/models"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestDeploymentAuthorizationTransferAndResult(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	server := &Server{cfg: Config{DBPath: dbPath}, db: db, hub: NewHub(db)}
	if err = server.initDeployments(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pc-one", "pc-two"} {
		if err = db.UpsertDevice(&models.Device{ID: id, Hostname: id, OS: "windows", Version: "0.2.56"}); err != nil {
			t.Fatal(err)
		}
		server.hub.agents[id] = &Client{DeviceID: id, Send: make(chan []byte, 2)}
	}
	call := func(role, targets string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		_ = form.WriteField("targets", targets)
		_ = form.WriteField("args", "[]")
		file, _ := form.CreateFormFile("installer", "package.msi")
		_, _ = file.Write([]byte("test installer, not executable"))
		_ = form.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/deployments", &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "tester", Role: role}))
		response := httptest.NewRecorder()
		server.handleDeployments(response, req)
		return response
	}
	if response := call("ga_pusat", `["pc-one"]`); response.Code != 403 {
		t.Fatalf("role bypass: %d", response.Code)
	}
	if response := call("admin", `["pc-one","pc-one"]`); response.Code != 400 {
		t.Fatalf("duplicate target: %d", response.Code)
	}
	if response := call("it_support", `["pc-one","pc-two"]`); response.Code != 202 {
		t.Fatalf("submit: %d %s", response.Code, response.Body.String())
	}
	var message struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.Unmarshal(<-server.hub.agents["pc-one"].Send, &message); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/deployment-package", nil)
	response := httptest.NewRecorder()
	server.handleDeploymentPackage(response, req)
	if response.Code != 404 {
		t.Fatal("unauthorized package download")
	}
	req.Header.Set("Authorization", "Bearer "+message.Data.ID)
	response = httptest.NewRecorder()
	server.handleDeploymentPackage(response, req)
	if response.Code != 200 || response.Body.String() != "test installer, not executable" {
		t.Fatalf("package: %d", response.Code)
	}
	payload, _ := json.Marshal(map[string]string{"id": message.Data.ID, "status": "succeeded", "detail": "exit 0"})
	server.handleDeploymentResult("pc-two", payload)
	var status string
	_ = db.db.QueryRow("SELECT status FROM deployments WHERE id=?", message.Data.ID).Scan(&status)
	if status != "queued" {
		t.Fatal("another device spoofed result")
	}
	server.handleDeploymentResult("pc-one", payload)
	_ = db.db.QueryRow("SELECT status FROM deployments WHERE id=?", message.Data.ID).Scan(&status)
	if status != "succeeded" {
		t.Fatal("result not persisted")
	}
	response = httptest.NewRecorder()
	server.handleDeploymentPackage(response, req)
	if response.Code != 404 {
		t.Fatal("completed deployment token still allowed")
	}
}
