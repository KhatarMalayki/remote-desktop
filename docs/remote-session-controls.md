# Remote session controls (0.2.52)

Update both server and endpoint agent. Buttons remain disabled for agents without
the capability handshake. Existing remote access role/location checks apply.
Control requests are audited as requests, not proof of successful native actions.
No file contents, passwords or key events are written to the audit log.

## Local input

Blokir Input uses Windows BlockInput on the same pinned OS thread that injects
remote input. Ctrl+Alt+Del on the local keyboard remains the Windows escape hatch.
Input release is attempted on disconnect, errors, desktop change and missed
15-second heartbeat. Native release failures are logged rather than reported as
success. BlockInput automatically releases if its owning thread exits.

## Privacy / layar hitam

Windows 10 2004+ with desktop composition, on Default desktop only. An opaque
topmost layered overlay covers the virtual screen, with WDA_EXCLUDEFROMCAPTURE
to keep it out of the remote image. A dedicated window thread avoids breaking
the existing capture/input desktop binding. Privacy also blocks local input.
Turn privacy off, then use Lepas Blokir Input to restore local input.

This is a best-effort visual curtain, NOT a confidentiality boundary. Lock screen,
UAC, exclusive fullscreen, display drivers and other topmost windows can behave
differently. Desktop changes (such as Win+L locking the PC to Winlogon) do NOT disconnect the
remote relay. The agent dynamically follows to Winlogon on the same connection,
re-asserts BlockInput on Winlogon so the local user cannot enter PIN/password physically,
and keeps the privacy overlay ready on Default so it covers the screen immediately upon unlock. The overlay has an independent 20-second expiry if the capture loop
stops servicing it. Notify the local user before enabling it. Verify physical
monitors and the viewer on the actual endpoint before using it for sensitive work.

## Send file

One file at a time, up to 100 MiB. Browser HTTPS/localhost is required for SHA-256.
The browser sends acknowledged 48 KiB chunks; the agent checks offsets, size,
filename and final SHA-256. It creates a unique RemoteDesk-received-* directory
under the agent account's temporary directory; the UI displays the exact path.
SYSTEM-run agents may require administrator access to that directory. Move the
received file elsewhere to retain it permanently; OS cleanup may remove temp files.
No arbitrary destination, overwriting existing files, auto-opening or execution.
Failed/cancelled/incomplete transfers are removed on cleanup. A 45-second transfer
inactivity timeout also closes the session; 30-second browser acknowledgement
timeout disconnects rather than risking stale replies. A lost final acknowledgement
means the result is unconfirmed: the complete file may already exist on the endpoint.

## Verification

- go test ./internal/agent: filename/size/offset/checksum/cleanup and mocked BlockInput.
- node --test web/remote.test.cjs: chunk protocol, confirmation and disconnect handling.
- Physical endpoint acceptance still required: normal remote clicks while input is
  blocked, all physical screens black while viewer stays usable, multi-monitor and
  resize, Ctrl+Alt+Del/UAC/lock transitions, disconnect/network loss, file hash/path.
- Do not interpret unit tests or a successful build as physical privacy verification.

## Native references checked

- Microsoft BlockInput: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-blockinput
- Microsoft SetWindowDisplayAffinity: https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowdisplayaffinity
- Installed github.com/kbinani/screenshot, commit 089614a94018, windows.go:
  Capture uses BitBlt with SRCCOPY. No third-party implementation was downloaded or executed.
