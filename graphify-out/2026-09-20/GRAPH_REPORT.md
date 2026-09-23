# Graph Report - remote-desktop  (2026-09-20)

## Corpus Check
- 42 files · ~96,174 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 609 nodes · 1274 edges · 40 communities (30 shown, 10 thin omitted)
- Extraction: 93% EXTRACTED · 7% INFERRED · 0% AMBIGUOUS · INFERRED: 85 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `aaa680da`
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
- loadDevices
- esc
- renderCharts
- init
- loadBranchAssets
- main
- SecuritySettings
- main
- validBranchType
- Cara Install RemoteDesk

## God Nodes (most connected - your core abstractions)
1. `DB` - 63 edges
2. `Server` - 62 edges
3. `jsonResp()` - 41 edges
4. `jsonError()` - 41 edges
5. `api()` - 41 edges
6. `getClaims()` - 40 edges
7. `showToast()` - 30 edges
8. `NewDB()` - 20 edges
9. `Agent` - 16 edges
10. `Hub` - 16 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `TestValidRemoteSessionID()` --calls--> `validRemoteSessionID()`  [INFERRED]
  internal/agent/remote_test.go → internal/agent/remote.go
- `New()` --calls--> `NewDB()`  [INFERRED]
  internal/server/server.go → internal/server/db.go

## Import Cycles
- None detected.

## Communities (40 total, 10 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.07
Nodes (33): AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), generateDeviceID(), Agent, Conn (+25 more)

### Community 1 - "Community 1"
Cohesion: 0.10
Nodes (27): FS, HandlerFunc, assetResponsibilityWarning(), canApproveSwitch(), deviceRecommendation(), generateToken(), getClaims(), RWMutex (+19 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (5): Conn, RWMutex, SignalMessage, Client, Hub

### Community 4 - "Community 4"
Cohesion: 0.13
Nodes (28): api(), closeChangePasswordModal(), closeDeviceModal(), closeManualAssetModal(), closeMFAModal(), closeVerificationModal(), copyMFASecret(), deleteDevice() (+20 more)

### Community 5 - "Community 5"
Cohesion: 0.12
Nodes (15): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+7 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): 1. Login, 2. Dashboard, 3. Halaman Devices, 4. Halaman Asset Inventory, 5. Halaman Remote Desktop, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Cepat (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.32
Nodes (11): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+3 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.18
Nodes (14): buildDeviceTable(), filterDevices(), goPage(), isVersionNewer(), loadDevices(), osIcon(), parseAgentVersion(), renderDevices() (+6 more)

### Community 10 - "Community 10"
Cohesion: 0.29
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 11 - "Community 11"
Cohesion: 0.12
Nodes (13): remoteCommand, Rectangle, handleRemoteInput(), Rectangle, handleRemoteInput(), mouseFlags(), prepareRemoteDesktop(), sendRemoteHotkey() (+5 more)

### Community 17 - "device.go"
Cohesion: 0.18
Nodes (8): Time, AssetVerification, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan, User

### Community 19 - "NewDB"
Cohesion: 0.12
Nodes (30): NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuthLogs(), TestBranchManagement() (+22 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.24
Nodes (5): a(), d(), g(), r(), s()

### Community 22 - "db.go"
Cohesion: 0.52
Nodes (6): isAssetUser(), mayReviewSwitch(), openSwitchHistory(), ownershipUI(), requestAssetSwitch(), reviewAssetSwitch()

### Community 26 - "validBranchType"
Cohesion: 0.08
Nodes (22): CHART_COLORS, chartDefaults(), closeDownloadAgentModal(), devices, loadSecurityLogs(), loadStats(), manualAssets, openSecurityLogsModal() (+14 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.18
Nodes (13): remoteScreenState, Image, captureRemoteFrame(), encodeRemoteFrame(), encodeRemoteFrameProfile(), Agent, Duration, Rectangle (+5 more)

### Community 28 - "loadDevices"
Cohesion: 0.39
Nodes (7): remoteDeskService, ChangeRequest, runSystemService(), startConsoleSystemWorker(), superviseConsoleWorker(), writeServiceDiagnostic(), Status

### Community 29 - "esc"
Cohesion: 0.22
Nodes (13): addBusinessUnit(), createBranch(), createUser(), deleteBranch(), editBranch(), loadBranches(), loadBranchesManagement(), loadBusinessUnits() (+5 more)

### Community 30 - "renderCharts"
Cohesion: 0.12
Nodes (16): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell) (+8 more)

### Community 31 - "init"
Cohesion: 0.28
Nodes (9): checkAuth(), connectWS(), doLogin(), doLoginMFA(), doLogout(), handleSignal(), init(), resetIdleTimer() (+1 more)

### Community 32 - "loadBranchAssets"
Cohesion: 0.21
Nodes (13): esc(), exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openHistoryModal(), renderAssets() (+5 more)

### Community 33 - "main"
Cohesion: 0.33
Nodes (6): closeChangeUsernameModal(), deleteUser(), loadUsers(), openUsersModal(), resetUserMFA(), submitChangeUsername()

### Community 34 - "SecuritySettings"
Cohesion: 0.04
Nodes (45): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya, Cara Install RemoteDesk, Checklist Setelah Install, Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute (+37 more)

## Knowledge Gaps
- **101 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `contextKey`, `build.sh script`, `update-server.sh script` (+96 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Community 1` to `device.go`, `Community 2`, `Community 3`?**
  _High betweenness centrality (0.162) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 1`, `Community 3`, `validBranchType`, `Cara Install RemoteDesk`, `Community 10`, `device.go`, `NewDB`, `.Close`, `ManualAsset`, `main`?**
  _High betweenness centrality (0.098) - this node is a cross-community bridge._
- **Why does `IsNewer()` connect `Community 0` to `Community 1`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **What connects `github.com/user/remote-desktop`, `Agent`, `contextKey` to the rest of the system?**
  _101 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.06980392156862746 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.10443037974683544 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06881720430107527 - nodes in this community are weakly interconnected._