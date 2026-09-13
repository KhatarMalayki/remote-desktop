package server

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/user/remote-desktop/internal/models"
)

type DB struct {
	db *sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS devices (
		id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL DEFAULT '',
		os TEXT NOT NULL DEFAULT '',
		arch TEXT NOT NULL DEFAULT '',
		ip TEXT NOT NULL DEFAULT '',
		local_ip TEXT NOT NULL DEFAULT '',
		cpu_model TEXT NOT NULL DEFAULT '',
		cpu_cores INTEGER NOT NULL DEFAULT 0,
		memory_total INTEGER NOT NULL DEFAULT 0,
		memory_used INTEGER NOT NULL DEFAULT 0,
		disk_total INTEGER NOT NULL DEFAULT 0,
		disk_used INTEGER NOT NULL DEFAULT 0,
		version TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'active',
		tags TEXT NOT NULL DEFAULT '',
		group_name TEXT NOT NULL DEFAULT 'default',
		branch TEXT NOT NULL DEFAULT '',
		verification_status TEXT NOT NULL DEFAULT 'unverified',
		verified_at DATETIME,
		verified_by TEXT NOT NULL DEFAULT '',
		verification_note TEXT NOT NULL DEFAULT '',
		note TEXT NOT NULL DEFAULT '',
		last_seen DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		registered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_devices_group ON devices(group_name);
	CREATE INDEX IF NOT EXISTS idx_devices_branch ON devices(branch);
	CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
	CREATE INDEX IF NOT EXISTS idx_devices_verification ON devices(verification_status);
	CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);

	CREATE TABLE IF NOT EXISTS device_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id TEXT NOT NULL,
		action TEXT NOT NULL,
		detail TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(device_id) REFERENCES devices(id)
	);
	CREATE INDEX IF NOT EXISTS idx_device_logs_device ON device_logs(device_id);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		branch TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS manual_assets (
		id TEXT PRIMARY KEY,
		asset_tag TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'other',
		branch TEXT NOT NULL,
		location TEXT NOT NULL DEFAULT '',
		assigned_to TEXT NOT NULL DEFAULT '',
		serial_number TEXT NOT NULL DEFAULT '',
		specs TEXT NOT NULL DEFAULT '',
		condition TEXT NOT NULL DEFAULT 'good',
		status TEXT NOT NULL DEFAULT 'active',
		verification_status TEXT NOT NULL DEFAULT 'unverified',
		verified_at DATETIME,
		verified_by TEXT NOT NULL DEFAULT '',
		verification_note TEXT NOT NULL DEFAULT '',
		created_by TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_manual_assets_branch ON manual_assets(branch);
	CREATE INDEX IF NOT EXISTS idx_manual_assets_verification ON manual_assets(verification_status);

	CREATE TABLE IF NOT EXISTS asset_verifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		asset_id TEXT NOT NULL,
		asset_type TEXT NOT NULL,
		asset_name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		condition TEXT NOT NULL DEFAULT '',
		verified_by TEXT NOT NULL,
		branch TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_asset_verifications_asset ON asset_verifications(asset_id);
	CREATE INDEX IF NOT EXISTS idx_asset_verifications_branch ON asset_verifications(branch);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Dynamic column migrations for existing databases
	alters := []string{
		`ALTER TABLE users ADD COLUMN branch TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE devices ADD COLUMN branch TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE devices ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'unverified'`,
		`ALTER TABLE devices ADD COLUMN verified_at DATETIME`,
		`ALTER TABLE devices ADD COLUMN verified_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE devices ADD COLUMN verification_note TEXT NOT NULL DEFAULT ''`,
	}
	for _, q := range alters {
		_, _ = db.Exec(q)
	}

	return nil
}

