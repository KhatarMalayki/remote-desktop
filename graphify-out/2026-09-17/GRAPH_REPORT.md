# Graph Report - remote-desktop  (2026-09-17)

## Corpus Check
- 30 files · ~41,189 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 519 nodes · 1111 edges · 28 communities (19 shown, 9 thin omitted)
- Extraction: 95% EXTRACTED · 5% INFERRED · 0% AMBIGUOUS · INFERRED: 59 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `48b66e63`
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
- device.go
- update-server.sh
- NewDB
- qrcode.min.js
- db.go
- ManualAsset
- main
- validBranchType
- SecuritySettings

## God Nodes (most connected - your core abstractions)
1. `DB` - 63 edges
2. `Server` - 60 edges
3. `jsonResp()` - 40 edges
4. `jsonError()` - 40 edges
5. `getClaims()` - 39 edges
6. `api()` - 39 edges
7. `showToast()` - 26 edges
8. `NewDB()` - 17 edges
9. `esc()` - 16 edges
10. `Agent` - 15 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `TestTokenClaims()` --calls--> `generateToken()`  [INFERRED]
  internal/server/db_test.go → internal/server/server.go
- `TestAgentVersionEndpoint()` --calls--> `NewHub()`  [INFERRED]
  internal/server/db_test.go → internal/server/hub.go

## Import Cycles
- None detected.

## Communities (28 total, 9 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.10
Nodes (24): Agent, AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), generateDeviceID(), Conn (+16 more)

### Community 1 - "Community 1"
Cohesion: 0.14
Nodes (14): FS, HandlerFunc, deviceRecommendation(), generateToken(), getClaims(), Hub, RWMutex, jsonError() (+6 more)

### Community 3 - "Community 3"
Cohesion: 0.13
Nodes (7): Conn, Hub, RWMutex, Hub, NewHub(), SignalMessage, Client

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (97): addBusinessUnit(), api(), buildDeviceTable(), CHART_COLORS, chartDefaults(), checkAuth(), closeChangePasswordModal(), closeChangeUsernameModal() (+89 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (43): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya, Cara Install RemoteDesk, Checklist Setelah Install, Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute (+35 more)

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
Cohesion: 0.29
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 11 - "Community 11"
Cohesion: 0.12
Nodes (16): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell) (+8 more)

### Community 17 - "device.go"
Cohesion: 0.22
Nodes (9): Time, AssetVerification, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan, User (+1 more)

### Community 19 - "NewDB"
Cohesion: 0.09
Nodes (38): envOr(), main(), NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentVersionEndpoint(), TestAuthLogs() (+30 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.24
Nodes (5): a(), d(), g(), r(), s()

### Community 22 - "db.go"
Cohesion: 0.52
Nodes (6): isAssetUser(), mayReviewSwitch(), openSwitchHistory(), ownershipUI(), requestAssetSwitch(), reviewAssetSwitch()

## Knowledge Gaps
- **100 isolated node(s):** `github.com/user/remote-desktop`, `Hub`, `contextKey`, `build.sh script`, `update-server.sh script` (+95 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Community 1` to `Community 3`, `device.go`, `Community 2`, `NewDB`?**
  _High betweenness centrality (0.113) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 1`, `Community 3`, `Community 10`, `device.go`, `NewDB`, `.Close`, `ManualAsset`, `main`, `validBranchType`, `SecuritySettings`?**
  _High betweenness centrality (0.105) - this node is a cross-community bridge._
- **Why does `NetworkScan` connect `device.go` to `Community 0`, `Community 1`?**
  _High betweenness centrality (0.044) - this node is a cross-community bridge._
- **What connects `github.com/user/remote-desktop`, `Hub`, `contextKey` to the rest of the system?**
  _100 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.09957325746799431 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.1408090117767537 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06896551724137931 - nodes in this community are weakly interconnected._