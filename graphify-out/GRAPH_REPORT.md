# Graph Report - remote-desktop  (2026-09-13)

## Corpus Check
- 25 files · ~20,958 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 361 nodes · 657 edges · 20 communities (15 shown, 5 thin omitted)
- Extraction: 97% EXTRACTED · 3% INFERRED · 0% AMBIGUOUS · INFERRED: 19 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

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

## God Nodes (most connected - your core abstractions)
1. `Server` - 33 edges
2. `DB` - 32 edges
3. `jsonResp()` - 21 edges
4. `jsonError()` - 20 edges
5. `api()` - 19 edges
6. `Agent` - 15 edges
7. `getClaims()` - 15 edges
8. `Hub` - 14 edges
9. `loadDevices()` - 13 edges
10. `Client` - 12 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `TestAgentVersionEndpoint()` --calls--> `NewDB()`  [INFERRED]
  internal/server/db_test.go → internal/server/db.go
- `TestDBAndBranchVerification()` --calls--> `NewDB()`  [INFERRED]
  internal/server/db_test.go → internal/server/db.go

## Import Cycles
- None detected.

## Communities (20 total, 5 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (20): Agent, AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), generateDeviceID(), Conn (+12 more)

### Community 1 - "Community 1"
Cohesion: 0.18
Nodes (16): FS, HandlerFunc, getClaims(), Hub, hashPassword(), jsonError(), jsonResp(), New() (+8 more)

### Community 2 - "Community 2"
Cohesion: 0.08
Nodes (15): migrate(), NewDB(), scanDevice(), scanDeviceRows(), scanManualAsset(), AssetVerification, Device, DeviceHeartbeat (+7 more)

### Community 3 - "Community 3"
Cohesion: 0.14
Nodes (7): Conn, Hub, Hub, NewHub(), SignalMessage, RWMutex, Client

### Community 4 - "Community 4"
Cohesion: 0.08
Nodes (63): api(), buildDeviceTable(), CHART_COLORS, chartDefaults(), checkAuth(), closeDeviceModal(), closeManualAssetModal(), closeVerificationModal() (+55 more)

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
Cohesion: 0.47
Nodes (5): TestAgentVersionEndpoint(), TestDBAndBranchVerification(), TestTokenClaims(), generateToken(), T

## Knowledge Gaps
- **100 isolated node(s):** `github.com/user/remote-desktop`, `Hub`, `contextKey`, `build.sh script`, `update-server.sh script` (+95 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `DB` connect `Community 2` to `Community 1`, `Community 3`?**
  _High betweenness centrality (0.093) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 2`, `Community 3`?**
  _High betweenness centrality (0.070) - this node is a cross-community bridge._
- **Why does `WSMessage` connect `Community 2` to `Community 0`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **What connects `github.com/user/remote-desktop`, `Hub`, `contextKey` to the rest of the system?**
  _100 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.11553030303030302 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.08305647840531562 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.14285714285714285 - nodes in this community are weakly interconnected._