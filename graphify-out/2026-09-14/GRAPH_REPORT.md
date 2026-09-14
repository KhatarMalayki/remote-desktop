# Graph Report - remote-desktop  (2026-09-14)

## Corpus Check
- 27 files · ~24,630 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 414 nodes · 805 edges · 22 communities (17 shown, 5 thin omitted)
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 41 edges (avg confidence: 0.77)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `f0736867`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Community 0
- Community 1
- Community 2
- Community 3
- Community 4
- Community 5
- Community 6
- Community 7
- Community 8
- Community 9
- Community 10
- Community 11
- Community 12
- Community 13
- AGENTS.md
- main
- update-server.sh
- TestAgentVersionEndpoint
- qrcode.min.js

## God Nodes (most connected - your core abstractions)
1. `Server` - 42 edges
2. `DB` - 37 edges
3. `jsonResp()` - 29 edges
4. `jsonError()` - 27 edges
5. `api()` - 25 edges
6. `getClaims()` - 22 edges
7. `showToast()` - 18 edges
8. `Agent` - 15 edges
9. `Hub` - 14 edges
10. `loadDevices()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `New()` --calls--> `NewDB()`  [INFERRED]
  internal/server/server.go → internal/server/db.go
- `TestDBAndBranchVerification()` --calls--> `hashPassword()`  [INFERRED]
  internal/server/db_test.go → internal/server/server.go

## Import Cycles
- None detected.

## Communities (22 total, 5 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (20): Agent, AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), generateDeviceID(), Conn (+12 more)

### Community 1 - "Community 1"
Cohesion: 0.13
Nodes (24): FS, HandlerFunc, checkLoginRateLimit(), generateToken(), getClaims(), Hub, Time, hashPassword() (+16 more)

### Community 2 - "Community 2"
Cohesion: 0.07
Nodes (14): Time, migrate(), scanDevice(), scanDeviceRows(), scanManualAsset(), AssetVerification, Device, DeviceHeartbeat (+6 more)

### Community 3 - "Community 3"
Cohesion: 0.14
Nodes (7): Conn, Hub, Hub, NewHub(), SignalMessage, RWMutex, Client

### Community 4 - "Community 4"
Cohesion: 0.06
Nodes (73): api(), buildDeviceTable(), CHART_COLORS, chartDefaults(), checkAuth(), closeChangePasswordModal(), closeDeviceModal(), closeManualAssetModal() (+65 more)

### Community 5 - "Community 5"
Cohesion: 0.06
Nodes (28): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya, Cara Install RemoteDesk, Checklist Setelah Install, Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute (+20 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): 1. Login, 2. Dashboard, 3. Halaman Devices, 4. Halaman Asset Inventory, 5. Halaman Remote Desktop, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Cepat (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.38
Nodes (5): Conn, Mutex, Hub, pipeRelay(), relaySession

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.12
Nodes (17): Auto-Start (LaunchDaemon), Cara 1: Langsung Jalankan (Paling Cepat), Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File (+9 more)

### Community 10 - "Community 10"
Cohesion: 0.12
Nodes (15): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+7 more)

### Community 11 - "Community 11"
Cohesion: 0.12
Nodes (16): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell) (+8 more)

### Community 19 - "TestAgentVersionEndpoint"
Cohesion: 0.26
Nodes (11): NewDB(), TestAgentVersionEndpoint(), TestChangePasswordAndRateLimit(), TestDBAndBranchVerification(), TestMFAWorkflow(), TestTokenClaims(), generateMFATicket(), GenerateTOTPCode() (+3 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.23
Nodes (6): a(), b(), d(), g(), r(), s()

## Knowledge Gaps
- **100 isolated node(s):** `github.com/user/remote-desktop`, `Hub`, `contextKey`, `build.sh script`, `update-server.sh script` (+95 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `DB` connect `Community 2` to `Community 3`, `Community 1`, `TestAgentVersionEndpoint`?**
  _High betweenness centrality (0.102) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 2`, `Community 3`?**
  _High betweenness centrality (0.080) - this node is a cross-community bridge._
- **Why does `WSMessage` connect `Community 2` to `Community 0`?**
  _High betweenness centrality (0.055) - this node is a cross-community bridge._
- **What connects `github.com/user/remote-desktop`, `Hub`, `contextKey` to the rest of the system?**
  _100 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.11553030303030302 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.13309134906231096 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.07123034227567067 - nodes in this community are weakly interconnected._