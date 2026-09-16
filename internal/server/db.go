package server

import (
	"database/sql"
	"fmt"
	"strconv"
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
		mfa_enabled INTEGER NOT NULL DEFAULT 0,
		mfa_secret TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS trusted_devices (
		token_hash TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		expires_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_trusted_devices_user ON trusted_devices(username);

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

	CREATE TABLE IF NOT EXISTS auth_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		ip TEXT NOT NULL,
		status TEXT NOT NULL,
		reason TEXT NOT NULL DEFAULT '',
		user_agent TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_auth_logs_created ON auth_logs(created_at);
	CREATE INDEX IF NOT EXISTS idx_auth_logs_status ON auth_logs(status);

	CREATE TABLE IF NOT EXISTS system_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS branches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL DEFAULT 'cabang',
		business_unit TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_branches_name ON branches(name);
	CREATE TABLE IF NOT EXISTS business_units (name TEXT PRIMARY KEY);
	CREATE TABLE IF NOT EXISTS branch_business_units (branch_id INTEGER NOT NULL, business_unit TEXT NOT NULL, PRIMARY KEY(branch_id, business_unit), FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE);
	CREATE TABLE IF NOT EXISTS asset_deletion_logs (id INTEGER PRIMARY KEY AUTOINCREMENT, asset_id TEXT NOT NULL, asset_type TEXT NOT NULL, asset_name TEXT NOT NULL, branch TEXT NOT NULL, deleted_by TEXT NOT NULL, reason TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	_, _ = db.Exec(`INSERT OR IGNORE INTO branches (name, type) VALUES ('Pusat', 'pusat')`)
	// Role Kacab has been replaced by ADH; preserve every existing account and scope.
	_, _ = db.Exec(`UPDATE users SET role='adh' WHERE role='kacab'`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO business_units(name) SELECT business_unit FROM branches WHERE TRIM(business_unit) != ''`)
	_, _ = db.Exec(`INSERT OR IGNORE INTO branch_business_units(branch_id, business_unit) SELECT id, business_unit FROM branches WHERE TRIM(business_unit) != ''`)

	// Dynamic column migrations for existing databases
	alters := []string{
		`ALTER TABLE users ADD COLUMN branch TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN mfa_enabled INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN mfa_secret TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE branches ADD COLUMN business_unit TEXT NOT NULL DEFAULT ''`,
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
	rows, err := d.db.Query(`SELECT id, username, role, branch, COALESCE(mfa_enabled, 0), created_at FROM users ORDER BY username ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []models.User
	for rows.Next() {
		var u models.User
		var mfaInt int
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.Branch, &mfaInt, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.MFAEnabled = mfaInt == 1
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
	var role string
	err := d.db.QueryRow(`SELECT role FROM users WHERE id=?`, id).Scan(&role)
	if err != nil {
		return err
	}
	if role == "admin" {
		var adminCount int
		_ = d.db.QueryRow(`SELECT COUNT(*) FROM users WHERE role='admin'`).Scan(&adminCount)
		if adminCount <= 1 {
			return fmt.Errorf("tidak dapat menghapus administrator terakhir")
		}
	}
	_, err = d.db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

func (d *DB) UpdateUsername(oldUsername, newUsername string) error {
	var count int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM users WHERE username=?`, newUsername).Scan(&count)
	if count > 0 {
		return fmt.Errorf("username '%s' sudah digunakan", newUsername)
	}
	_, err := d.db.Exec(`UPDATE users SET username=? WHERE username=?`, newUsername, oldUsername)
	return err
}

func (d *DB) UpdatePassword(username, passwordHash string) error {
	_, err := d.db.Exec(`UPDATE users SET password_hash=? WHERE username=?`, passwordHash, username)
	return err
}

func (d *DB) TrustDevice(username, tokenHash string, expiresAt time.Time) error {
	_, err := d.db.Exec(`INSERT INTO trusted_devices (token_hash, username, expires_at) VALUES (?, ?, ?)`, tokenHash, username, expiresAt)
	return err
}

func (d *DB) IsTrustedDevice(username, tokenHash string) bool {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM trusted_devices WHERE username=? AND token_hash=? AND expires_at>?`, username, tokenHash, time.Now()).Scan(&count)
	return err == nil && count == 1
}

func (d *DB) RevokeTrustedDevices(username string) {
	_, _ = d.db.Exec(`DELETE FROM trusted_devices WHERE username=?`, username)
}

func (d *DB) ListBusinessUnits() ([]string, error) {
	rows, err := d.db.Query(`SELECT name FROM business_units ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		result = append(result, name)
	}
	return result, rows.Err()
}

func (d *DB) AddBusinessUnit(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("business unit name is required")
	}
	_, err := d.db.Exec(`INSERT INTO business_units(name) VALUES (?)`, name)
	return err
}

func (d *DB) SetBranchBusinessUnits(id int64, units []string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM branch_business_units WHERE branch_id=?`, id); err != nil {
		return err
	}
	for _, unit := range units {
		unit = strings.TrimSpace(unit)
		if unit == "" {
			continue
		}
		if _, err = tx.Exec(`INSERT OR IGNORE INTO business_units(name) VALUES (?)`, unit); err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO branch_business_units(branch_id, business_unit) VALUES (?, ?)`, id, unit); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) SetBranchBusinessUnitsByName(name string, units []string) error {
	var id int64
	if err := d.db.QueryRow(`SELECT id FROM branches WHERE name=?`, name).Scan(&id); err != nil {
		return err
	}
	return d.SetBranchBusinessUnits(id, units)
}

func (d *DB) GetUserMFA(username string) (bool, string, error) {
	var enabled int
	var secret string
	err := d.db.QueryRow(`SELECT COALESCE(mfa_enabled, 0), COALESCE(mfa_secret, '') FROM users WHERE username=?`, username).
		Scan(&enabled, &secret)
	return enabled == 1, secret, err
}

func (d *DB) SetUserMFA(username, secret string, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	_, err := d.db.Exec(`UPDATE users SET mfa_enabled=?, mfa_secret=? WHERE username=?`, en, secret, username)
	return err
}

func (d *DB) ResetUserMFA(id int64) error {
	_, err := d.db.Exec(`UPDATE users SET mfa_enabled=0, mfa_secret='' WHERE id=?`, id)
	return err
}

func (d *DB) ResetUserPassword(id int64, passwordHash string) error {
	_, err := d.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, passwordHash, id)
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

func (d *DB) RecordAssetDeletion(id, assetType, name, branch, deletedBy, reason string) error {
	_, err := d.db.Exec(`INSERT INTO asset_deletion_logs (asset_id, asset_type, asset_name, branch, deleted_by, reason) VALUES (?, ?, ?, ?, ?, ?)`, id, assetType, name, branch, deletedBy, reason)
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
		"branch":        branch,
		"total_assets":  totalAll,
		"manual_assets": mTotal,
		"device_assets": dTotal,
		"verified":      verifiedAll,
		"unverified":    unverifiedAll,
		"discrepancy":   discrepancyAll,
	}, nil
}

func (d *DB) GetBranches() ([]string, error) {
	q := `
	SELECT DISTINCT branch FROM (
		SELECT name AS branch FROM branches WHERE name != ''
		UNION
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

func (d *DB) RecordAuthLog(username, ip, status, reason, userAgent string) error {
	_, err := d.db.Exec(`INSERT INTO auth_logs (username, ip, status, reason, user_agent) VALUES (?, ?, ?, ?, ?)`,
		username, ip, status, reason, userAgent)
	return err
}

func (d *DB) GetAuthLogs(limit int) ([]models.AuthLog, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := d.db.Query(`SELECT id, username, ip, status, reason, user_agent, created_at FROM auth_logs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuthLog
	for rows.Next() {
		var l models.AuthLog
		if err := rows.Scan(&l.ID, &l.Username, &l.IP, &l.Status, &l.Reason, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (d *DB) GetSecuritySettings() models.SecuritySettings {
	s := models.SecuritySettings{
		RateLimitEnabled: true,
		MaxLoginAttempts: 5,
		BlockDurationMin: 15,
		IPWhitelist:      "127.0.0.1, ::1",
	}
	var val string
	if err := d.db.QueryRow(`SELECT value FROM system_settings WHERE key='rate_limit_enabled'`).Scan(&val); err == nil {
		s.RateLimitEnabled = val == "true" || val == "1"
	}
	if err := d.db.QueryRow(`SELECT value FROM system_settings WHERE key='max_login_attempts'`).Scan(&val); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			s.MaxLoginAttempts = n
		}
	}
	if err := d.db.QueryRow(`SELECT value FROM system_settings WHERE key='block_duration_minutes'`).Scan(&val); err == nil {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			s.BlockDurationMin = n
		}
	}
	if err := d.db.QueryRow(`SELECT value FROM system_settings WHERE key='ip_whitelist'`).Scan(&val); err == nil {
		s.IPWhitelist = val
	}
	return s
}

func (d *DB) SaveSecuritySettings(s models.SecuritySettings) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	enStr := "false"
	if s.RateLimitEnabled {
		enStr = "true"
	}
	pairs := map[string]string{
		"rate_limit_enabled":     enStr,
		"max_login_attempts":     strconv.Itoa(s.MaxLoginAttempts),
		"block_duration_minutes": strconv.Itoa(s.BlockDurationMin),
		"ip_whitelist":           s.IPWhitelist,
	}

	for k, v := range pairs {
		_, err := tx.Exec(`INSERT INTO system_settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=CURRENT_TIMESTAMP`, k, v)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) ListBranches() ([]models.Branch, error) {
	rows, err := d.db.Query(`SELECT b.id, b.name, b.type, COALESCE(GROUP_CONCAT(bbu.business_unit, '|'), ''), b.created_at FROM branches b LEFT JOIN branch_business_units bbu ON bbu.branch_id=b.id GROUP BY b.id ORDER BY b.type ASC, b.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Branch
	for rows.Next() {
		var b models.Branch
		var units string
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &units, &b.CreatedAt); err != nil {
			return nil, err
		}
		if units != "" {
			b.BusinessUnits = strings.Split(units, "|")
		}
		list = append(list, b)
	}
	return list, nil
}

func (d *DB) GetBranch(id int64) (*models.Branch, error) {
	row := d.db.QueryRow(`SELECT id, name, type, created_at FROM branches WHERE id=?`, id)
	var b models.Branch
	if err := row.Scan(&b.ID, &b.Name, &b.Type, &b.CreatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func (d *DB) CreateBranch(name, branchType, businessUnit string) error {
	name = strings.TrimSpace(name)
	branchType = strings.TrimSpace(branchType)
	if name == "" {
		return fmt.Errorf("branch name is required")
	}
	if !validBranchType(branchType) {
		return fmt.Errorf("invalid branch type")
	}
	_, err := d.db.Exec(`INSERT INTO branches (name, type, business_unit) VALUES (?, ?, ?)`, name, branchType, strings.TrimSpace(businessUnit))
	return err
}

func (d *DB) UpdateBranch(id int64, name, branchType, businessUnit string) error {
	name = strings.TrimSpace(name)
	branchType = strings.TrimSpace(branchType)
	if name == "" {
		return fmt.Errorf("branch name is required")
	}
	if !validBranchType(branchType) {
		return fmt.Errorf("invalid branch type")
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oldName string
	if err := tx.QueryRow(`SELECT name FROM branches WHERE id=?`, id).Scan(&oldName); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE branches SET name=?, type=?, business_unit=? WHERE id=?`, name, branchType, strings.TrimSpace(businessUnit), id); err != nil {
		return err
	}
	if oldName != name {
		if _, err := tx.Exec(`UPDATE manual_assets SET branch=? WHERE branch=?`, name, oldName); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE devices SET branch=?, group_name=CASE WHEN group_name=? THEN ? ELSE group_name END WHERE branch=? OR group_name=?`, name, oldName, name, oldName, oldName); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE users SET branch=? WHERE branch=?`, name, oldName); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE asset_verifications SET branch=? WHERE branch=?`, name, oldName); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *DB) DeleteBranch(id int64) error {
	var name string
	if err := d.db.QueryRow(`SELECT name FROM branches WHERE id=?`, id).Scan(&name); err != nil {
		return err
	}
	if name == "Pusat" {
		return fmt.Errorf("Pusat is a required location and cannot be deleted")
	}
	var used int
	err := d.db.QueryRow(`SELECT ((SELECT COUNT(*) FROM manual_assets WHERE branch=?) + (SELECT COUNT(*) FROM devices WHERE branch=? OR group_name=?) + (SELECT COUNT(*) FROM users WHERE branch=?))`, name, name, name, name).Scan(&used)
	if err != nil {
		return err
	}
	if used > 0 {
		return fmt.Errorf("location is still used by %d record(s); rename it or reassign its records first", used)
	}
	_, err = d.db.Exec(`DELETE FROM branches WHERE id=?`, id)
	return err
}

func validBranchType(branchType string) bool {
	return len(strings.TrimSpace(branchType)) > 0 && len(strings.TrimSpace(branchType)) <= 60
}
