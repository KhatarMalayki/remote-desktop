package models

import (
	"encoding/json"
	"time"
)

type Device struct {
	ID                 string     `json:"id"`
	Hostname           string     `json:"hostname"`
	OS                 string     `json:"os"`
	Arch               string     `json:"arch"`
	IP                 string     `json:"ip"`
	LocalIP            string     `json:"local_ip"`
	CPUModel           string     `json:"cpu_model"`
	CPUCores           int        `json:"cpu_cores"`
	MemoryTotal        uint64     `json:"memory_total"`
	MemoryUsed         uint64     `json:"memory_used"`
	DiskTotal          uint64     `json:"disk_total"`
	DiskUsed           uint64     `json:"disk_used"`
	Version            string     `json:"version"`
	Status             string     `json:"status"`
	Tags               string     `json:"tags"`
	GroupName          string     `json:"group"`
	Branch             string     `json:"branch"`
	VerificationStatus string     `json:"verification_status"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	VerifiedBy         string     `json:"verified_by"`
	VerificationNote   string     `json:"verification_note"`
	Note               string     `json:"note"`
	LastSeen           time.Time  `json:"last_seen"`
	RegisteredAt       time.Time  `json:"registered_at"`
	Online             bool       `json:"online"`
}

type ManualAsset struct {
	ID                 string     `json:"id"`
	AssetTag           string     `json:"asset_tag"`
	Name               string     `json:"name"`
	Category           string     `json:"category"` // pc, laptop, printer, pos, network, other
	Branch             string     `json:"branch"`
	Location           string     `json:"location"`
	AssignedTo         string     `json:"assigned_to"`
	SerialNumber       string     `json:"serial_number"`
	Specs              string     `json:"specs"`
	Condition          string     `json:"condition"`           // good, fair, damaged
	Status             string     `json:"status"`              // active, maintenance, disposed
	VerificationStatus string     `json:"verification_status"` // unverified, verified, discrepancy
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	VerifiedBy         string     `json:"verified_by"`
	VerificationNote   string     `json:"verification_note"`
	CreatedBy          string     `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type AssetVerification struct {
	ID        int64     `json:"id"`
	AssetID   string    `json:"asset_id"`
	AssetType string    `json:"asset_type"` // manual, device
	AssetName string    `json:"asset_name"`
	Status    string    `json:"status"` // verified, discrepancy, unverified
	Condition string    `json:"condition"`
	VerifiedBy string   `json:"verified_by"`
	Branch    string    `json:"branch"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID         int64     `json:"id"`
	Username   string    `json:"username"`
	Role       string    `json:"role"` // admin, kacab, viewer
	Branch     string    `json:"branch"`
	MFAEnabled bool      `json:"mfa_enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuthLog struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	Status    string    `json:"status"` // success, failed, blocked, mfa_failed
	Reason    string    `json:"reason"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

type Branch struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"` // pusat, cabang, bisnis_unit, service_point, pool, site
	CreatedAt time.Time `json:"created_at"`
}

type SecuritySettings struct {
	RateLimitEnabled bool   `json:"rate_limit_enabled"`
	MaxLoginAttempts int    `json:"max_login_attempts"`
	BlockDurationMin int    `json:"block_duration_minutes"`
	IPWhitelist      string `json:"ip_whitelist"`
}

type BlockedIPInfo struct {
	IP          string    `json:"ip"`
	FailedCount int       `json:"failed_count"`
	BlockedAt   time.Time `json:"blocked_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	MinutesLeft int       `json:"minutes_left"`
}

type DeviceHeartbeat struct {
	ID         string  `json:"id"`
	CPUUsage   float64 `json:"cpu_usage"`
	MemoryUsed uint64  `json:"memory_used"`
	DiskUsed   uint64  `json:"disk_used"`
	Uptime     int64   `json:"uptime"`
}

type SignalMessage struct {
	Type    string `json:"type"`
	From    string `json:"from"`
	To      string `json:"to"`
	Payload string `json:"payload"`
}

type WSMessage struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
}
