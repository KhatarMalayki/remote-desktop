package agent

import "testing"

func TestRustDeskCLIConfigDecodesExport(t *testing.T) {
	exported := `\=0nI9c3Uah2dzBDUCV0KlVGN3sUZSljVQNXd1NkazgmVadjcJF1QBR1cRJnRIFlI6ISeltmIsIiI6ISawFmIsIiI6ISehxWZyJCLiUWbuk3Zvx2bul3cugWazFWa0FmaiojI0N3boJye`
	got, err := rustDeskCLIConfig(exported)
	if err != nil {
		t.Fatal(err)
	}
	want := "host=jatiasih.synology.me,key=QHFrQsTACQIr7ZVh3jCuusPV9ReK74ee+EBP0swhZSw=,relay=,api=,"
	if got != want {
		t.Fatalf("rustDeskCLIConfig()=%q, want %q", got, want)
	}
}

func TestRustDeskCLIConfigRejectsUnsafeValues(t *testing.T) {
	if _, err := rustDeskCLIConfig("host=example.com,key=bad\nvalue,"); err == nil {
		t.Fatal("expected unsafe plaintext config to be rejected")
	}
}
