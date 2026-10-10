# Advertise LAN pilot (0.2.69)

Status: implemented for controlled testing. No real Windows NAT/hub/LAN end-to-end session has been verified by this change. Existing device-to-device VPN ping success is not evidence of subnet routing success.

## Scope and prerequisites

- Admin explicitly selects a Windows gateway, a canonical private IPv4 LAN CIDR (/16 through /30), and one permitted client device ID. No automatic discovery or advertisement of every adapter.
- Gateway and client require agent 0.2.69+, online status, and the updated server. Any registered Windows device may be selected; the gateway must pass elevated-service, active local subnet, NetNat, and adapter checks.
- Retains the pilot ceiling: two devices total, one gateway/subnet/client, 15 minutes, 60-second control lease. No persistence or automatic reconnect of the approved LAN session. Expand only after live lifecycle and failover tests.
- LAN CIDR must exactly match an active interface subnet on the gateway. The client and Linux hub must have no overlapping non-default routes. DNS/default routes remain unchanged. Client-resolved control and WireGuard endpoints must not fall inside the imported LAN route.
- Existing Windows NetNat or running Internet Connection Sharing is rejected, not removed or reconfigured. This can make a PC using container/virtualization networking ineligible. No Windows features or drivers are installed for LAN routing.

## Operation

1. Disconnect existing VPN sessions for the two selected devices.
2. Open the gateway's VPN dialog. Enable **Advertise LAN**, enter its local subnet (example `192.168.1.0/24`), and choose or type the permitted client's device ID. Confirm the displayed gateway, subnet, client, and temporary forwarding/NAT changes.
3. Wait for gateway **CONNECTED**. This verifies the VPN handshake and successful local setup, not reachability of every LAN host.
4. Open the permitted client's VPN dialog. Connect and confirm the exact LAN route. Other client IDs, changed subnet confirmations, offline/stale gateways, and older agents are rejected.
5. Test a known LAN host/service from the client. Host firewalls can still deny ping or application traffic; the feature does not disable them. LAN targets see the gateway's LAN address through NAT.
6. Disconnect the gateway to turn advertise off or change its configuration. The dependent client is also disconnected so its WireGuard-managed LAN route is removed. Reconnection requires a new explicit activation.

## Safety and cleanup

- The hub assigns the LAN AllowedIPs only to the gateway peer. Client AllowedIPs add only the approved LAN prefix. Linux forwarding defaults to deny except VPN-to-VPN and the explicit client `/32` to approved LAN rule; reverse LAN traffic requires an established/related flow. LAN-sourced traffic to the hub itself is denied.
- Hub route/firewall mutations record inverse commands in the session. Partial failures revoke the peer and retry outstanding cleanup. A pending resource cleanup blocks reuse; it is not reported as a successful disconnect. Runtime errors or externally altered resources may require operator inspection.
- The gateway writes `ProgramData/RemoteDeskVPN/lan.json` before changing forwarding/NAT. The journal stores the session-specific NAT name, exact prefixes, adapter GUIDs, and original per-interface forwarding settings. Only these resources are removed/restored. Ownership/configuration mismatches fail closed, retain the journal, and surface an error.
- Watchdog expiry, local service stop, explicit disconnect, agent update, and next agent startup use gateway cleanup. Failed cleanup is retried; journal removal occurs only after success. NAT removal does not delete unrelated NAT definitions. Forwarding changes use ActiveStore, not system-wide persistent routing flags.
- Activation and disconnect requests are recorded in the existing audit log. Session state remains in memory; server/container loss removes control keepalives and triggers the independent agent lease watchdog.

## Verification

- Shared Go tests cover canonical private subnet validation, default/public/overlapping routes, endpoint capture, one-role limits, and gateway/client AllowedIPs differences.
- Server tests mock `ip`, `wg`, and `iptables`: admin/version/allowlist checks, stale gateways, scoped ACLs, partial failure rollback, retry, and gateway-to-client cascading disconnect.
- Windows tests use temporary journals and mocked network/ACL calls. They test write-before-mutation, rollback, retained cleanup state, re-entry, ownership rejection, and PowerShell parser acceptance. They do not enable NAT or forwarding on the development machine.
- Browser unit tests cover opt-in, confirmation cancellation, client IDs outside the current device page, locked active configuration, and role restrictions.

Before publishing as verified LAN access, run a real two-PC test: connect/ping/service access, reject a non-approved client, reject overlapping LANs, stop gateway via tray, expire its lease, restart the gateway agent, and confirm `Get-NetNat`, per-interface forwarding, client routes, hub routes, and hub ACLs return to the previous state. Also verify rejection on a PC with existing NetNat/ICS without altering it.

## Upstream references reviewed

- Microsoft, NAT setup and one-NAT-network-per-host limitation: https://learn.microsoft.com/en-us/virtualization/hyper-v-on-windows/user-guide/setup-nat-network
- Microsoft, New-NetNat prefix parameters: https://learn.microsoft.com/en-us/powershell/module/netnat/new-netnat
- Microsoft, Set-NetIPInterface Forwarding and PolicyStore: https://learn.microsoft.com/en-us/powershell/module/nettcpip/set-netipinterface
- Microsoft, Get-NetAdapter interface index selection: https://learn.microsoft.com/en-us/powershell/module/netadapter/get-netadapter

Documentation was consulted on 2026-10-10. The installed Windows edition/module support and actual network forwarding still require the live checks above.
