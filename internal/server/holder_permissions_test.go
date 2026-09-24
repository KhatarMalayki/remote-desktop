package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/remote-desktop/internal/models"
)

func TestHolderManagementPermissions(t *testing.T) {
	s := securityServer(t)
	for _, role := range []string{"admin", "ga_pusat", "it_support", "adh", "viewer"} {
		branch := ""
		if role == "adh" {
			branch = "Bandung"
		}
		if err := s.db.CreateUser(role, "fixture", role, branch); err != nil {
			t.Fatal(err)
		}
	}
	call := func(handler http.HandlerFunc, method, url, body, role string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		who := &UserClaims{Username: role, Role: role, Branch: "Bandung"}
		response := httptest.NewRecorder()
		handler(response, req.WithContext(context.WithValue(req.Context(), userClaimsKey, who)))
		return response
	}
	for _, role := range []string{"admin", "ga_pusat", "it_support", "adh", "viewer"} {
		t.Run(role, func(t *testing.T) {
			asset := &models.ManualAsset{AssetTag: role, Name: "Laptop", Branch: "Bandung", OwnerUsername: "adh", AssignedTo: "Budi"}
			if err := s.db.CreateManualAsset(asset); err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`{"asset_id":%q,"asset_type":"manual","to_owner":"admin","assigned_to":"Sari tanpa akun","reason":"Serah terima kepada karyawan baru"}`, asset.ID)
			response := call(s.handleSwitchRequests, "POST", "/api/assets/switch-requests", body, role)
			if role == "viewer" {
				if response.Code != 403 {
					t.Fatalf("viewer: %d %s", response.Code, response.Body.String())
				}
				return
			}
			if response.Code != 201 {
				t.Fatalf("request: %d %s", response.Code, response.Body.String())
			}
			var request models.AssetSwitchRequest
			if err := json.Unmarshal(response.Body.Bytes(), &request); err != nil {
				t.Fatal(err)
			}
			if role == "it_support" {
				if request.Status != "pending" {
					t.Fatal("IT auto-approved")
				}
				saved, _ := s.db.GetManualAsset(asset.ID)
				if saved.AssignedTo != "Budi" {
					t.Fatal("pending request changed holder")
				}
				url := fmt.Sprintf("/api/assets/switch-requests/%d/approve", request.ID)
				for _, denied := range []string{"it_support", "adh"} {
					if response := call(s.handleSwitchRequestSubroute, "POST", url, "{}", denied); response.Code != 403 {
						t.Fatalf("%s approved IT request", denied)
					}
				}
				if response := call(s.handleSwitchRequestSubroute, "POST", url, "{}", "ga_pusat"); response.Code != 200 {
					t.Fatalf("GA approval: %s", response.Body.String())
				}
			} else if request.Status != "approved" {
				t.Fatal("authorized role did not assign holder")
			}
			saved, _ := s.db.GetManualAsset(asset.ID)
			if saved.AssignedTo != "Sari tanpa akun" || saved.OwnerUsername != "admin" {
				t.Fatalf("holder not assigned: %+v", saved)
			}
			response = call(s.handleManualAsset, "PUT", "/api/assets/manual/"+asset.ID, `{"assigned_to":"Bypass"}`, "it_support")
			if response.Code != 200 {
				t.Fatalf("metadata update: %s", response.Body.String())
			}
			saved, _ = s.db.GetManualAsset(asset.ID)
			if saved.AssignedTo != "Sari tanpa akun" {
				t.Fatal("metadata bypassed approval")
			}
		})
	}
	if eligibleHolderAt("adh", "Medan", "Bandung") || eligibleHolderAt("viewer", "Bandung", "Bandung") {
		t.Fatal("invalid PIC scope")
	}
	options, err := s.db.ListHolderOptions("Bandung")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, option := range options {
		if option.Username == "admin" {
			found = true
		}
	}
	if !found {
		t.Fatal("admin missing from PIC options")
	}
}
