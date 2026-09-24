package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

const maxAttachmentSize = 10 << 20

func (d *DB) AddAttachment(a *models.AssetAttachment) error {
	r, err := d.db.Exec(`INSERT INTO asset_attachments(entity_type,entity_id,asset_id,original_name,stored_name,content_type,size,uploaded_by,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		a.EntityType, a.EntityID, a.AssetID, a.OriginalName, a.StoredName, a.ContentType, a.Size, a.UploadedBy, time.Now())
	if err == nil {
		a.ID, err = r.LastInsertId()
	}
	return err
}

func (d *DB) ListAttachments(entityType string, entityID int64) ([]models.AssetAttachment, error) {
	rows, err := d.db.Query(`SELECT id,entity_type,entity_id,asset_id,original_name,stored_name,content_type,size,uploaded_by,created_at FROM asset_attachments WHERE entity_type=? AND entity_id=? ORDER BY created_at`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AssetAttachment
	for rows.Next() {
		var a models.AssetAttachment
		if err := rows.Scan(&a.ID, &a.EntityType, &a.EntityID, &a.AssetID, &a.OriginalName, &a.StoredName, &a.ContentType, &a.Size, &a.UploadedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d *DB) GetAttachment(id int64) (*models.AssetAttachment, error) {
	var a models.AssetAttachment
	err := d.db.QueryRow(`SELECT id,entity_type,entity_id,asset_id,original_name,stored_name,content_type,size,uploaded_by,created_at FROM asset_attachments WHERE id=?`, id).Scan(&a.ID, &a.EntityType, &a.EntityID, &a.AssetID, &a.OriginalName, &a.StoredName, &a.ContentType, &a.Size, &a.UploadedBy, &a.CreatedAt)
	return &a, err
}

func (d *DB) AddActivity(a *models.AssetActivity) error {
	_, err := d.db.Exec(`INSERT INTO asset_activities(category,action,actor,branch,asset_id,asset_type,asset_name,detail,reference_type,reference_id) VALUES(?,?,?,?,?,?,?,?,?,?)`, a.Category, a.Action, a.Actor, a.Branch, a.AssetID, a.AssetType, a.AssetName, a.Detail, a.ReferenceType, a.ReferenceID)
	return err
}

func (d *DB) ListHolderOptions(branch string) ([]models.User, error) {
	branch = strings.TrimSpace(branch)
	rows, err := d.db.Query(`SELECT id,username,role,branch,mfa_enabled,created_at FROM users WHERE role IN ('admin','adh','spv','user','ga_pusat','it_support') ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Branch, &u.MFAEnabled, &u.CreatedAt); err != nil {
			return nil, err
		}
		if eligibleHolderAt(u.Role, u.Branch, branch) {
			out = append(out, u)
		}
	}
	return out, rows.Err()
}

