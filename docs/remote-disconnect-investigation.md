# Remote disconnect investigation — 2026-09-30

## Verified in source

- Agent sends unchanged frames at least every two seconds when capture succeeds. Idle silence and a specific proxy timeout are not established causes.
- Capture previously terminated the relay on its first error. Capture now retries for up to ten seconds, reports interruption, then reports persistent failure before closing.
- Browser previously allowed only two desktop-transition reconnects in ten seconds. It now allows five in thirty seconds, resets on manual connection, and cancels pending reconnect on Stop.
- Dashboard inactivity logout remains thirty minutes. Protection heartbeat expiry still releases protection and ends the session; this safety behavior is unchanged.
- Relay now sends protocol pings every twenty seconds, bounds data writes to ten seconds, and sends a close reason to the viewer. These are resilience measures, not proof of a proxy fault.
- Viewer displays the observed connection leg or explicit agent error. An abnormal browser close cannot identify whether operator network, proxy, or server failed.

## Verification and limits

`go test ./...`, `node --test web/*.test.cjs`, and JavaScript syntax checking passed. A real local WebSocket test verifies that the relay close reason reaches the viewer. Browser reason classification is tested independently.

UAC switching, sustained idle through the deployed proxy, and connection loss on the actual endpoint remain untested. Reproduce those after deploying both server and agent; correlate the displayed reason with timestamped agent/server logs. Do not call the incident resolved from unit tests alone.
