package agent

import "testing"

func TestDeploymentExitStatus(t *testing.T) {
	for code, want := range map[int]string{0: "succeeded", 3010: "reboot_required", 1641: "reboot_required", 1: "failed", 1603: "failed"} {
		got, _ := deploymentExitStatus(code)
		if got != want {
			t.Fatalf("exit %d: %s", code, got)
		}
	}
}
