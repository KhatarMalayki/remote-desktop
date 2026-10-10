package agent

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func lanTestState() vpnLANState {
	return vpnLANState{Name: "RemoteDeskPilotLAN-" + strings.Repeat("a", 32), Network: "10.77.0.0/24", Subnet: "192.168.1.0/24", External: "192.168.1.10/32", Interfaces: []vpnLANInterface{
		{Index: 10, GUID: "{11111111-1111-1111-1111-111111111111}", Forwarding: "Disabled"},
		{Index: 20, GUID: "{22222222-2222-2222-2222-222222222222}", Forwarding: "Enabled"},
	}}
}

func TestVPNLANJournalRollbackAndRestartCleanup(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	originalSecure := vpnLANSecureDirectory
	vpnLANSecureDirectory = func() error { return nil }
	t.Cleanup(func() { vpnLANSecureDirectory = originalSecure })
	original := vpnLANPowerShell
	t.Cleanup(func() { vpnLANPowerShell = original })
	state := lanTestState()
	applyFails, cleanupFails := true, true
	calls := 0
	vpnLANPowerShell = func(script, input string) ([]byte, error) {
		calls++
		var got vpnLANState
		if err := json.Unmarshal([]byte(input), &got); err != nil {
			t.Fatal(err)
		}
		if got.Interfaces[0].Forwarding != "Disabled" || got.Interfaces[1].Forwarding != "Enabled" {
			t.Fatal("original forwarding state lost")
		}
		if _, err := os.Stat(filepath.Join(vpnDirectory(), "lan.json")); err != nil {
			t.Fatal("network mutation before durable journal", err)
		}
		if script == vpnLANApplyScript && applyFails {
			return nil, errors.New("apply failed")
		}
		if script == vpnLANCleanupScript && cleanupFails {
			return nil, errors.New("cleanup failed")
		}
		return nil, nil
	}
	if err := vpnApplyLANState(state); err == nil {
		t.Fatal("failed apply ignored")
	}
	if calls != 2 {
		t.Fatalf("rollback not attempted: %d", calls)
	}
	if _, err := os.Stat(filepath.Join(vpnDirectory(), "lan.json")); err != nil {
		t.Fatal("failed cleanup lost journal")
	}
	if err := vpnApplyLANState(state); err == nil {
		t.Fatal("pending cleanup overwritten")
	}
	cleanupFails = false
	if err := vpnCleanupLAN(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vpnDirectory(), "lan.json")); !os.IsNotExist(err) {
		t.Fatal("successful cleanup retained journal")
	}
	applyFails = false
	if err := vpnApplyLANState(state); err != nil {
		t.Fatal(err)
	}
	if err := vpnCleanupLAN(); err != nil {
		t.Fatal(err)
	}
	before := calls
	if err := vpnCleanupLAN(); err != nil || calls != before {
		t.Fatal("cleanup is not idempotent")
	}
}

func TestVPNLANRejectsUnownedCleanup(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	originalSecure := vpnLANSecureDirectory
	vpnLANSecureDirectory = func() error { return nil }
	t.Cleanup(func() { vpnLANSecureDirectory = originalSecure })
	original := vpnLANPowerShell
	t.Cleanup(func() { vpnLANPowerShell = original })
	vpnLANPowerShell = func(string, string) ([]byte, error) { t.Fatal("invalid journal executed"); return nil, nil }
	for _, mutate := range []func(*vpnLANState){
		func(state *vpnLANState) { state.Name = "OtherApplication" },
		func(state *vpnLANState) { state.Network = "0.0.0.0/0" },
		func(state *vpnLANState) { state.External = "8.8.8.8/32" },
		func(state *vpnLANState) { state.Interfaces[0].Index = 0 },
		func(state *vpnLANState) { state.Interfaces[0].GUID = "injected" },
	} {
		state := lanTestState()
		mutate(&state)
		raw, _ := json.Marshal(state)
		if err := vpnAtomicFile("lan.json", raw); err != nil {
			t.Fatal(err)
		}
		if vpnCleanupLAN() == nil {
			t.Fatal("unowned/invalid journal accepted")
		}
	}
}

func TestVPNLANPowerShellSyntax(t *testing.T) {
	parser := `$tokens=$null;$errors=$null;[System.Management.Automation.Language.Parser]::ParseInput([Console]::In.ReadToEnd(),[ref]$tokens,[ref]$errors)|Out-Null;if($errors.Count){$errors|Out-String|Write-Output;exit 1}`
	for _, script := range []string{vpnLANInspectScript, vpnLANApplyScript, vpnLANCleanupScript} {
		command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", parser)
		command.Stdin = strings.NewReader(script)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("invalid PowerShell: %v %s", err, output)
		}
	}
}

func TestVPNLANDirectoryValidationPrecedesJournalExecution(t *testing.T) {
	t.Setenv("ProgramData", t.TempDir())
	if err := os.MkdirAll(vpnDirectory(), 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(lanTestState())
	if err := vpnAtomicFile("lan.json", raw); err != nil {
		t.Fatal(err)
	}
	originalSecure, originalShell := vpnLANSecureDirectory, vpnLANPowerShell
	t.Cleanup(func() { vpnLANSecureDirectory, vpnLANPowerShell = originalSecure, originalShell })
	denied := errors.New("directory ownership rejected")
	vpnLANSecureDirectory = func() error { return denied }
	vpnLANPowerShell = func(string, string) ([]byte, error) { t.Fatal("untrusted journal executed"); return nil, nil }
	if !errors.Is(vpnCleanupLAN(), denied) {
		t.Fatal("ownership rejection ignored")
	}
}