func (d *DB) UpsertDevice(dev *models.Device) error {
	branch := dev.Branch
	if branch == "" {
		branch = dev.GroupName
	}
	query := `
	INSERT INTO devices (id, hostname, os, arch, ip, local_ip, cpu_model, cpu_cores,
		memory_total, memory_used, disk_total, disk_used, version, status, tags, group_name, branch, note, last_seen, registered_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		hostname=excluded.hostname, os=excluded.os, arch=excluded.arch,
		ip=excluded.ip, local_ip=excluded.local_ip, cpu_model=excluded.cpu_model,
		cpu_cores=excluded.cpu_cores, memory_total=excluded.memory_total,
		memory_used=excluded.memory_used, disk_total=excluded.disk_total,
		disk_used=excluded.disk_used, version=excluded.version,
		last_seen=excluded.last_seen
	`
	_, err := d.db.Exec(query,
		dev.ID, dev.Hostname, dev.OS, dev.Arch, dev.IP, dev.LocalIP,
		dev.CPUModel, dev.CPUCores, dev.MemoryTotal, dev.MemoryUsed,
		dev.DiskTotal, dev.DiskUsed, dev.Version, dev.Status,
		dev.Tags, dev.GroupName, branch, dev.Note, dev.LastSeen, dev.RegisteredAt,
	)
	return err
}

func (d *DB) UpdateHeartbeat(hb *models.DeviceHeartbeat) error {
	_, err := d.db.Exec(
		`UPDATE devices SET memory_used=?, disk_used=?, last_seen=? WHERE id=?`,
		hb.MemoryUsed, hb.DiskUsed, time.Now(), hb.ID,
	)
	return err
}

func (d *DB) GetDevice(id string) (*models.Device, error) {
	row := d.db.QueryRow(`SELECT id, hostname, os, arch, ip, local_ip, cpu_model, cpu_cores,
		memory_total, memory_used, disk_total, disk_used, version, status, tags, group_name,
		branch, verification_status, verified_at, verified_by, verification_note, note,
		last_seen, registered_at FROM devices WHERE id=?`, id)
	return scanDevice(row)
}

