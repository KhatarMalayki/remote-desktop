package versioncmp

import "testing"

func TestIsNewer(t *testing.T) {
	tests := []struct {
		candidate string
		current   string
		want      bool
	}{
		{"0.2.12", "0.2.11", true},
		{"v1.0.0", "0.9.9", true},
		{"0.2.11", "0.2.11", false},
		{"0.1.4", "0.2.11", false},
		{"0.2", "0.2.0", false},
		{"bad", "0.2.11", false},
		{"0.2.12", "bad", false},
	}
	for _, test := range tests {
		if got := IsNewer(test.candidate, test.current); got != test.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", test.candidate, test.current, got, test.want)
		}
	}
}
