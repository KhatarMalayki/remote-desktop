package agent

import "testing"

func TestValidRemoteSessionID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "uuid style", value: "sess-4e64c240-20ac-45ad-8058-49933a377534", valid: true},
		{name: "underscore", value: "session_test_1", valid: true},
		{name: "empty", value: "", valid: false},
		{name: "slash", value: "session/agent", valid: false},
		{name: "query injection", value: "session?key=stolen", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validRemoteSessionID(test.value); got != test.valid {
				t.Fatalf("validRemoteSessionID(%q) = %v, want %v", test.value, got, test.valid)
			}
		})
	}
}
