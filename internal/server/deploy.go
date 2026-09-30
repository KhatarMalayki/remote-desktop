package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/user/remote-desktop/internal/versioncmp"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) initDeployments() error {
	_, err := s.db.db.Exec(`CREATE TABLE IF NOT EXISTS deployments (id TEXT PRIMARY KEY, device TEXT NOT NULL, name TEXT NOT NULL, package TEXT NOT NULL, digest TEXT NOT NULL, args TEXT NOT NULL, status TEXT NOT NULL, detail TEXT NOT NULL DEFAULT '', created INTEGER NOT NULL)`)
	return err
}

func (s *Server) handleDeployments(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if claims.Role != "admin" && claims.Role != "it_support" {
		jsonError(w, "forbidden", 403)
		return
	}
	if r.Method == http.MethodGet {
		rows, err := s.db.db.Query("SELECT id,device,name,status,detail,created FROM deployments ORDER BY created DESC LIMIT 200")
		if err != nil {
			jsonError(w, "database unavailable", 500)
			return
		}
		defer rows.Close()
		result := []map[string]interface{}{}
		for rows.Next() {
			var id, device, name, status, detail string
			var created int64
			if err := rows.Scan(&id, &device, &name, &status, &detail, &created); err != nil {
				jsonError(w, "database read failed", 500)
				return
			}
			if (status == "queued" || status == "running") && time.Now().Unix()-created > 1800 {
				status = "unknown"
				detail = "Tidak ada hasil final; periksa target sebelum mencoba ulang."
			}
			result = append(result, map[string]interface{}{"id": id, "device": device, "name": name, "status": status, "detail": detail, "created": created})
		}
		jsonResp(w, result, 200)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 101<<20)
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		jsonError(w, "upload maksimum 100 MiB", 400)
		return
	}
	defer r.MultipartForm.RemoveAll()
	var targets, args []string
	if json.Unmarshal([]byte(r.FormValue("targets")), &targets) != nil || len(targets) == 0 || len(targets) > 50 || json.Unmarshal([]byte(r.FormValue("args")), &args) != nil || len(args) > 64 {
		jsonError(w, "target/argumen tidak valid", 400)
		return
	}
	seen := map[string]bool{}
	for _, id := range targets {
		dev, err := s.db.GetDevice(id)
		if err != nil || seen[id] || dev.OS != "windows" || !s.hub.IsOnline(id) || versioncmp.IsNewer("0.2.56", dev.Version) {
			jsonError(w, "target harus unik, Windows online, agent >= 0.2.56", 400)
			return
		}
		seen[id] = true
	}
	for _, arg := range args {
		if len(arg) > 2048 || strings.ContainsRune(arg, 0) {
			jsonError(w, "argumen tidak valid", 400)
			return
		}
	}
	file, header, err := r.FormFile("installer")
	if err != nil {
		jsonError(w, "installer wajib", 400)
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".msi" && ext != ".exe" {
		jsonError(w, "hanya MSI/EXE", 400)
		return
	}
	if ext == ".exe" && len(args) == 0 {
		jsonError(w, "EXE wajib argumen silent/no-restart sesuai vendor", 400)
		return
	}
	if ext == ".msi" && len(args) > 0 {
		jsonError(w, "MSI memakai /qn /norestart; argumen tambahan belum didukung", 400)
		return
	}
	dir := filepath.Join(filepath.Dir(s.cfg.DBPath), "deploy-packages")
	if err = os.MkdirAll(dir, 0700); err != nil {
		jsonError(w, "storage unavailable", 500)
		return
	}
	output, err := os.CreateTemp(dir, "package-*")
	if err != nil {
		jsonError(w, "storage unavailable", 500)
		return
	}
	saved := false
	defer func() {
		output.Close()
		if !saved {
			os.Remove(output.Name())
		}
	}()
	digest := sha256.New()
	size, err := io.Copy(io.MultiWriter(output, digest), io.LimitReader(file, (100<<20)+1))
	if err != nil || size == 0 || size > 100<<20 {
		jsonError(w, "ukuran installer tidak valid", 400)
		return
	}
	if err = output.Close(); err != nil {
		jsonError(w, "storage write failed", 500)
		return
	}
	checksum := hex.EncodeToString(digest.Sum(nil))
	encodedArgs, _ := json.Marshal(args)
	tx, err := s.db.db.Begin()
	if err != nil {
		jsonError(w, "database unavailable", 500)
		return
	}
	defer tx.Rollback()
	ids := []string{}
	for _, device := range targets {
		secret := make([]byte, 32)
		if _, err = rand.Read(secret); err != nil {
			jsonError(w, "random unavailable", 500)
			return
		}
		id := hex.EncodeToString(secret)
		ids = append(ids, id)
		_, err = tx.Exec("INSERT INTO deployments(id,device,name,package,digest,args,status,created) VALUES(?,?,?,?,?,?,?,?)", id, device, header.Filename, output.Name(), checksum, string(encodedArgs), "queued", time.Now().Unix())
		if err != nil {
			jsonError(w, "database write failed", 500)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		jsonError(w, "database commit failed", 500)
		return
	}
	saved = true
	for index, device := range targets {
		if err := s.db.RecordAuthLog(claims.Username, r.RemoteAddr, "application_deploy", fmt.Sprintf("device=%s sha256=%s", device, checksum), r.UserAgent()); err != nil {
			_, _ = s.db.db.Exec("UPDATE deployments SET status='failed',detail='audit log unavailable; not dispatched' WHERE id=?", ids[index])
			continue
		}
		payload, _ := json.Marshal(map[string]interface{}{"action": "deploy_application", "data": map[string]interface{}{"id": ids[index], "extension": ext, "digest": checksum, "args": args}})
		if !s.hub.SendToAgent(device, payload) {
			_, _ = s.db.db.Exec("UPDATE deployments SET status='failed',detail='agent disconnected before dispatch' WHERE id=?", ids[index])
		}
	}
	jsonResp(w, map[string]interface{}{"count": len(ids)}, 202)
}

func (s *Server) handleDeploymentPackage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	var path string
	err := s.db.db.QueryRow("SELECT package FROM deployments WHERE id=? AND created>? AND status IN ('queued','running')", id, time.Now().Add(-30*time.Minute).Unix()).Scan(&path)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func (s *Server) handleDeploymentResult(device string, data json.RawMessage) {
	var result struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Detail) > 4096 {
		return
	}
	switch result.Status {
	case "running", "succeeded", "failed", "reboot_required", "unknown":
	default:
		return
	}
	_, _ = s.db.db.Exec("UPDATE deployments SET status=?, detail=? WHERE id=? AND device=? AND status IN ('queued','running')", result.Status, result.Detail, result.ID, device)
}
