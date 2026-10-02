package server

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var errPasswordScreenUnavailable = errors.New("Skrining password bocor belum tersedia; coba lagi nanti. Password belum diubah")

func checkPwnedPassword(ctx context.Context, client *http.Client, password string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	digest := fmt.Sprintf("%X", sha1.Sum([]byte(password)))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pwnedpasswords.com/range/"+digest[:5], nil)
	if err != nil {
		return false, errPasswordScreenUnavailable
	}
	req.Header.Set("Add-Padding", "true")
	req.Header.Set("User-Agent", "RemoteDesk-Password-Screening")
	boundedClient := *client
	boundedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := boundedClient.Do(req)
	if err != nil {
		return false, errPasswordScreenUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, errPasswordScreenUnavailable
	}
	const maxResponse = 2 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		return false, errPasswordScreenUnavailable
	}
	rows, breached := 0, false
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		suffix, countText, ok := strings.Cut(line, ":")
		count, parseErr := strconv.ParseUint(countText, 10, 64)
		if !ok || len(suffix) != 35 || parseErr != nil {
			return false, errPasswordScreenUnavailable
		}
		for _, digit := range suffix {
			if !strings.ContainsRune("0123456789ABCDEF", digit) {
				return false, errPasswordScreenUnavailable
			}
		}
		rows++
		if suffix == digest[5:] && count > 0 {
			breached = true
		}
	}
	if rows == 0 {
		return false, errPasswordScreenUnavailable
	}
	return breached, nil
}

func (s *Server) screenPassword(ctx context.Context, password string) (bool, error) {
	if s.breachCheck == nil {
		return false, errPasswordScreenUnavailable
	}
	return s.breachCheck(ctx, password)
}

func (s *Server) acceptNewPassword(w http.ResponseWriter, r *http.Request, password string) bool {
	if err := validatePassword(password); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return false
	}
	breached, err := s.screenPassword(r.Context(), password)
	if err != nil {
		jsonError(w, errPasswordScreenUnavailable.Error(), http.StatusServiceUnavailable)
		return false
	}
	if breached {
		jsonError(w, "Password ditemukan dalam database kebocoran; gunakan frasa lain yang unik", http.StatusBadRequest)
		return false
	}
	return true
}
