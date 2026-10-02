package server

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"
)

func agentBinaryCandidates(directory, platform, arch string) ([]string, error) {
	if (platform != "windows" && platform != "linux" && platform != "darwin") || (arch != "amd64" && arch != "arm64") {
		return nil, fmt.Errorf("platform atau arsitektur agent tidak didukung")
	}
	name := "rd-agent-" + platform + "-" + arch
	if platform == "windows" {
		name += ".exe"
	}
	paths := []string{}
	if directory != "" {
		paths = append(paths, filepath.Join(directory, name))
	}
	paths = append(paths, filepath.Join("bin", "agents", name), filepath.Join("bin", name))
	if platform == "windows" && arch == "amd64" {
		if directory != "" {
			paths = append(paths, filepath.Join(directory, "rd-agent.exe"))
		}
		paths = append(paths, filepath.Join("bin", "rd-agent.exe"), "rd-agent.exe")
	}
	return paths, nil
}

func addMacInstallers(writer *zip.Writer, arch string) error {
	nativeArch := "x86_64"
	if arch == "arm64" {
		nativeArch = "arm64"
	}
	for name, content := range map[string]string{
		"pasang-otomatis.command": strings.ReplaceAll(macInstallScript, "@ARCH@", nativeArch),
		"hapus-otomatis.command":  macUninstallScript,
	} {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0700)
		file, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err := file.Write([]byte(content)); err != nil {
			return err
		}
	}
	return nil
}

const macInstallScript = `#!/bin/bash
set -euo pipefail
umask 077
fail() { printf '%s\n' "$1" >&2; exit 1; }
[[ "$(uname -s)" == Darwin ]] || fail 'Installer ini hanya untuk macOS.'
[[ "$(uname -m)" == '@ARCH@' ]] || fail 'Arsitektur tidak cocok. Unduh paket Intel atau Apple Silicon yang sesuai.'
[[ "$(id -u)" != 0 ]] || fail 'Jalankan dari akun pengguna yang login, tanpa sudo.'
[[ ! -e /Library/LaunchDaemons/com.remotedesk.agent.plist ]] || fail 'Instalasi sistem lama terdeteksi. Minta admin memigrasikannya sebelum memasang agent per-user.'
source_dir="$(cd -- "$(dirname -- "$0")" && pwd -P)"
target="$HOME/Library/Application Support/RemoteDesk"
plist="$HOME/Library/LaunchAgents/com.remotedesk.agent.plist"
domain="gui/$(id -u)"
job="$domain/com.remotedesk.agent"
launchctl print "$domain" >/dev/null || fail 'Login ke desktop macOS sebelum memasang.'
[[ -f "$source_dir/rd-agent" && -f "$source_dir/agent.json" ]] || fail 'Ekstrak seluruh isi ZIP terlebih dahulu.'
plutil -lint -s "$source_dir/agent.json" || fail 'Konfigurasi paket tidak valid.'
[[ ! -L "$target" && ! -L "$plist" ]] || fail 'Lokasi instalasi berupa symbolic link; dibatalkan.'
mkdir -p "$target" "$HOME/Library/LaunchAgents"
chmod 700 "$target"
for name in rd-agent agent.json agent.log; do
  [[ ! -L "$target/$name" ]] || fail 'File instalasi berupa symbolic link; dibatalkan.'
done
if [[ -f "$target/agent.json" ]]; then
  plutil -lint -s "$target/agent.json" || fail 'Konfigurasi lama rusak; tidak ditimpa.'
fi
stage="$(mktemp -d "$target/.install.XXXXXX")"
changed=0
finish() {
  status=$?
  trap - EXIT
  if [[ $status != 0 && $changed == 1 ]]; then
    launchctl bootout "$job" >/dev/null 2>&1 || true
    if [[ -f "$stage/old-agent" ]]; then cp -p "$stage/old-agent" "$target/rd-agent"; fi
    if [[ -f "$stage/old.plist" ]]; then
      cp -p "$stage/old.plist" "$plist"
      launchctl bootstrap "$domain" "$plist" || printf '%s\n' 'Pemulihan auto-start gagal; hubungi admin.' >&2
    else
      rm -f "$plist"
    fi
    printf '%s\n' 'Instalasi gagal. Konfigurasi dan log tetap disimpan.' >&2
  fi
  rm -f "$stage/rd-agent" "$stage/new.plist" "$stage/old-agent" "$stage/old.plist"
  rmdir "$stage"
  exit "$status"
}
trap finish EXIT
cp "$source_dir/rd-agent" "$stage/rd-agent"
chmod 700 "$stage/rd-agent"
if [[ -f "$target/rd-agent" ]]; then cp -p "$target/rd-agent" "$stage/old-agent"; fi
if [[ -f "$plist" ]]; then cp -p "$plist" "$stage/old.plist"; fi
plutil -create xml1 "$stage/new.plist"
plutil -insert Label -string com.remotedesk.agent "$stage/new.plist"
plutil -insert Program -string "$target/rd-agent" "$stage/new.plist"
plutil -insert WorkingDirectory -string "$target" "$stage/new.plist"
plutil -insert RunAtLoad -bool YES "$stage/new.plist"
plutil -insert KeepAlive -bool YES "$stage/new.plist"
plutil -insert ThrottleInterval -integer 30 "$stage/new.plist"
plutil -insert StandardOutPath -string "$target/agent.log" "$stage/new.plist"
plutil -insert StandardErrorPath -string "$target/agent.log" "$stage/new.plist"
plutil -lint -s "$stage/new.plist"
if launchctl print "$job" >/dev/null 2>&1; then launchctl bootout "$job"; fi
changed=1
if [[ ! -f "$target/agent.json" ]]; then
  cp "$source_dir/agent.json" "$target/agent.json"
else
  printf '%s\n' 'Konfigurasi lama, server, lokasi, dan device ID dipertahankan.'
fi
chmod 600 "$target/agent.json"
mv -f "$stage/rd-agent" "$target/rd-agent"
mv -f "$stage/new.plist" "$plist"
chmod 600 "$plist"
touch "$target/agent.log"
chmod 600 "$target/agent.log"
launchctl bootstrap "$domain" "$plist"
launchctl kickstart "$job"
launchctl print "$job" >/dev/null
printf '%s\n' 'Auto-start RemoteDesk terdaftar untuk akun ini. Periksa status online di dashboard dan agent.log; registrasi belum membuktikan koneksi server berhasil.'
printf '%s\n' 'Agent berjalan setelah login pengguna, bukan sebelum login. Installer ini tidak menambahkan dukungan kontrol keyboard/mouse macOS.'
read -r -p 'Tekan Enter untuk menutup...' || true
`

const macUninstallScript = `#!/bin/bash
set -euo pipefail
[[ "$(uname -s)" == Darwin && "$(id -u)" != 0 ]] || { printf '%s\n' 'Jalankan dari akun macOS yang memasang agent, tanpa sudo.'; exit 1; }
read -r -p 'Hapus auto-start RemoteDesk untuk akun ini? Ketik YA: ' answer
[[ "$answer" == YA ]] || exit 0
job="gui/$(id -u)/com.remotedesk.agent"
if launchctl print "$job" >/dev/null 2>&1; then launchctl bootout "$job"; fi
rm -f "$HOME/Library/LaunchAgents/com.remotedesk.agent.plist"
printf '%s\n' 'Auto-start dihapus. Binary, konfigurasi, device ID, dan log tetap disimpan di ~/Library/Application Support/RemoteDesk.'
read -r -p 'Tekan Enter untuk menutup...' || true
`
