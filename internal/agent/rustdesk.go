package agent

import (
	"regexp"
	"strings"
)

var rustDeskIDPattern = regexp.MustCompile(`^[0-9]{6,20}$`)

func parseRustDeskID(output string) string {
	for _, field := range strings.Fields(strings.TrimSpace(output)) {
		candidate := strings.TrimSpace(field)
		if rustDeskIDPattern.MatchString(candidate) {
			return candidate
		}
	}
	return ""
}
