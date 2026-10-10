# Windows VPN tray

The agent starts the tray in active Windows user sessions using the user's token,
not the SYSTEM account. It checks for new sessions or an exited tray every 30 seconds.
Only one tray runs per session. Explorer restarts restore the icon automatically.
No scheduled task or new autorun entry is installed.

- Hover or click the tray to see local tunnel service state and adapter IPv4.
  Active means the local service is running, not proof of a successful peer handshake.
- Connect requests approval through the installed SYSTEM agent. The server reads
  persistent device policy, denied by default. No browser or admin login is needed.
  The tray displays approval/denial; approval means preparation, not a handshake.
- Disconnect stops the verified local tunnel and asks the server to cancel any
  pending session. The existing watchdog handles lease/LAN cleanup. Errors are shown.
- The optional admin-panel link remains separate from Connect.
- The tray remains available while the agent runs; there is no Close menu.
- `RD_VPN_NO_TRAY=true` disables automatic launch.
- Unelevated portable agents launch their own tray. Elevated portable agents skip
  automatic launch; run `rd-agent.exe --vpn-tray --vpn-panel https://your-server/`
  from a normal user session instead. The tray refuses elevated execution.
  Self-service Connect requires the installed SYSTEM service, not portable mode.

## Admin policy

In **Administrasi > Master VPN**, find the device and select **Kelola**.
Enable **Connect mandiri dari tray**, optionally enter an
approved private LAN subnet, then save. This authorizes all interactive local users
on that device. The client cannot select another device, publish LAN, or supply routes.
LAN access additionally requires the active gateway's existing device whitelist.
Saving policy stops an active self-service session; disabling it prevents reconnect.
The server checks policy again before provisioning/renewing the lease. Failures to
read policy deny access. Cleanup failures remain visible and retry through the watchdog.

Gateway/subnet settings are also in Master VPN. Select the gateway, fill its subnet
and permitted client, then explicitly activate it. These remain session-scoped,
not persistent gateway policies. The device's **Koneksi VPN** dialog contains only
status, Connect/Disconnect, and a shortcut to Master VPN; opening it cannot apply
an unsaved gateway configuration from the master page.

This is device-level self-service on the existing pilot, not NetBird feature parity:
the two-device/15-minute limits remain; approved VPN members can communicate with
each other. No per-user SSO, peer/port policy, or automatic reconnect is added.
Authentication still trusts the existing enrolled agent/API-key WebSocket channel.

## Local boundary and verification

The local message-mode named pipe rejects remote clients. Interactive users receive
read/write access, not permission to create server pipe instances. Only fixed
connect/disconnect requests are accepted. API keys and tunnel keys stay in the agent.
Client SQOS prevents a spoofed pipe server from impersonating the tray user. Reads
are bounded; absent server responses do not count as approval. Rate limiting applies
to server connect requests. Disconnect is always available regardless of policy.

`go test ./internal/server ./internal/agent -run 'TestVPNSelf|TestVPNTray' -count=1`
covers policy, revocation, caller-config rejection, LAN whitelist, offline behavior,
status presentation, and a native Windows named-pipe round trip. Tests do not create
a real VPN or alter network configuration. Interactive verification still requires
an installed agent: login/RDP, Explorer restart, Connect, Disconnect, and LAN cleanup.
Windows may initially place the icon inside the notification overflow menu.
