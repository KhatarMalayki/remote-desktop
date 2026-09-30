package models

type TrackedApplication struct {
	Label string `json:"label"`
	Match string `json:"match"`
}

type InstalledApplication struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Service string `json:"service,omitempty"`
}

type EndpointReport struct {
	Status       string                 `json:"status"`
	Detail       string                 `json:"detail,omitempty"`
	Applications []InstalledApplication `json:"applications"`
	LockSeconds  uint32                 `json:"lock_seconds"`
	LockPresent  bool                   `json:"lock_present"`
	Managed      bool                   `json:"managed"`
	Drift        bool                   `json:"drift"`
	PolicyError  string                 `json:"policy_error,omitempty"`
}

type ApplicationDetection struct {
	Label   string                 `json:"label"`
	Status  string                 `json:"status"`
	Matches []InstalledApplication `json:"matches,omitempty"`
}

type LockPolicyRequest struct {
	ID      string `json:"id"`
	Mode    string `json:"mode"`
	Seconds uint32 `json:"seconds"`
}

func (request LockPolicyRequest) Valid() bool {
	return len(request.ID) == 32 && (request.Mode == "audit" || request.Mode == "restore" || request.Mode == "apply") && request.Seconds >= 60 && request.Seconds <= 86400
}

type LockPolicyResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type EndpointState struct {
	CheckedAt    int64                  `json:"checked_at"`
	Report       *EndpointReport        `json:"report,omitempty"`
	Applications []ApplicationDetection `json:"applications"`
	Policy       *LockPolicyResult      `json:"policy,omitempty"`
}
