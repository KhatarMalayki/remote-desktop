package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func GenerateTOTPSecret() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

func GenerateTOTPCode(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	code := binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff
	code = code % 1000000
	return fmt.Sprintf("%06d", code), nil
}

func ValidateTOTPCode(secret, code string) bool {
	cleanCode := strings.TrimSpace(code)
	if len(cleanCode) != 6 {
		return false
	}
	now := time.Now()
	// Check current, previous (-30s), and next (+30s) intervals
	for _, offset := range []time.Duration{-30 * time.Second, 0, 30 * time.Second} {
		c, err := GenerateTOTPCode(secret, now.Add(offset))
		if err == nil && c == cleanCode {
			return true
		}
	}
	return false
}

func generateMFATicket(username, role, branch, secret string) string {
	// Ticket valid for 5 minutes only
	payload := fmt.Sprintf("mfa|%s|%s|%s|%d", username, role, branch, time.Now().Add(5*time.Minute).Unix())
	h := sha256.Sum256([]byte(payload + secret))
	sig := hex.EncodeToString(h[:8])
	return hex.EncodeToString([]byte(payload)) + "." + sig
}

func parseMFATicket(ticket, secret string) (*UserClaims, bool) {
	parts := strings.SplitN(ticket, ".", 2)
	if len(parts) != 2 {
		return nil, false
	}
	payloadBytes, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, false
	}
	payload := string(payloadBytes)

	h := sha256.Sum256([]byte(payload + secret))
	expectedSig := hex.EncodeToString(h[:8])
	if parts[1] != expectedSig {
		return nil, false
	}

	fields := strings.Split(payload, "|")
	if len(fields) != 5 || fields[0] != "mfa" {
		return nil, false
	}
	exp, err := strconv.ParseInt(fields[4], 10, 64)
	if err != nil || time.Now().Unix() >= exp {
		return nil, false
	}
	return &UserClaims{
		Username: fields[1],
		Role:     fields[2],
		Branch:   fields[3],
	}, true
}
