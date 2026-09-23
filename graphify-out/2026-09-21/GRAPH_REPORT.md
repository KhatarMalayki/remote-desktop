# Graph Report - remote-desktop  (2026-09-21)

## Corpus Check
- 57 files · ~104,028 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 701 nodes · 1447 edges · 56 communities (46 shown, 10 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 114 edges (avg confidence: 0.78)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `0664d5f8`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Community 0
- Community 1
- Community 2
- Community 3
- RemoteDesk
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
- Cara Pasang Agent di Komputer
- Cara Install RemoteDesk
- single_instance_windows.go
- main
- parseRustDeskID
- SecuritySettings
- Daftar API
- API Reference
- Pasang Server di VPS Linux (Tanpa Docker)
- 3. Halaman Devices
- 5. Halaman Remote Desktop
- 2. Dashboard
- validBranchType
- .TrustDevice

## God Nodes (most connected - your core abstractions)
1. `Server` - 67 edges
2. `DB` - 66 edges
3. `jsonResp()` - 43 edges
4. `jsonError()` - 43 edges
5. `api()` - 43 edges
6. `getClaims()` - 42 edges
7. `showToast()` - 34 edges
8. `NewDB()` - 27 edges
9. `Agent` - 24 edges
10. `Hub` - 16 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `installRustDesk()` --calls--> `New()`  [INFERRED]
  internal/agent/rustdesk_manage_windows.go → internal/server/server.go
- `TestRelayRejectsUnauthorizedConnectionsBeforeUpgrade()` --calls--> `NewDB()`  [INFERRED]
  internal/server/relay_test.go → internal/server/db.go

## Import Cycles
- None detected.

## Communities (56 total, 10 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (32): AgentConfig, SystemInfo, envOr(), main(), Int64, CleanupOldExecutable(), generateDeviceID(), Agent (+24 more)

### Community 1 - "Community 1"
Cohesion: 0.08
Nodes (34): envOr(), main(), FS, HandlerFunc, assetResponsibilityWarning(), canApproveSwitch(), deviceRecommendation(), generateToken() (+26 more)

### Community 3 - "Community 3"
Cohesion: 0.16
Nodes (5): Conn, RWMutex, SignalMessage, Client, Hub

### Community 4 - "RemoteDesk"
Cohesion: 0.29
Nodes (7): Apa Itu RemoteDesk?, Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana?, License, RemoteDesk, Tech Stack

### Community 5 - "Community 5"
Cohesion: 0.25
Nodes (8): Auto-Start (LaunchDaemon), Cara Pasang Agent di Komputer, Deploy ke Banyak Komputer Sekaligus, FAQ Agent, Langsung Jalankan, Pasang di macOS, Penjelasan Parameter Agent, Yang Kamu Butuhkan Sebelum Mulai

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): 1. Login, 2. Dashboard, 3. Halaman Devices, 4. Halaman Asset Inventory, 5. Halaman Remote Desktop, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Cepat (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.19
Nodes (16): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+8 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.24
Nodes (10): buildDeviceTable(), isVersionNewer(), osIcon(), parseAgentVersion(), renderDevices(), renderRecentDevices(), timeAgo(), updateDeviceAgent() (+2 more)

### Community 10 - "Community 10"
Cohesion: 0.29
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 11 - "Community 11"
Cohesion: 0.19
Nodes (15): activeInputDesktopName(), Rectangle, handleRemoteInput(), isSecureInputDesktop(), mouseFlags(), prepareRemoteDesktop(), releaseRemoteInputs(), sendKeyboardInput() (+7 more)

### Community 17 - "device.go"
Cohesion: 0.22
Nodes (9): Time, AssetVerification, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan, User (+1 more)

### Community 19 - "NewDB"
Cohesion: 0.16
Nodes (31): NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuthLogs(), TestBranchManagement() (+23 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.24
Nodes (5): a(), d(), g(), r(), s()

### Community 22 - "db.go"
Cohesion: 0.52
Nodes (6): isAssetUser(), mayReviewSwitch(), openSwitchHistory(), ownershipUI(), requestAssetSwitch(), reviewAssetSwitch()

### Community 26 - "validBranchType"
Cohesion: 0.08
Nodes (20): CHART_COLORS, chartDefaults(), closeDownloadAgentModal(), devices, loadStats(), manualAssets, pendingAgentUpdates, quickRemote() (+12 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.10
Nodes (17): remoteCommand, remoteScreenState, Image, captureRemoteFrame(), encodeRemoteFrame(), encodeRemoteFrameProfile(), Agent, Duration (+9 more)

### Community 28 - "loadDevices"
Cohesion: 0.27
Nodes (11): remoteDeskService, ChangeRequest, enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker(), startSecureDesktopRelay(), superviseConsoleWorker() (+3 more)

### Community 29 - "esc"
Cohesion: 0.15
Nodes (15): closeChangePasswordModal(), closeMFAModal(), configureRustDesk(), copyMFASecret(), exportBranchAssetsCSV(), openReconfigureModal(), openRustDesk(), openSelectedRustDesk() (+7 more)

### Community 30 - "renderCharts"
Cohesion: 0.29
Nodes (7): closeManualAssetModal(), closeVerificationModal(), deleteManualAsset(), loadBranchAssets(), onBranchChange(), saveManualAsset(), submitVerification()

### Community 31 - "init"
Cohesion: 0.22
Nodes (11): checkAuth(), closeChangeUsernameModal(), connectWS(), doLogin(), doLoginMFA(), doLogout(), handleSignal(), init() (+3 more)

### Community 32 - "loadBranchAssets"
Cohesion: 0.29
Nodes (8): exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), renderAssets(), renderBranchAssets(), setAssetTab()

### Community 33 - "main"
Cohesion: 0.15
Nodes (28): addBusinessUnit(), api(), createBranch(), createUser(), deleteBranch(), deleteUser(), editBranch(), esc() (+20 more)

### Community 34 - "SecuritySettings"
Cohesion: 0.20
Nodes (12): closeDeviceModal(), closeRustDeskManageModal(), deleteDevice(), filterDevices(), goPage(), loadDevices(), loadGroups(), renderPagination() (+4 more)

### Community 35 - "main"
Cohesion: 0.14
Nodes (18): rustDeskExportConfig, rustDeskCLIConfig(), T, TestRustDeskCLIConfigDecodesExport(), TestRustDeskCLIConfigRejectsUnsafeValues(), ManageRustDesk(), findRustDeskBinary(), Duration (+10 more)

### Community 36 - "Cara Pasang Agent di Komputer"
Cohesion: 0.20
Nodes (10): Agent Connect → Register, Alur Kerja (Flow), Arsitektur & Kenapa Bisa Ringan, Gambaran Besar, Heartbeat (Setiap 30 Detik), Kenapa Pakai Go?, Kenapa Pakai SQLite (Bukan MySQL/PostgreSQL)?, Limitasi (Yang Tidak Bisa) (+2 more)

### Community 37 - "Cara Install RemoteDesk"
Cohesion: 0.40
Nodes (4): acquireNamedWorkerLock(), acquireSystemWorkerLock(), T, TestSystemWorkerLockRejectsDuplicate()

### Community 39 - "single_instance_windows.go"
Cohesion: 0.40
Nodes (4): scheduleServiceManagedUpdate(), serviceUpdateScript(), T, TestServiceUpdateHelperStopsServiceBeforeReplacingBinary()

### Community 40 - "main"
Cohesion: 0.14
Nodes (14): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya, Cara Install RemoteDesk, Checklist Setelah Install, Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute (+6 more)

### Community 42 - "parseRustDeskID"
Cohesion: 0.29
Nodes (4): parseRustDeskID(), T, TestParseRustDeskID(), getRustDeskID()

### Community 46 - "Daftar API"
Cohesion: 0.12
Nodes (15): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+7 more)

### Community 47 - "API Reference"
Cohesion: 0.40
Nodes (5): Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File, Pasang di Linux

### Community 49 - "Pasang Server di VPS Linux (Tanpa Docker)"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 50 - "3. Halaman Devices"
Cohesion: 0.50
Nodes (4): Cara 1: Langsung Jalankan (Paling Cepat), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Pasang di Windows

### Community 51 - "5. Halaman Remote Desktop"
Cohesion: 0.50
Nodes (4): CPU Usage, Hitungan Memori: Kok Bisa 1000 Device di 1 GB?, Jadi untuk 1000 device:, Setiap koneksi agent pakai berapa memori?

### Community 52 - "2. Dashboard"
Cohesion: 0.50
Nodes (4): Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell)

## Knowledge Gaps
- **103 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig`, `contextKey`, `build.sh script` (+98 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Community 1` to `device.go`, `Community 2`, `Community 3`?**
  _High betweenness centrality (0.113) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 1`, `Community 3`, `Community 10`, `SecuritySettings`, `device.go`, `NewDB`, `.Close`, `validBranchType`, `ManualAsset`, `main`, `.TrustDevice`?**
  _High betweenness centrality (0.091) - this node is a cross-community bridge._
- **Why does `validRemoteSessionID()` connect `SecuritySettings` to `Community 0`?**
  _High betweenness centrality (0.063) - this node is a cross-community bridge._
- **What connects `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig` to the rest of the system?**
  _103 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.06363636363636363 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.0812989921612542 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06666666666666667 - nodes in this community are weakly interconnected._