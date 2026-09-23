# Graph Report - remote-desktop  (2026-09-23)

## Corpus Check
- 61 files · ~150,043 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 798 nodes · 1739 edges · 53 communities (47 shown, 6 thin omitted)
- Extraction: 89% EXTRACTED · 11% INFERRED · 0% AMBIGUOUS · INFERRED: 188 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c1348959`
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
- .Close
- ManualAsset
- main
- validBranchType
- SecuritySettings
- loadDevices
- esc
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
- Hitungan Memori: Kok Bisa 1000 Device di 1 GB?
- README.md
- Pasang Server di VPS Linux (Tanpa Docker)
- 3. Halaman Devices
- authentication.md

## God Nodes (most connected - your core abstractions)
1. `Server` - 70 edges
2. `DB` - 69 edges
3. `jsonError()` - 54 edges
4. `jsonResp()` - 49 edges
5. `getClaims()` - 48 edges
6. `api()` - 44 edges
7. `showToast()` - 37 edges
8. `NewDB()` - 34 edges
9. `Agent` - 24 edges
10. `userAllowsBranch()` - 22 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `ScanLocalNetwork()` --calls--> `getLocalIP()`  [INFERRED]
  internal/agent/network_scan.go → internal/agent/sysinfo.go
- `installRustDesk()` --calls--> `New()`  [INFERRED]
  internal/agent/rustdesk_manage_windows.go → internal/server/server.go

## Import Cycles
- None detected.

## Communities (53 total, 6 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.08
Nodes (25): AgentConfig, SystemInfo, envOr(), main(), Int64, CleanupOldExecutable(), generateDeviceID(), Agent (+17 more)

### Community 1 - "Community 1"
Cohesion: 0.08
Nodes (35): FS, canAccessSwitch(), Request, ResponseWriter, Server, validatePassword(), assetResponsibilityWarning(), branchesMatch() (+27 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (3): DB, Time, SecuritySettings

### Community 3 - "Community 3"
Cohesion: 0.10
Nodes (11): Conn, DB, RWMutex, rustDeskBootstrapSetting(), IsNewer(), parse(), T, TestIsNewer() (+3 more)

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (12): Auto-Start (LaunchDaemon), Cara 1: Langsung Jalankan (Paling Cepat), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara Pasang Agent di Komputer, Deploy ke Banyak Komputer Sekaligus, FAQ Agent, Langsung Jalankan (+4 more)

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
Cohesion: 0.15
Nodes (16): closeChangePasswordModal(), closeDownloadAgentModal(), closeMFAModal(), copyDownloadAgentLink(), copyMFASecret(), exportBranchAssetsCSV(), openReconfigureModal(), openRustDesk() (+8 more)

### Community 10 - "Community 10"
Cohesion: 0.31
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 11 - "Community 11"
Cohesion: 0.19
Nodes (15): activeInputDesktopName(), Rectangle, handleRemoteInput(), isSecureInputDesktop(), mouseFlags(), prepareRemoteDesktop(), releaseRemoteInputs(), sendKeyboardInput() (+7 more)

### Community 17 - "device.go"
Cohesion: 0.12
Nodes (15): arpNeighbors(), ScanLocalNetwork(), ManageRustDesk(), Time, IP, AuthLog, BlockedIPInfo, Branch (+7 more)

### Community 19 - "NewDB"
Cohesion: 0.06
Nodes (64): envOr(), main(), Request, ResponseWriter, Server, hashPassword(), authRequest(), HandlerFunc (+56 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.23
Nodes (6): a(), b(), d(), g(), r(), s()

### Community 22 - "db.go"
Cohesion: 0.16
Nodes (15): closeHandoverModal(), closeRelocateModal(), closeServiceModal(), isAssetUser(), loadAssetActivities(), mayReviewSwitch(), openAssetTimeline(), openSwitchHistory() (+7 more)

### Community 23 - ".Close"
Cohesion: 0.18
Nodes (3): validBranchType(), splitBranches(), AssetVerification

### Community 25 - "main"
Cohesion: 0.13
Nodes (5): DB, AssetActivity, AssetAttachment, AssetSwitchRequest, User

### Community 26 - "validBranchType"
Cohesion: 0.08
Nodes (25): CHART_COLORS, chartDefaults(), devices, loadHistoryLogs(), loadSecurityLogs(), manualAssets, openHistoryModal(), openSecurityLogsModal() (+17 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.10
Nodes (17): remoteCommand, remoteScreenState, Image, captureRemoteFrame(), encodeRemoteFrame(), encodeRemoteFrameProfile(), Agent, Duration (+9 more)

### Community 28 - "loadDevices"
Cohesion: 0.27
Nodes (11): remoteDeskService, ChangeRequest, enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker(), startSecureDesktopRelay(), superviseConsoleWorker() (+3 more)

### Community 29 - "esc"
Cohesion: 0.17
Nodes (13): closeDeviceModal(), closeManualAssetModal(), closeRustDeskManageModal(), closeVerificationModal(), deleteDevice(), deleteManualAsset(), loadBranchAssets(), loadGroups() (+5 more)

### Community 31 - "init"
Cohesion: 0.29
Nodes (7): checkAuth(), doLogin(), doLoginMFA(), doLogout(), openChangePasswordModal(), openMFAModal(), resetIdleTimer()

### Community 32 - "loadBranchAssets"
Cohesion: 0.16
Nodes (16): buildBranchOptionsHTML(), esc(), exportAssets(), fmtBytes(), loadRoles(), onSearchBranchAssets(), openDeviceModal(), openDownloadAgentModal() (+8 more)

### Community 33 - "main"
Cohesion: 0.24
Nodes (16): addBusinessUnit(), api(), closeBranchesModal(), configureRustDesk(), createBranch(), deleteBranch(), editBranch(), loadBranches() (+8 more)

### Community 34 - "SecuritySettings"
Cohesion: 0.22
Nodes (11): closeChangeUsernameModal(), closeEditUserModal(), createUser(), deleteUser(), getSelectValues(), loadUsers(), openUsersModal(), resetUserMFA() (+3 more)

### Community 35 - "main"
Cohesion: 0.18
Nodes (15): rustDeskExportConfig, rustDeskCLIConfig(), T, TestRustDeskCLIConfigDecodesExport(), TestRustDeskCLIConfigRejectsUnsafeValues(), findRustDeskBinary(), Duration, installRustDesk() (+7 more)

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
Cohesion: 0.20
Nodes (10): Cara Install RemoteDesk, Checklist Setelah Install, Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute, Langkah 3: Jalankan server (tes dulu), Langkah 4: Bikin Jalan Otomatis (Systemd), Pakai Caddy (Paling Gampang, Auto SSL), Pasang HTTPS (Biar Aman) (+2 more)

### Community 42 - "parseRustDeskID"
Cohesion: 0.29
Nodes (4): parseRustDeskID(), T, TestParseRustDeskID(), getRustDeskID()

### Community 43 - "SecuritySettings"
Cohesion: 0.13
Nodes (21): buildDeviceTable(), connectWS(), filterDevices(), goPage(), handleSignal(), init(), isVersionNewer(), loadDevices() (+13 more)

### Community 46 - "Daftar API"
Cohesion: 0.12
Nodes (15): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+7 more)

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.40
Nodes (5): Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File, Pasang di Linux

### Community 48 - "README.md"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 49 - "Pasang Server di VPS Linux (Tanpa Docker)"
Cohesion: 0.50
Nodes (4): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya

### Community 50 - "3. Halaman Devices"
Cohesion: 0.50
Nodes (4): CPU Usage, Hitungan Memori: Kok Bisa 1000 Device di 1 GB?, Jadi untuk 1000 device:, Setiap koneksi agent pakai berapa memori?

## Knowledge Gaps
- **106 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig`, `RoleDefinition`, `credentialToken` (+101 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewDB()` connect `NewDB` to `Community 10`, `Community 2`?**
  _High betweenness centrality (0.084) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `NewDB`, `device.go`, `Community 3`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 10`, `device.go`, `NewDB`, `.Close`, `ManualAsset`, `main`?**
  _High betweenness centrality (0.074) - this node is a cross-community bridge._
- **Are the 7 inferred relationships involving `jsonError()` (e.g. with `.authorizeUserToken()` and `.handleActivities()`) actually correct?**
  _`jsonError()` has 7 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig` to the rest of the system?**
  _106 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.08309178743961353 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.08415841584158416 - nodes in this community are weakly interconnected._