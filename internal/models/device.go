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
	RustDeskID         string     `json:"rustdesk_id"`
	Status             string     `json:"status"`
	Tags               string     `json:"tags"`
	GroupName          string     `json:"group"`
	Branch             string     `json:"branch"`
	VerificationStatus string     `json:"verification_status"`
	VerifiedAt         *time.Time `json:"verified_at,omitempty"`
	VerifiedBy         string     `json:"verified_by"`
	VerificationNote   string     `json:"verification_note"`
	Note               string     `json:"note"`
	AcquisitionYear    int        `json:"acquisition_year"`
	OwnerUsername      string     `json:"owner_username"`
	AssignedTo         string     `json:"assigned_to"`
	Condition          string     `json:"condition"`
	Recommendation     string     `json:"recommendation"`
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
	AcquisitionYear    int        `json:"acquisition_year"`
	OwnerUsername      string     `json:"owner_username"`
	Recommendation     string     `json:"recommendation"`
	CreatedBy          string     `json:"created_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type AssetVerification struct {
	ID         int64     `json:"id"`
	AssetID    string    `json:"asset_id"`
	AssetType  string    `json:"asset_type"` // manual, device
	AssetName  string    `json:"asset_name"`
	Status     string    `json:"status"` // verified, discrepancy, unverified
	Condition  string    `json:"condition"`
	VerifiedBy string    `json:"verified_by"`
	Branch     string    `json:"branch"`
	Notes      string    `json:"notes"`
	CreatedAt  time.Time `json:"created_at"`
}

type RoleDefinition struct {
	Role        string          `json:"role"`
	Name        string          `json:"name"`
	BadgeColor  string          `json:"badge_color"`
	Scope       string          `json:"scope"`
	Description string          `json:"description"`
	Permissions map[string]bool `json:"permissions"`
}

type User struct {
	ID         int64     `json:"id"`
	Username   string    `json:"username"`
	Role       string    `json:"role"` // admin, it_support, ga_pusat, adh, spv, user, viewer
	Branch     string    `json:"branch"`
	MFAEnabled bool      `json:"mfa_enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

type AssetSwitchRequest struct {
	SwapAssetID    string            `json:"swap_asset_id"`
	SwapAssetType  string            `json:"swap_asset_type"`
	ID             int64             `json:"id"`
	AssetID        string            `json:"asset_id"`
	AssetType      string            `json:"asset_type"`
	AssetName      string            `json:"asset_name"`
	Branch         string            `json:"branch"`
	FromOwner      string            `json:"from_owner"`
	ToOwner        string            `json:"to_owner"`
	AssignedTo     string            `json:"assigned_to"`
	RequestedBy    string            `json:"requested_by"`
	Reason         string            `json:"reason"`
	Status         string            `json:"status"`
	ReviewedBy     string            `json:"reviewed_by"`
	ReviewedAt     *time.Time        `json:"reviewed_at,omitempty"`
	ReviewNote     string            `json:"review_note"`
	Responsibility string            `json:"responsibility"`
	Recommendation string            `json:"recommendation"`
	Operation      string            `json:"operation"` // assignment, handover
	Attachments    []AssetAttachment `json:"attachments,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type AssetAttachment struct {
	ID           int64     `json:"id"`
	EntityType   string    `json:"entity_type"`
	EntityID     int64     `json:"entity_id"`
	AssetID      string    `json:"asset_id"`
	OriginalName string    `json:"original_name"`
	StoredName   string    `json:"-"`
	ContentType  string    `json:"content_type"`
	Size         int64     `json:"size"`
	UploadedBy   string    `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
}

type AssetActivity struct {
	ID            int64     `json:"id"`
	Category      string    `json:"category"`
	Action        string    `json:"action"`
	Actor         string    `json:"actor"`
	Branch        string    `json:"branch"`
	AssetID       string    `json:"asset_id"`
	AssetType     string    `json:"asset_type"`
	AssetName     string    `json:"asset_name"`
	Detail        string    `json:"detail"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   int64     `json:"reference_id"`
	CreatedAt     time.Time `json:"created_at"`
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
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"` // pusat, cabang, bisnis_unit, service_point, pool, site
	BusinessUnit  string    `json:"business_unit"`
	BusinessUnits []string  `json:"business_units"`
	CreatedAt     time.Time `json:"created_at"`
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
	RustDeskID string  `json:"rustdesk_id"`
}

type RustDeskCommand struct {
	RequestID   string `json:"request_id"`
	Operation   string `json:"operation"`
	Password    string `json:"password,omitempty"`
	Config      string `json:"config,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

type RustDeskResult struct {
	RequestID  string `json:"request_id"`
	Operation  string `json:"operation"`
	RustDeskID string `json:"rustdesk_id,omitempty"`
	Error      string `json:"error,omitempty"`
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

type NetworkScanHost struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname,omitempty"`
}

type NetworkScan struct {
	ID         string            `json:"id"`
	DeviceID   string            `json:"device_id"`
	Subnet     string            `json:"subnet,omitempty"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	Hosts      []NetworkScanHost `json:"hosts,omitempty"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt *time.Time        `json:"finished_at,omitempty"`
}