func (d *DB) ListActivities(branch, assetID, category, search, fromDate, toDate string, limit int) ([]models.AssetActivity, error) {
	where := "1=1"
	var args []interface{}
	if branch != "" {
		branches := splitBranches(branch)
		if len(branches) == 1 {
			where += " AND branch=?"
			args = append(args, branches[0])
		} else if len(branches) > 1 {
			var orClauses []string
			for _, b := range branches {
				orClauses = append(orClauses, "branch=?")
				args = append(args, b)
			}
			where += " AND (" + strings.Join(orClauses, " OR ") + ")"
		}
	}
	if assetID != "" {
		where += " AND asset_id=?"
		args = append(args, assetID)
	}
	if category != "" {
		where += " AND category=?"
		args = append(args, category)
	}
	if search != "" {
		where += " AND (asset_name LIKE ? OR actor LIKE ? OR detail LIKE ?)"
		q := "%" + search + "%"
		args = append(args, q, q, q)
	}
	if fromDate != "" {
		where += " AND created_at>=?"
		args = append(args, fromDate+" 00:00:00")
	}
	if toDate != "" {
		if parsed, err := time.Parse("2006-01-02", toDate); err == nil {
			where += " AND created_at<?"
			args = append(args, parsed.AddDate(0, 0, 1).Format("2006-01-02")+" 00:00:00")
		}
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	args = append(args, limit)
	rows, err := d.db.Query(fmt.Sprintf(`SELECT id,category,action,actor,branch,asset_id,asset_type,asset_name,detail,reference_type,reference_id,created_at FROM asset_activities WHERE %s ORDER BY created_at DESC LIMIT ?`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AssetActivity
	for rows.Next() {
		var a models.AssetActivity
		if err := rows.Scan(&a.ID, &a.Category, &a.Action, &a.Actor, &a.Branch, &a.AssetID, &a.AssetType, &a.AssetName, &a.Detail, &a.ReferenceType, &a.ReferenceID, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d *DB) RelocateAsset(assetType, assetID, toBranch, toLocation string) error {
	var exists int
	baseBranch := toBranch
	if idx := strings.Index(toBranch, " - "); idx != -1 {
		baseBranch = strings.TrimSpace(toBranch[:idx])
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM branches WHERE name=? OR name=?`, toBranch, baseBranch).Scan(&exists); err != nil || exists < 1 {
		return fmt.Errorf("lokasi tujuan tidak terdaftar")
	}
	var result interface{ RowsAffected() (int64, error) }
	var err error
	if assetType == "manual" {
		result, err = d.db.Exec(`UPDATE manual_assets SET owner_username=CASE WHEN branch<>? THEN '' ELSE owner_username END,assigned_to=CASE WHEN branch<>? THEN '' ELSE assigned_to END,branch=?,location=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`, toBranch, toBranch, toBranch, toLocation, assetID)
	} else if assetType == "device" {
		result, err = d.db.Exec(`UPDATE devices SET owner_username=CASE WHEN COALESCE(NULLIF(branch,''),group_name)<>? THEN '' ELSE owner_username END,branch=?,group_name=? WHERE id=?`, toBranch, toBranch, toBranch, assetID)
	} else {
		return fmt.Errorf("invalid asset type")
	}
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return fmt.Errorf("asset not found")
	}
	return nil
}

func (s *Server) recordSwitchActivity(sw *models.AssetSwitchRequest, action, actor, detail string) {
	_ = s.db.AddActivity(&models.AssetActivity{Category: "handover", Action: action, Actor: actor, Branch: sw.Branch, AssetID: sw.AssetID, AssetType: sw.AssetType, AssetName: sw.AssetName, Detail: detail, ReferenceType: "handover", ReferenceID: sw.ID})
}

func canAccessSwitch(claims *UserClaims, sw *models.AssetSwitchRequest) bool {
	if isCentralRole(claims.Role) {
		return true
	}
	if claims.Role == "adh" {
		return claims.Branch != "" && userAllowsBranch(claims.Branch, sw.Branch)
	}
	return claims.Username == sw.RequestedBy || claims.Username == sw.FromOwner || claims.Username == sw.ToOwner
}

func (s *Server) handleActivities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	c := getClaims(r)
	branch := r.URL.Query().Get("branch")
	assetID := r.URL.Query().Get("asset_id")
	if isAssetHolderRole(c.Role) {
		if assetID == "" {
			jsonError(w, "asset_id is required", 403)
			return
		}
		allowed := false
		if a, err := s.db.GetManualAsset(assetID); err == nil && a.OwnerUsername == c.Username {
			allowed = true
		}
		if d, err := s.db.GetDevice(assetID); err == nil && d.OwnerUsername == c.Username {
			allowed = true
		}
		if !allowed {
			jsonError(w, "forbidden", 403)
			return
		}
	}
	if c.Role == "adh" && c.Branch != "" {
		if branch != "" && !userAllowsBranch(c.Branch, branch) {
			jsonError(w, "forbidden", 403)
			return
		}
		if branch == "" {
			branch = c.Branch
		}
	} else if isAssetHolderRole(c.Role) {
		branch = c.Branch
	}
	items, err := s.db.ListActivities(branch, assetID, r.URL.Query().Get("category"), r.URL.Query().Get("search"), r.URL.Query().Get("from"), r.URL.Query().Get("to"), 250)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, items, 200)
}

func (s *Server) handleHolderOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	c := getClaims(r)
	branch := strings.TrimSpace(r.URL.Query().Get("branch"))
	if c.Role == "adh" && c.Branch != "" {
		if branch != "" && !userAllowsBranch(c.Branch, branch) {
			jsonError(w, "forbidden: branch not assigned", 403)
			return
		}
		if branch == "" {
			branch = splitBranches(c.Branch)[0]
		}
	} else if isAssetHolderRole(c.Role) && c.Branch != "" {
		branch = c.Branch
	}
	if branch == "" {
		jsonError(w, "branch is required", 400)
		return
	}
	items, err := s.db.ListHolderOptions(branch)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, items, 200)
}

func (s *Server) handleRelocateAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	c := getClaims(r)
	if c.Role == "viewer" || isAssetHolderRole(c.Role) {
		jsonError(w, "forbidden", 403)
		return
	}
	var req struct {
		AssetID    string `json:"asset_id"`
		AssetType  string `json:"asset_type"`
		ToBranch   string `json:"to_branch"`
		ToLocation string `json:"to_location"`
		Reason     string `json:"reason"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.AssetID) == "" || strings.TrimSpace(req.ToBranch) == "" || len([]rune(strings.TrimSpace(req.Reason))) < 10 {
		jsonError(w, "Aset, lokasi tujuan, dan alasan minimal 10 karakter wajib diisi", 400)
		return
	}
	var name, fromBranch, fromLocation, fromOwner string
	if req.AssetType == "manual" {
		a, err := s.db.GetManualAsset(req.AssetID)
		if err != nil {
			jsonError(w, "asset not found", 404)
			return
		}
		name, fromBranch, fromLocation, fromOwner = a.Name, a.Branch, a.Location, a.OwnerUsername
	} else {
		d, err := s.db.GetDevice(req.AssetID)
		if err != nil {
			jsonError(w, "device not found", 404)
			return
		}
		name, fromBranch, fromLocation, fromOwner = d.Hostname, d.Branch, d.LocalIP, d.OwnerUsername
		if fromBranch == "" {
			fromBranch = d.GroupName
		}
	}
	if c.Role == "adh" && (c.Branch == "" || !userAllowsBranch(c.Branch, fromBranch) || !userAllowsBranch(c.Branch, req.ToBranch)) {
		jsonError(w, "ADH hanya dapat mengubah titik lokasi di cabang yang ditugaskan kepadanya; mutasi antar cabang dilakukan GA Pusat/admin", 403)
		return
	}
	if err := s.db.RelocateAsset(req.AssetType, req.AssetID, strings.TrimSpace(req.ToBranch), strings.TrimSpace(req.ToLocation)); err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	detail := fmt.Sprintf("%s / %s → %s / %s. Alasan: %s", fromBranch, fromLocation, strings.TrimSpace(req.ToBranch), strings.TrimSpace(req.ToLocation), strings.TrimSpace(req.Reason))
	if fromBranch != strings.TrimSpace(req.ToBranch) && fromOwner != "" {
		detail += ". PIC lama " + fromOwner + " dilepas; tetapkan PIC awal di lokasi baru."
	}
	_ = s.db.AddActivity(&models.AssetActivity{Category: "location", Action: "relocated", Actor: c.Username, Branch: strings.TrimSpace(req.ToBranch), AssetID: req.AssetID, AssetType: req.AssetType, AssetName: name, Detail: detail})
	jsonResp(w, map[string]string{"status": "relocated"}, 200)
}

func (s *Server) handleAttachment(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/assets/attachments/"), 10, 64)
	if err != nil {
		jsonError(w, "invalid attachment", 400)
		return
	}
	a, err := s.db.GetAttachment(id)
	if err != nil {
		jsonError(w, "attachment not found", 404)
		return
	}
	sw, err := s.db.GetSwitchRequest(a.EntityID)
	if err != nil || !canAccessSwitch(getClaims(r), sw) {
		jsonError(w, "forbidden", 403)
		return
	}
	path := filepath.Join(s.attachmentsDir, a.StoredName)
	f, err := os.Open(path)
	if err != nil {
		jsonError(w, "attachment unavailable", 404)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.OriginalName}))
	io.Copy(w, f)
}

func (s *Server) handleSwitchAttachment(w http.ResponseWriter, r *http.Request, sw *models.AssetSwitchRequest) {
	if !canAccessSwitch(getClaims(r), sw) {
		jsonError(w, "forbidden", 403)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentSize+1024*1024)
	if err := r.ParseMultipartForm(maxAttachmentSize); err != nil {
		jsonError(w, "Lampiran maksimal 10 MB", 400)
		return
	}
	f, h, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file is required", 400)
		return
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	_, _ = f.Seek(0, io.SeekStart)
	contentType := http.DetectContentType(head)
	allowed := map[string]string{"application/pdf": ".pdf", "image/jpeg": ".jpg", "image/png": ".png"}
	ext, ok := allowed[contentType]
	if !ok {
		jsonError(w, "Lampiran hanya PDF, JPG, atau PNG", 400)
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		jsonError(w, "failed to store attachment", 500)
		return
	}
	stored := hex.EncodeToString(b) + ext
	path := filepath.Join(s.attachmentsDir, stored)
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		jsonError(w, "failed to store attachment", 500)
		return
	}
	size, copyErr := io.Copy(out, io.LimitReader(f, maxAttachmentSize+1))
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil || size > maxAttachmentSize {
		_ = os.Remove(path)
		jsonError(w, "Lampiran maksimal 10 MB", 400)
		return
	}
	a := &models.AssetAttachment{EntityType: "handover", EntityID: sw.ID, AssetID: sw.AssetID, OriginalName: filepath.Base(h.Filename), StoredName: stored, ContentType: contentType, Size: size, UploadedBy: getClaims(r).Username}
	if err := s.db.AddAttachment(a); err != nil {
		_ = os.Remove(path)
		jsonError(w, "failed to save attachment", 500)
		return
	}
	s.recordSwitchActivity(sw, "attachment_added", getClaims(r).Username, "Lampiran serah terima: "+a.OriginalName)
	jsonResp(w, a, 201)
}
