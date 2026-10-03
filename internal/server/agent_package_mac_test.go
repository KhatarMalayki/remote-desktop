package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMacAgentPackage(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			directory := t.TempDir()
			binary := "MAC-" + arch
			if err := os.WriteFile(filepath.Join(directory, "rd-agent-darwin-"+arch), []byte(binary), 0700); err != nil {
				t.Fatal(err)
			}
			s := &Server{cfg: Config{AgentsDir: directory, APIKey: "fixture-secret"}}
			req := httptest.NewRequest("GET", "/api/agent/package?branch=MacOffice&os=darwin&arch="+arch, nil)
			req = req.WithContext(context.WithValue(req.Context(), userClaimsKey, &UserClaims{Username: "admin", Role: "admin"}))
			response := httptest.NewRecorder()
			s.handleAgentPackageDownload(response, req)
			if response.Code != 200 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if !strings.Contains(response.Header().Get("Content-Disposition"), "macOS-"+arch) {
				t.Fatal("archive name does not identify architecture")
			}
			archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
			if err != nil {
				t.Fatal(err)
			}
			contents := map[string]string{}
			for _, file := range archive.File {
				reader, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(reader)
				reader.Close()
				if err != nil {
					t.Fatal(err)
				}
				contents[file.Name] = string(data)
				if file.Name == "rd-agent" || strings.HasSuffix(file.Name, ".command") {
					if file.Mode().Perm() != 0700 {
						t.Fatalf("missing executable/private mode: %s %v", file.Name, file.Mode())
					}
				}
				if strings.HasSuffix(file.Name, ".bat") || strings.HasSuffix(file.Name, ".exe") {
					t.Fatal("Windows files in Mac archive")
				}
			}
			if len(contents) != 5 || contents["rd-agent"] != binary {
				t.Fatalf("unexpected archive contents: %v", contents)
			}
			var config map[string]interface{}
			if err := json.Unmarshal([]byte(contents["agent.json"]), &config); err != nil || config["api_key"] != "fixture-secret" || config["branch"] != "MacOffice" {
				t.Fatalf("invalid config: %v", err)
			}
			script := contents["pasang-otomatis.command"]
			for _, required := range []string{"launchctl bootstrap", "launchctl kickstart", "trap finish EXIT", "old-agent", "Konfigurasi lama", "plutil -insert Program -string", "sw_vers -productVersion", "Screen Recording", "Accessibility", "Secure Input"} {
				if !strings.Contains(script, required) {
					t.Fatalf("installer missing %s", required)
				}
			}
			if strings.Contains(script, "fixture-secret") || strings.Contains(script, "@ARCH@") || strings.Contains(script, "spctl") || strings.Contains(script, "xattr") {
				t.Fatal("installer exposes credentials or bypasses quarantine")
			}
		})
	}
}

func TestAgentBinaryPlatformIsolation(t *testing.T) {
	paths, err := agentBinaryCandidates("agents", "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if filepath.Base(path) != "rd-agent-darwin-arm64" {
			t.Fatalf("cross-platform fallback: %s", path)
		}
	}
	for _, platform := range []string{"../windows", "unknown"} {
		if _, err := agentBinaryCandidates("agents", platform, "amd64"); err == nil {
			t.Fatal("invalid platform accepted")
		}
	}
	if _, err := agentBinaryCandidates("agents", "darwin", "../arm64"); err == nil {
		t.Fatal("invalid architecture accepted")
	}
}

func TestMacInstallerShellSyntax(t *testing.T) {
	shell, err := exec.LookPath("bash")
	if runtime.GOOS == "windows" {
		shell = `C:\Program Files\Git\bin\bash.exe`
		_, err = os.Stat(shell)
	}
	if err != nil {
		t.Skip("bash unavailable; macOS runtime validation still required")
	}
	for _, script := range []string{macInstallScript, macUninstallScript} {
		command := exec.Command(shell, "-n", "-s")
		command.Stdin = strings.NewReader(script)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("shell syntax: %v %s", err, output)
		}
	}
}
