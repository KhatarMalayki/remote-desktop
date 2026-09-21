package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type rustDeskExportConfig struct {
	Host  string `json:"host"`
	Relay string `json:"relay"`
	API   string `json:"api"`
	Key   string `json:"key"`
}

// rustDeskCLIConfig converts RustDesk's reversed-base64 export into the
// plaintext CLI form. The latter avoids a Windows client bug where --config
// accepts an encoded string (notably one beginning with '=') but changes
// nothing and still exits successfully.
func rustDeskCLIConfig(exported string) (string, error) {
	exported = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(exported), `\`))
	if strings.HasPrefix(exported, "host=") {
		if strings.ContainsAny(exported, "\r\n\x00") {
			return "", fmt.Errorf("config RustDesk plaintext tidak valid")
		}
		if !strings.HasSuffix(exported, ",") {
			exported += ","
		}
		return exported, nil
	}
	if exported == "" {
		return "", fmt.Errorf("config RustDesk kosong")
	}
	runes := []rune(exported)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	decoded, err := base64.StdEncoding.DecodeString(string(runes))
	if err != nil {
		return "", fmt.Errorf("config RustDesk tidak dapat didekode: %w", err)
	}
	var config rustDeskExportConfig
	if err := json.Unmarshal(decoded, &config); err != nil {
		return "", fmt.Errorf("isi config RustDesk tidak valid: %w", err)
	}
	for _, value := range []string{config.Host, config.Relay, config.API, config.Key} {
		if strings.ContainsAny(value, ",\r\n\x00") {
			return "", fmt.Errorf("nilai config RustDesk tidak aman")
		}
	}
	if config.Host == "" || config.Key == "" {
		return "", fmt.Errorf("ID server atau key RustDesk kosong")
	}
	return fmt.Sprintf("host=%s,key=%s,relay=%s,api=%s,", config.Host, config.Key, config.Relay, config.API), nil
}
