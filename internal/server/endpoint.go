package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/user/remote-desktop/internal/models"
	"github.com/user/remote-desktop/internal/versioncmp"
)

var defaultTrackedApplications = []models.TrackedApplication{
	{Label: "ME", Match: "ManageEngine|Desktop Central|Endpoint Central"},
	{Label: "SE", Match: "SentinelOne|Sentinel Agent|SentinelAgent"},
	{Label: "AnyDesk", Match: "AnyDesk"},
}

func (db *DB) initEndpoints() error {
	_, err := db.db.Exec(`CREATE TABLE IF NOT EXISTS endpoint_reports (device TEXT PRIMARY KEY, report TEXT NOT NULL, checked INTEGER NOT NULL);
	CREATE TABLE IF NOT EXISTS endpoint_policy (device TEXT PRIMARY KEY, id TEXT NOT NULL, request TEXT NOT NULL, status TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '', actor TEXT NOT NULL, created INTEGER NOT NULL);`)
	return err
}

func (db *DB) trackedApplications() ([]models.TrackedApplication, error) {
	var raw string
	err := db.db.QueryRow(`SELECT value FROM system_settings WHERE key='tracked_applications'`).Scan(&raw)
	if err == sql.ErrNoRows {
		return defaultTrackedApplications, nil
	}
	if err != nil {
		return nil, err
	}
	var apps []models.TrackedApplication
	err = json.Unmarshal([]byte(raw), &apps)
	return apps, err
}

func validTrackedApplications(apps []models.TrackedApplication) bool {
	if len(apps) > 30 {
		return false
	}
	seen := map[string]bool{}
	for _, app := range apps {
		label := strings.TrimSpace(app.Label)
		if label == "" || len(label) > 24 || seen[strings.ToLower(label)] || len(app.Match) > 256 {
			return false
		}
		seen[strings.ToLower(label)] = true
		parts := strings.Split(app.Match, "|")
		if len(parts) > 8 {
			return false
		}
		for _, part := range parts {
			if len(strings.TrimSpace(part)) < 3 {
				return false
			}
		}
	}
	return true
}

func (s *Server) handleTrackedApplications(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" {
		jsonError(w, "hanya admin dapat mengelola daftar aplikasi", 403)
		return
	}
	switch r.Method {
	case http.MethodGet:
		apps, err := s.db.trackedApplications()
		if err != nil {
			jsonError(w, "gagal membaca daftar aplikasi", 500)
			return
		}
		jsonResp(w, apps, 200)
	case http.MethodPut:
		var apps []models.TrackedApplication
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&apps) != nil || !validTrackedApplications(apps) {
			jsonError(w, "maksimum 30 aplikasi; label unik 1–24 karakter; kata pencocokan minimal 3 karakter, pisahkan dengan |", 400)
			return
		}
		for index := range apps {
			apps[index].Label = strings.TrimSpace(apps[index].Label)
		}
		raw, _ := json.Marshal(apps)
		if err := s.db.SetSystemSetting("tracked_applications", string(raw)); err != nil {
			jsonError(w, "gagal menyimpan", 500)
			return
		}
		_ = s.db.AddLog("", "tracked_applications", claims.Username+": "+string(raw))
		jsonResp(w, map[string]string{"status": "ok"}, 200)
	default:
		http.Error(w, "method not allowed", 405)
	}
}

func detectApplications(apps []models.TrackedApplication, report *models.EndpointReport) []models.ApplicationDetection {
	result := make([]models.ApplicationDetection, 0, len(apps))
	for _, app := range apps {
		detection := models.ApplicationDetection{Label: app.Label, Status: "unknown"}
		if report != nil {
			if report.Status == "ok" {
				detection.Status = "not_detected"
			}
			for _, installed := range report.Applications {
				for _, pattern := range strings.Split(app.Match, "|") {
					if strings.Contains(strings.ToLower(installed.Name), strings.ToLower(strings.TrimSpace(pattern))) {
						detection.Status = "detected"
						if len(detection.Matches) < 10 {
							detection.Matches = append(detection.Matches, installed)
						}
						break
					}
				}
			}
		}
		result = append(result, detection)
	}
	return result
}

func (s *Server) enrichEndpoint(device *models.Device) {
	state := &models.EndpointState{}
	var raw string
	if err := s.db.db.QueryRow(`SELECT report,checked FROM endpoint_reports WHERE device=?`, device.ID).Scan(&raw, &state.CheckedAt); err == nil {
		var report models.EndpointReport
		if json.Unmarshal([]byte(raw), &report) == nil {
			state.Report = &report
		}
	}
	apps, _ := s.db.trackedApplications()
	state.Applications = detectApplications(apps, state.Report)
	var result models.LockPolicyResult
	if s.db.db.QueryRow(`SELECT id,status,detail FROM endpoint_policy WHERE device=?`, device.ID).Scan(&result.ID, &result.Status, &result.Detail) == nil {
		state.Policy = &result
	}
	device.Endpoint = state
}

