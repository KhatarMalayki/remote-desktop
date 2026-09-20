package versioncmp

import (
	"strconv"
	"strings"
)

// IsNewer reports whether candidate is a strictly newer dotted-numeric version
// than current. Invalid or incomplete version strings are rejected so an update
// source can never replace a running agent with an unknown or older build.
func IsNewer(candidate, current string) bool {
	candidateParts, candidateOK := parse(candidate)
	currentParts, currentOK := parse(current)
	if !candidateOK || !currentOK {
		return false
	}

	length := len(candidateParts)
	if len(currentParts) > length {
		length = len(currentParts)
	}
	for i := 0; i < length; i++ {
		var candidatePart, currentPart int
		if i < len(candidateParts) {
			candidatePart = candidateParts[i]
		}
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}
		if candidatePart != currentPart {
			return candidatePart > currentPart
		}
	}
	return false
}

func parse(value string) ([]int, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	value = strings.SplitN(value, "-", 2)[0]
	if value == "" {
		return nil, false
	}

	rawParts := strings.Split(value, ".")
	parts := make([]int, len(rawParts))
	for i, raw := range rawParts {
		if raw == "" {
			return nil, false
		}
		part, err := strconv.Atoi(raw)
		if err != nil || part < 0 {
			return nil, false
		}
		parts[i] = part
	}
	return parts, true
}
