package agent

import "testing"

func TestUserRelayDelegation(t *testing.T) {
	if !shouldDelegateUserRelay("Default", false) {
		t.Fatal("Default desktop must use interactive user relay")
	}
	if shouldDelegateUserRelay("Winlogon", false) {
		t.Fatal("Winlogon must use secure relay")
	}
	if shouldDelegateUserRelay("Default", true) {
		t.Fatal("forced secure transition must not start user relay")
	}
}
