package server

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type breachTransport func(*http.Request) (*http.Response, error)

func (transport breachTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestPwnedPasswordRangePrivacyAndFailures(t *testing.T) {
	password := "  frasa unik dengan spasi 8491  "
	digest := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	for _, fixture := range []struct {
		name, body string
		status     int
		breached   bool
		failed     bool
	}{
		{"found", digest[5:] + ":19\r\n", 200, true, false},
		{"padding", digest[5:] + ":0\r\n", 200, false, false},
		{"not found", strings.Repeat("0", 35) + ":25\n", 200, false, false},
		{"unavailable", "", 503, false, true},
		{"redirect", "", 302, false, true},
		{"empty", "", 200, false, true},
		{"invalid", "not a hash:10", 200, false, true},
		{"invalid count", digest[5:] + ":-1", 200, false, true},
		{"oversized", strings.Repeat("x", (2<<20)+1), 200, false, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: breachTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.String() != "https://api.pwnedpasswords.com/range/"+digest[:5] || req.Body != nil || req.Header.Get("Add-Padding") != "true" || req.Header.Get("Authorization") != "" {
					t.Fatal("lookup sent more than the padded hash-prefix request")
				}
				if _, ok := req.Context().Deadline(); !ok {
					t.Fatal("lookup has no deadline")
				}
				return &http.Response{StatusCode: fixture.status, Header: http.Header{"Location": []string{"https://other.invalid/"}}, Body: io.NopCloser(strings.NewReader(fixture.body)), Request: req}, nil
			})}
			breached, err := checkPwnedPassword(context.Background(), client, password)
			if breached != fixture.breached || (err != nil) != fixture.failed || calls != 1 {
				t.Fatalf("breached=%t err=%v calls=%d", breached, err, calls)
			}
		})
	}
	client := &http.Client{Transport: breachTransport(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := checkPwnedPassword(ctx, client, password); err == nil {
		t.Fatal("cancelled lookup reported success")
	}
}

func TestBreachScreeningAcrossPasswordRoutes(t *testing.T) {
	oldPassword := "original account phrase 5892"
	oldHash := hashPassword(oldPassword)
	newPassword := "new account phrase 93841"
	for _, outcome := range []string{"breached", "unavailable", "clean"} {
		for _, route := range []string{"create", "edit", "reset", "change"} {
			t.Run(outcome+"/"+route, func(t *testing.T) {
				s := securityServer(t)
				if err := s.db.CreateUser("target", oldHash, "admin", ""); err != nil {
					t.Fatal(err)
				}
				calls := 0
				s.breachCheck = func(_ context.Context, password string) (bool, error) {
					calls++
					if password != newPassword {
						t.Fatal("wrong password screened")
					}
					if outcome == "unavailable" {
						return false, errPasswordScreenUnavailable
					}
					return outcome == "breached", nil
				}
				method, path, expected := "POST", "/api/users", 201
				body := map[string]string{"username": "another", "password": newPassword, "role": "viewer"}
				handler := s.handleUsers
				switch route {
				case "edit":
					method, path, expected, handler = "PUT", "/api/users/1", 200, s.handleUserUpdate
					body["username"] = "target"
					body["role"] = "admin"
				case "reset":
					path, expected, handler = "/api/users/1/reset-password", 200, s.handleResetUserPassword
					body = map[string]string{"new_password": newPassword}
				case "change":
					path, expected, handler = "/api/auth/change-password", 200, s.handleChangePassword
					body = map[string]string{"old_password": oldPassword, "new_password": newPassword, "confirm_password": newPassword}
				}
				if outcome == "breached" {
					expected = 400
				} else if outcome == "unavailable" {
					expected = 503
				}
				payload, _ := json.Marshal(body)
				req := httptest.NewRequest(method, path, strings.NewReader(string(payload)))
				req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "target", Role: "admin"}))
				response := httptest.NewRecorder()
				handler(response, req)
				if response.Code != expected || calls != 1 {
					t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
				}
				if outcome != "clean" {
					_, actualHash, role, _, err := s.db.GetUser("target")
					if err != nil || actualHash != oldHash || role != "admin" {
						t.Fatal("failed screening changed account data")
					}
					if _, _, _, _, err := s.db.GetUser("another"); err == nil {
						t.Fatal("failed screening created an account")
					}
				}
			})
		}
	}
}

func TestBreachScreeningAtLogin(t *testing.T) {
	password := "existing user phrase 64920"
	storedHash := hashPassword(password)
	for _, outcome := range []string{"breached", "unavailable", "clean"} {
		t.Run(outcome, func(t *testing.T) {
			s := securityServer(t)
			if err := s.db.CreateUser("target", storedHash, "admin", ""); err != nil {
				t.Fatal(err)
			}
			if err := s.db.UpdatePassword("target", storedHash); err != nil {
				t.Fatal(err)
			}
			calls := 0
			s.breachCheck = func(context.Context, string) (bool, error) {
				calls++
				if outcome == "unavailable" {
					return false, errPasswordScreenUnavailable
				}
				return outcome == "breached", nil
			}
			authRequest(t, s, "/api/auth/login", "", map[string]string{"username": "target", "password": "wrong"}, s.handleLogin, 401)
			if calls != 0 {
				t.Fatal("incorrect credentials reached external screening")
			}
			authRequest(t, s, "/api/auth/login", "", map[string]string{"username": "target", "password": password}, s.handleLogin, 200)
			_, _, _, mustChange, err := s.accountState("target")
			if err != nil || mustChange != (outcome == "breached") || calls != 1 {
				t.Fatalf("mustChange=%t err=%v calls=%d", mustChange, err, calls)
			}
			if outcome != "clean" {
				status := "password_compromised"
				if outcome == "unavailable" {
					status = "password_screen_unavailable"
				}
				var count int
				if err := s.db.db.QueryRow("SELECT COUNT(*) FROM auth_logs WHERE status=?", status).Scan(&count); err != nil || count != 1 {
					t.Fatalf("audit status missing: %v count=%d", err, count)
				}
			}
		})
	}
}
