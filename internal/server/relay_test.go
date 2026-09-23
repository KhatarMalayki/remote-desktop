package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/remote-desktop/internal/models"
)

func TestValidRelayID(t *testing.T) {
	for value, want := range map[string]bool{
		"sess-123_ABC":    true,
		"":                false,
		"../admin":        false,
		"session?token=x": false,
	} {
		if got := validRelayID(value); got != want {
			t.Fatalf("validRelayID(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestRelayRejectsUnauthorizedConnectionsBeforeUpgrade(t *testing.T) {
	db, err := NewDB(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dev := &models.Device{ID: "device-1", Hostname: "PC Bandung", Branch: "Bandung", GroupName: "Bandung", LastSeen: time.Now(), RegisteredAt: time.Now()}
	if err := db.UpsertDevice(dev); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, hub: NewHub(db), cfg: Config{APIKey: "agent-key", JWTSecret: "jwt-secret"}}

	for _, u := range []struct{ name, role, branch string }{{"holder", "user", "Bandung"}, {"adh-medan", "adh", "Medan"}} {
		if err := db.CreateUser(u.name, "fixture", u.role, u.branch); err != nil {
			t.Fatal(err)
		}
		if _, err := db.db.Exec(`UPDATE users SET mfa_enabled=1, mfa_secret='fixture', must_change_password=0 WHERE username=?`, u.name); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name string
		url  string
		code int
	}{
		{name: "viewer without token", url: "/ws/relay/sess-1/viewer?device_id=device-1", code: http.StatusUnauthorized},
		{name: "holder cannot remote", url: "/ws/relay/sess-1/viewer?device_id=device-1&token=" + s.issueCredentialToken("holder", "session"), code: http.StatusForbidden},
		{name: "adh other branch", url: "/ws/relay/sess-1/viewer?device_id=device-1&token=" + s.issueCredentialToken("adh-medan", "session"), code: http.StatusForbidden},
		{name: "agent wrong key", url: "/ws/relay/sess-1/agent?device_id=device-1&key=wrong", code: http.StatusUnauthorized},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			s.handleRelayWS(recorder, httptest.NewRequest(http.MethodGet, test.url, nil))
			if recorder.Code != test.code {
				t.Fatalf("got status %d, want %d", recorder.Code, test.code)
			}
		})
	}
}

func TestCloseRelaySessionWritesEndAuditOnce(t *testing.T) {
	startedAt := time.Now().Add(-3 * time.Second)
	var endCount int
	var duration time.Duration
	sess := &relaySession{
		deviceID:  "device-1",
		started:   true,
		startedAt: startedAt,
		audit: relayAudit{onEnd: func(value time.Duration) {
			endCount++
			duration = value
		}},
	}
	relayMu.Lock()
	relaySessions["audit-session"] = sess
	relayMu.Unlock()

	closeRelaySession("audit-session", sess)
	closeRelaySession("audit-session", sess)
	if endCount != 1 {
		t.Fatalf("end audit count = %d, want 1", endCount)
	}
	if duration < 2*time.Second || duration > 4*time.Second {
		t.Fatalf("unexpected audited duration: %s", duration)
	}
}