func (d *DB) ListDevices(group, search string, limit, offset int) ([]*models.Device, int, error) {
	where := "1=1"
	args := []interface{}{}
	if group != "" {
		where += " AND (group_name=? OR branch=?)"
		args = append(args, group, group)
	}
	if search != "" {
		where += " AND (hostname LIKE ? OR ip LIKE ? OR id LIKE ? OR branch LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s, s)
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := d.db.QueryRow("SELECT COUNT(*) FROM devices WHERE "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT id, hostname, os, arch, ip, local_ip, cpu_model, cpu_cores,
		memory_total, memory_used, disk_total, disk_used, version, status, tags, group_name,
		branch, verification_status, verified_at, verified_by, verification_note, note,
		last_seen, registered_at FROM devices WHERE %s ORDER BY last_seen DESC LIMIT ? OFFSET ?`, where)
	args = append(args, limit, offset)
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		dev, err := scanDeviceRows(rows)
		if err != nil {
			return nil, 0, err
		}
		devices = append(devices, dev)
	}
	return devices, total, nil
}

func (d *DB) UpdateDeviceMeta(id, tags, group, note string) error {
	_, err := d.db.Exec(`UPDATE devices SET tags=?, group_name=?, branch=?, note=? WHERE id=?`,
		tags, group, group, note, id)
	return err
}

func (d *DB) DeleteDevice(id string) error {
	_, err := d.db.Exec(`DELETE FROM devices WHERE id=?`, id)
	return err
}

func (d *DB) GetGroups() ([]string, error) {
	rows, err := d.db.Query(`SELECT DISTINCT group_name FROM devices WHERE group_name != '' UNION SELECT DISTINCT branch FROM devices WHERE branch != '' ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		if g != "" {
			groups = append(groups, g)
		}
	}
	return groups, nil
}

func (d *DB) AddLog(deviceID, action, detail string) error {
	_, err := d.db.Exec(`INSERT INTO device_logs (device_id, action, detail) VALUES (?, ?, ?)`,
		deviceID, action, detail)
	return err
}

func (d *DB) GetLogs(deviceID string, limit int) ([]map[string]interface{}, error) {
	rows, err := d.db.Query(
		`SELECT id, device_id, action, detail, created_at FROM device_logs WHERE device_id=? ORDER BY created_at DESC LIMIT ?`,
		deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var devID, action, detail string
		var createdAt time.Time
		if err := rows.Scan(&id, &devID, &action, &detail, &createdAt); err != nil {
			return nil, err
		}
		logs = append(logs, map[string]interface{}{
			"id":         id,
			"device_id":  devID,
			"action":     action,
			"detail":     detail,
			"created_at": createdAt,
		})
	}
	return logs, nil
}

func (d *DB) Stats() (map[string]interface{}, error) {
	var total, online int
	threshold := time.Now().Add(-2 * time.Minute)
	d.db.QueryRow(`SELECT COUNT(*) FROM devices`).Scan(&total)
	d.db.QueryRow(`SELECT COUNT(*) FROM devices WHERE last_seen > ?`, threshold).Scan(&online)

	osRows, _ := d.db.Query(`SELECT os, COUNT(*) as cnt FROM devices GROUP BY os ORDER BY cnt DESC`)
	osDist := map[string]int{}
	if osRows != nil {
		defer osRows.Close()
		for osRows.Next() {
			var osName string
			var cnt int
			osRows.Scan(&osName, &cnt)
			osDist[osName] = cnt
		}
	}

	return map[string]interface{}{
		"total_devices":   total,
		"online_devices":  online,
		"offline_devices": total - online,
		"os_distribution": osDist,
	}, nil
}

// ---------------- USER MANAGEMENT ----------------

func (d *DB) EnsureAdmin(username, passwordHash string) error {
	_, err := d.db.Exec(
		`INSERT OR IGNORE INTO users (username, password_hash, role, branch) VALUES (?, ?, 'admin', '')`,
		username, passwordHash)
	return err
}

func (d *DB) GetUser(username string) (int, string, string, string, error) {
	var id int
	var hash, role, branch string
	err := d.db.QueryRow(`SELECT id, password_hash, role, branch FROM users WHERE username=?`, username).
		Scan(&id, &hash, &role, &branch)
	return id, hash, role, branch, err
}

func (d *DB) ListUsers() ([]models.User, error) {
	rows, err := d.db.Query(`SELECT id, username, role, branch, created_at FROM users ORDER BY username ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Branch, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (d *DB) CreateUser(username, passwordHash, role, branch string) error {
	_, err := d.db.Exec(
		`INSERT INTO users (username, password_hash, role, branch) VALUES (?, ?, ?, ?)`,
		username, passwordHash, role, branch)
	return err
}

func (d *DB) DeleteUser(id int64) error {
	_, err := d.db.Exec(`DELETE FROM users WHERE id=? AND role != 'admin'`, id)
	return err
}

// ---------------- MANUAL ASSETS ----------------

func (d *DB) CreateManualAsset(a *models.ManualAsset) error {
	if a.ID == "" {
		a.ID = fmt.Sprintf("ast-%d", time.Now().UnixNano())
	}
	if a.VerificationStatus == "" {
		a.VerificationStatus = "unverified"
	}
	if a.Status == "" {
		a.Status = "active"
	}
	if a.Condition == "" {
		a.Condition = "good"
	}
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now

	query := `INSERT INTO manual_assets (
		id, asset_tag, name, category, branch, location, assigned_to, serial_number,
		specs, condition, status, verification_status, verified_at, verified_by,
		verification_note, created_by, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query,
		a.ID, a.AssetTag, a.Name, a.Category, a.Branch, a.Location, a.AssignedTo, a.SerialNumber,
		a.Specs, a.Condition, a.Status, a.VerificationStatus, a.VerifiedAt, a.VerifiedBy,
		a.VerificationNote, a.CreatedBy, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (d *DB) UpdateManualAsset(a *models.ManualAsset) error {
	a.UpdatedAt = time.Now()
	query := `UPDATE manual_assets SET
		asset_tag=?, name=?, category=?, branch=?, location=?, assigned_to=?,
		serial_number=?, specs=?, condition=?, status=?, updated_at=?
		WHERE id=?`
	_, err := d.db.Exec(query,
		a.AssetTag, a.Name, a.Category, a.Branch, a.Location, a.AssignedTo,
		a.SerialNumber, a.Specs, a.Condition, a.Status, a.UpdatedAt, a.ID)
	return err
}

func (d *DB) DeleteManualAsset(id string) error {
	_, err := d.db.Exec(`DELETE FROM manual_assets WHERE id=?`, id)
	return err
}

func (d *DB) GetManualAsset(id string) (*models.ManualAsset, error) {
	row := d.db.QueryRow(`SELECT id, asset_tag, name, category, branch, location, assigned_to,
		serial_number, specs, condition, status, verification_status, verified_at, verified_by,
		verification_note, created_by, created_at, updated_at
		FROM manual_assets WHERE id=?`, id)
	return scanManualAsset(row)
}

func (d *DB) ListManualAssets(branch, category, verificationStatus, search string) ([]models.ManualAsset, error) {
	where := "1=1"
	var args []interface{}
	if branch != "" {
		where += " AND branch=?"
		args = append(args, branch)
	}
	if category != "" {
		where += " AND category=?"
		args = append(args, category)
	}
	if verificationStatus != "" {
		where += " AND verification_status=?"
		args = append(args, verificationStatus)
	}
	if search != "" {
		where += " AND (name LIKE ? OR asset_tag LIKE ? OR location LIKE ? OR assigned_to LIKE ? OR serial_number LIKE ?)"
		s := "%" + search + "%"
		args = append(args, s, s, s, s, s)
	}

	query := fmt.Sprintf(`SELECT id, asset_tag, name, category, branch, location, assigned_to,
		serial_number, specs, condition, status, verification_status, verified_at, verified_by,
		verification_note, created_by, created_at, updated_at
		FROM manual_assets WHERE %s ORDER BY updated_at DESC`, where)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.ManualAsset
	for rows.Next() {
		a, err := scanManualAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *a)
	}
	return assets, nil
}

func (d *DB) VerifyManualAsset(id, status, condition, verifiedBy, notes string) error {
	now := time.Now()
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var name, branch string
	err = tx.QueryRow(`SELECT name, branch FROM manual_assets WHERE id=?`, id).Scan(&name, &branch)
	if err != nil {
		return err
	}

	upd := `UPDATE manual_assets SET verification_status=?, condition=COALESCE(NULLIF(?,''), condition),
		verified_at=?, verified_by=?, verification_note=?, updated_at=? WHERE id=?`
	if _, err := tx.Exec(upd, status, condition, now, verifiedBy, notes, now, id); err != nil {
		return err
	}

	ins := `INSERT INTO asset_verifications (asset_id, asset_type, asset_name, status, condition, verified_by, branch, notes, created_at)
		VALUES (?, 'manual', ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.Exec(ins, id, name, status, condition, verifiedBy, branch, notes, now); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) VerifyDevice(id, status, condition, verifiedBy, notes string) error {
	now := time.Now()
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var hostname, branch string
	err = tx.QueryRow(`SELECT hostname, COALESCE(NULLIF(branch,''), group_name) FROM devices WHERE id=?`, id).Scan(&hostname, &branch)
	if err != nil {
		return err
	}

	upd := `UPDATE devices SET verification_status=?, verified_at=?, verified_by=?, verification_note=? WHERE id=?`
	if _, err := tx.Exec(upd, status, now, verifiedBy, notes, id); err != nil {
		return err
	}

	ins := `INSERT INTO asset_verifications (asset_id, asset_type, asset_name, status, condition, verified_by, branch, notes, created_at)
		VALUES (?, 'device', ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.Exec(ins, id, hostname, status, condition, verifiedBy, branch, notes, now); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *DB) GetAssetVerifications(assetID, branch string, limit int) ([]models.AssetVerification, error) {
	where := "1=1"
	var args []interface{}
	if assetID != "" {
		where += " AND asset_id=?"
		args = append(args, assetID)
	}
	if branch != "" {
		where += " AND branch=?"
		args = append(args, branch)
	}
	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf(`SELECT id, asset_id, asset_type, asset_name, status, condition, verified_by, branch, notes, created_at
		FROM asset_verifications WHERE %s ORDER BY created_at DESC LIMIT ?`, where)
	args = append(args, limit)

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vers []models.AssetVerification
	for rows.Next() {
		var v models.AssetVerification
		if err := rows.Scan(&v.ID, &v.AssetID, &v.AssetType, &v.AssetName, &v.Status, &v.Condition, &v.VerifiedBy, &v.Branch, &v.Notes, &v.CreatedAt); err != nil {
			return nil, err
		}
		vers = append(vers, v)
	}
	return vers, nil
}

func (d *DB) GetBranchStats(branch string) (map[string]interface{}, error) {
	whereManual := "1=1"
	whereDevice := "1=1"
	var argsM, argsD []interface{}
	if branch != "" {
		whereManual += " AND branch=?"
		argsM = append(argsM, branch)
		whereDevice += " AND (branch=? OR group_name=?)"
		argsD = append(argsD, branch, branch)
	}

	var mTotal, mVerified, mDiscrepancy int
	_ = d.db.QueryRow("SELECT COUNT(*) FROM manual_assets WHERE "+whereManual, argsM...).Scan(&mTotal)
	_ = d.db.QueryRow("SELECT COUNT(*) FROM manual_assets WHERE verification_status='verified' AND "+whereManual, argsM...).Scan(&mVerified)
	_ = d.db.QueryRow("SELECT COUNT(*) FROM manual_assets WHERE verification_status='discrepancy' AND "+whereManual, argsM...).Scan(&mDiscrepancy)

	var dTotal, dVerified, dDiscrepancy int
	_ = d.db.QueryRow("SELECT COUNT(*) FROM devices WHERE "+whereDevice, argsD...).Scan(&dTotal)
	_ = d.db.QueryRow("SELECT COUNT(*) FROM devices WHERE verification_status='verified' AND "+whereDevice, argsD...).Scan(&dVerified)
	_ = d.db.QueryRow("SELECT COUNT(*) FROM devices WHERE verification_status='discrepancy' AND "+whereDevice, argsD...).Scan(&dDiscrepancy)

	totalAll := mTotal + dTotal
	verifiedAll := mVerified + dVerified
	discrepancyAll := mDiscrepancy + dDiscrepancy
	unverifiedAll := totalAll - (verifiedAll + discrepancyAll)

	return map[string]interface{}{
		"branch":              branch,
		"total_assets":        totalAll,
		"manual_assets":       mTotal,
		"device_assets":       dTotal,
		"verified":            verifiedAll,
		"unverified":          unverifiedAll,
		"discrepancy":         discrepancyAll,
	}, nil
}

func (d *DB) GetBranches() ([]string, error) {
	q := `
	SELECT DISTINCT branch FROM (
		SELECT branch FROM manual_assets WHERE branch != ''
		UNION
		SELECT branch FROM devices WHERE branch != ''
		UNION
		SELECT group_name AS branch FROM devices WHERE group_name != '' AND group_name != 'default'
		UNION
		SELECT branch FROM users WHERE branch != ''
	) ORDER BY 1`
	rows, err := d.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []string
	for rows.Next() {
		var b string
		if err := rows.Scan(&b); err == nil && strings.TrimSpace(b) != "" {
			branches = append(branches, strings.TrimSpace(b))
		}
	}
	return branches, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanDevice(row scanner) (*models.Device, error) {
	dev := &models.Device{}
	var vAt sql.NullTime
	err := row.Scan(&dev.ID, &dev.Hostname, &dev.OS, &dev.Arch, &dev.IP, &dev.LocalIP,
		&dev.CPUModel, &dev.CPUCores, &dev.MemoryTotal, &dev.MemoryUsed,
		&dev.DiskTotal, &dev.DiskUsed, &dev.Version, &dev.Status,
		&dev.Tags, &dev.GroupName, &dev.Branch, &dev.VerificationStatus, &vAt,
		&dev.VerifiedBy, &dev.VerificationNote, &dev.Note, &dev.LastSeen, &dev.RegisteredAt)
	if err != nil {
		return nil, err
	}
	if vAt.Valid {
		dev.VerifiedAt = &vAt.Time
	}
	if dev.Branch == "" {
		dev.Branch = dev.GroupName
	}
	if dev.VerificationStatus == "" {
		dev.VerificationStatus = "unverified"
	}
	return dev, nil
}

func scanDeviceRows(rows *sql.Rows) (*models.Device, error) {
	return scanDevice(rows)
}

func scanManualAsset(row scanner) (*models.ManualAsset, error) {
	a := &models.ManualAsset{}
	var vAt sql.NullTime
	err := row.Scan(&a.ID, &a.AssetTag, &a.Name, &a.Category, &a.Branch, &a.Location,
		&a.AssignedTo, &a.SerialNumber, &a.Specs, &a.Condition, &a.Status,
		&a.VerificationStatus, &vAt, &a.VerifiedBy, &a.VerificationNote,
		&a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if vAt.Valid {
		a.VerifiedAt = &vAt.Time
	}
	if a.VerificationStatus == "" {
		a.VerificationStatus = "unverified"
	}
	return a, nil
}