func (s *Server) handleLockPolicy(w http.ResponseWriter, r *http.Request) {
	if getClaims(r).Role != "admin" {
		jsonError(w, "hanya admin dapat mengirim policy", 403)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var input struct {
		Device  string `json:"device"`
		Mode    string `json:"mode"`
		Seconds uint32 `json:"seconds"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil {
		jsonError(w, "permintaan tidak valid", 400)
		return
	}
	device, err := s.db.GetDevice(input.Device)
	if err != nil {
		jsonError(w, "device tidak ditemukan", 404)
		return
	}
	if device.OS != "windows" || versioncmp.IsNewer("0.2.58", device.Version) {
		jsonError(w, "memerlukan Windows agent >=0.2.58", 409)
		return
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		jsonError(w, "gagal membuat request", 500)
		return
	}
	request := models.LockPolicyRequest{ID: hex.EncodeToString(random[:]), Mode: input.Mode, Seconds: input.Seconds}
	if !request.Valid() {
		jsonError(w, "mode audit/apply/restore; waktu 60–86400 detik", 400)
		return
	}
	raw, _ := json.Marshal(request)
	result, err := s.db.db.Exec(`INSERT INTO endpoint_policy(device,id,request,status,detail,actor,created) VALUES(?,?,?,'pending','Menunggu agent; kedaluwarsa 24 jam',?,?) ON CONFLICT(device) DO UPDATE SET id=excluded.id,request=excluded.request,status=excluded.status,detail=excluded.detail,actor=excluded.actor,created=excluded.created WHERE endpoint_policy.status NOT IN ('pending','delivered')`, device.ID, request.ID, string(raw), getClaims(r).Username, time.Now().Unix())
	if err != nil {
		jsonError(w, "gagal menyimpan policy", 500)
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		jsonError(w, "policy sebelumnya masih menunggu hasil; refresh sebelum mengirim ulang", 409)
		return
	}
	_ = s.db.AddLog(device.ID, "lock_policy_requested", getClaims(r).Username+": "+string(raw))
	s.dispatchLockPolicy(device.ID)
	jsonResp(w, map[string]string{"status": "queued", "id": request.ID}, 202)
}

func (s *Server) dispatchLockPolicy(deviceID string) {
	var raw, id, status string
	var created int64
	if s.db.db.QueryRow(`SELECT request,id,status,created FROM endpoint_policy WHERE device=? AND status IN ('pending','delivered')`, deviceID).Scan(&raw, &id, &status, &created) != nil {
		return
	}
	if time.Now().Unix()-created > 86400 {
		_, _ = s.db.db.Exec(`UPDATE endpoint_policy SET status='unknown',detail='Kedaluwarsa; periksa device sebelum mengirim ulang' WHERE device=? AND id=? AND status IN ('pending','delivered')`, deviceID, id)
		return
	}
	if s.hub == nil || !s.hub.IsOnline(deviceID) {
		return
	}
	device, err := s.db.GetDevice(deviceID)
	if err != nil || device.OS != "windows" || versioncmp.IsNewer("0.2.58", device.Version) {
		return
	}
	message, _ := json.Marshal(map[string]interface{}{"action": "lock_policy", "data": json.RawMessage(raw)})
	if s.hub.SendToAgent(deviceID, message) {
		_, _ = s.db.db.Exec(`UPDATE endpoint_policy SET status='delivered',detail='Dikirim; menunggu konfirmasi agent' WHERE device=? AND id=? AND status='pending'`, deviceID, id)
	}
}

func (s *Server) receiveEndpointReport(deviceID string, raw json.RawMessage) {
	if len(raw) > 1024*1024 {
		return
	}
	var report models.EndpointReport
	if json.Unmarshal(raw, &report) != nil || len(report.Applications) > 2000 || (report.Status != "ok" && report.Status != "unknown" && report.Status != "unsupported") {
		return
	}
	for _, app := range report.Applications {
		if len(app.Name) > 512 || len(app.Version) > 128 || len(app.Service) > 64 {
			return
		}
	}
	if len(report.Detail) > 1024 || len(report.PolicyError) > 1024 {
		return
	}
	_, _ = s.db.db.Exec(`INSERT INTO endpoint_reports(device,report,checked) VALUES(?,?,?) ON CONFLICT(device) DO UPDATE SET report=excluded.report,checked=excluded.checked`, deviceID, string(raw), time.Now().Unix())
}

func (s *Server) receiveLockPolicyResult(deviceID string, raw json.RawMessage) {
	var result models.LockPolicyResult
	if len(raw) > 4096 || json.Unmarshal(raw, &result) != nil || len(result.ID) != 32 || len(result.Detail) > 2048 {
		return
	}
	switch result.Status {
	case "audited", "configured", "restored", "conflict", "failed", "unknown", "unsupported":
	default:
		return
	}
	updated, err := s.db.db.Exec(`UPDATE endpoint_policy SET status=?,detail=? WHERE device=? AND id=? AND status IN ('pending','delivered')`, result.Status, result.Detail, deviceID, result.ID)
	if err == nil {
		if count, _ := updated.RowsAffected(); count > 0 {
			_ = s.db.AddLog(deviceID, "lock_policy_result", fmt.Sprintf("%s: %s %s", result.ID, result.Status, result.Detail))
		}
	}
}
