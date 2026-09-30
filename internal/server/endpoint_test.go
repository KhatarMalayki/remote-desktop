package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/user/remote-desktop/internal/models"
)

func TestTrackedApplicationsAPIAndDetection(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "endpoint.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db}

	apps, err := s.db.trackedApplications()
	if err != nil || len(apps) != 3 {
		t.Fatalf("default apps: %v err=%v", apps, err)
	}

	reqBad := httptest.NewRequest(http.MethodPut, "/api/endpoint/applications", bytes.NewReader([]byte(`[{"label":"TooLongLabelOver24CharsHere","match":"AnyDesk"}]`)))
	reqBad = reqBad.WithContext(context.WithValue(reqBad.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
	recBad := httptest.NewRecorder()
	s.handleTrackedApplications(recBad, reqBad)
	if recBad.Code != 400 {
		t.Fatalf("expected 400 bad tracked app, got %d", recBad.Code)
	}

	payload, _ := json.Marshal([]models.TrackedApplication{
		{Label: "ME", Match: "ManageEngine|Desktop Central"},
		{Label: "SE", Match: "SentinelOne"},
		{Label: "AnyDesk", Match: "AnyDesk"},
		{Label: "Custom", Match: "CustomApp"},
	})
	reqOK := httptest.NewRequest(http.MethodPut, "/api/endpoint/applications", bytes.NewReader(payload))
	reqOK = reqOK.WithContext(context.WithValue(reqOK.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
	recOK := httptest.NewRecorder()
	s.handleTrackedApplications(recOK, reqOK)
	if recOK.Code != 200 {
		t.Fatalf("save tracked apps: got %d %s", recOK.Code, recOK.Body.String())
	}

	report := &models.EndpointReport{
		Status: "ok",
		Applications: []models.InstalledApplication{
			{Name: "Sentinel Agent v23.1", Version: "23.1.0"},
			{Name: "CustomApp Service", Service: "running"},
		},
	}
	detected := detectApplications([]models.TrackedApplication{
		{Label: "ME", Match: "ManageEngine"},
		{Label: "SE", Match: "Sentinel"},
		{Label: "Custom", Match: "CustomApp"},
	}, report)
	if len(detected) != 3 || detected[0].Status != "not_detected" || detected[1].Status != "detected" || detected[2].Status != "detected" {
		t.Fatalf("unexpected detections: %+v", detected)
	}
}

func TestLockPolicyEndpointAuthorizationAndQueue(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "endpoint_policy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, hub: NewHub(db)}
	device := &models.Device{ID: "dev-test-01", Hostname: "PC-TEST", OS: "windows", Version: "0.2.58"}
	if err := db.UpsertDevice(device); err != nil {
		t.Fatal(err)
	}

	body := bytes.NewReader([]byte(`{"device":"dev-test-01","mode":"audit","seconds":900}`))
	reqForbidden := httptest.NewRequest(http.MethodPost, "/api/endpoint/lock-policy", body)
	reqForbidden = reqForbidden.WithContext(context.WithValue(reqForbidden.Context(), userClaimsKey, &UserClaims{Username: "support", Role: "it_support"}))
	recForbidden := httptest.NewRecorder()
	s.handleLockPolicy(recForbidden, reqForbidden)
	if recForbidden.Code != 403 {
		t.Fatalf("hanya admin: got %d", recForbidden.Code)
	}

	reqOK := httptest.NewRequest(http.MethodPost, "/api/endpoint/lock-policy", bytes.NewReader([]byte(`{"device":"dev-test-01","mode":"audit","seconds":900}`)))
	reqOK = reqOK.WithContext(context.WithValue(reqOK.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
	recOK := httptest.NewRecorder()
	s.handleLockPolicy(recOK, reqOK)
	if recOK.Code != 202 {
		t.Fatalf("expected 202 queued: got %d %s", recOK.Code, recOK.Body.String())
	}

	var res map[string]string
	if err := json.Unmarshal(recOK.Body.Bytes(), &res); err != nil || len(res["id"]) != 32 {
		t.Fatalf("invalid queue response: %s", recOK.Body.String())
	}

	s.receiveLockPolicyResult("dev-test-01", json.RawMessage(`{"id":"`+res["id"]+`","status":"audited","detail":"tidak ada perubahan"}`))
	reloaded, _ := s.db.GetDevice("dev-test-01")
	s.enrichEndpoint(reloaded)
	if reloaded.Endpoint == nil || reloaded.Endpoint.Policy == nil || reloaded.Endpoint.Policy.Status != "audited" {
		t.Fatalf("policy state not enriched: %+v", reloaded.Endpoint)
	}
}
