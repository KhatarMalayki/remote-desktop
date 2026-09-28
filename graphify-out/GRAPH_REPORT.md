# Graph Report - remote-desktop  (2026-09-28)

## Corpus Check
- 69 files · ~156,199 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 859 nodes · 1869 edges · 57 communities (49 shown, 8 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 217 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `00ff1bc4`
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
- Hitungan Memori: Kok Bisa 1000 Device di 1 GB?
- Pasang Server di VPS Linux (Tanpa Docker)
- 3. Halaman Devices
- authentication.md
- 4. Halaman Asset Inventory
- dialogs.test.cjs
- dialogs.js

## God Nodes (most connected - your core abstractions)
1. `Server` - 73 edges
2. `DB` - 72 edges
3. `jsonError()` - 57 edges
4. `getClaims()` - 51 edges
5. `jsonResp()` - 51 edges
6. `api()` - 48 edges
7. `NewDB()` - 40 edges
8. `showToast()` - 40 edges
9. `Agent` - 24 edges
10. `userAllowsBranch()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `TestValidRemoteSessionID()` --calls--> `validRemoteSessionID()`  [INFERRED]
  internal/agent/remote_test.go → internal/agent/remote.go
- `installRustDesk()` --calls--> `New()`  [INFERRED]
  internal/agent/rustdesk_manage_windows.go → internal/server/server.go

## Import Cycles
- None detected.

## Communities (57 total, 8 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (34): AgentConfig, SystemInfo, envOr(), main(), Int64, CleanupOldExecutable(), generateDeviceID(), Agent (+26 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (42): FS, canAccessSwitch(), Request, ResponseWriter, Server, validatePassword(), verifyPassword(), Request (+34 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (6): Conn, DB, RWMutex, SignalMessage, Client, Hub

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.12
Nodes (17): Auto-Start (LaunchDaemon), Cara 1: Langsung Jalankan (Paling Cepat), Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File (+9 more)

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): 1. Login, 2. Dashboard, 3. Halaman Devices, 4. Halaman Asset Inventory, 5. Halaman Remote Desktop, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Cepat (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.21
Nodes (15): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+7 more)

### Community 8 - "Community 8"
Cohesion: 0.12
Nodes (16): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server, ❌ Error "go: command not found", ❌ Info hardware salah/kosong (+8 more)

### Community 9 - "Community 9"
Cohesion: 0.10
Nodes (37): api(), closeChangePasswordModal(), closeDeviceModal(), closeManualAssetModal(), closeMFAModal(), closeRustDeskManageModal(), closeVerificationModal(), configureRustDesk() (+29 more)

### Community 10 - "Community 10"
Cohesion: 0.36
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 11 - "Community 11"
Cohesion: 0.20
Nodes (17): activeInputDesktopName(), Rectangle, handleRemoteInput(), isSecureInputDesktop(), mouseFlags(), prepareRemoteDesktop(), releaseRemoteInputs(), sendKeyboardInput() (+9 more)

### Community 17 - "device.go"
Cohesion: 0.14
Nodes (9): Time, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan, RoleDefinition, User (+1 more)

### Community 19 - "NewDB"
Cohesion: 0.11
Nodes (43): NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuditLogFilters(), TestAuthLogs() (+35 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.24
Nodes (5): a(), b(), d(), r(), s()

### Community 22 - "db.go"
Cohesion: 0.16
Nodes (15): closeHandoverModal(), closeRelocateModal(), closeServiceModal(), isAssetUser(), loadAssetActivities(), mayReviewSwitch(), openAssetTimeline(), openSwitchHistory() (+7 more)

### Community 23 - ".Close"
Cohesion: 0.18
Nodes (3): validBranchType(), splitBranches(), AssetVerification

### Community 25 - "main"
Cohesion: 0.21
Nodes (4): DB, AssetActivity, AssetAttachment, AssetSwitchRequest

### Community 26 - "validBranchType"
Cohesion: 0.07
Nodes (21): closeDownloadAgentModal(), devices, loadHistoryLogs(), loadRoles(), loadSecurityLogs(), manualAssets, openHistoryModal(), openRolesModal() (+13 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.10
Nodes (16): remoteCommand, remoteScreenState, Image, captureRemoteFrame(), encodeRemoteFrame(), encodeRemoteFrameProfile(), Agent, Duration (+8 more)

### Community 28 - "loadDevices"
Cohesion: 0.27
Nodes (11): remoteDeskService, ChangeRequest, enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker(), startSecureDesktopRelay(), superviseConsoleWorker() (+3 more)

### Community 29 - "esc"
Cohesion: 0.22
Nodes (9): Activity Log Device, Daftar API, Detail Satu Device, Hapus Device, List Semua Device, List Semua Group, Login, Statistik Dashboard (+1 more)

### Community 30 - "renderCharts"
Cohesion: 0.33
Nodes (6): checkAuth(), doLogin(), doLoginMFA(), doLogout(), openChangePasswordModal(), resetIdleTimer()

### Community 31 - "init"
Cohesion: 0.39
Nodes (8): CHART_COLORS, chartDefaults(), openProcessList(), renderCharts(), renderDiskChart(), renderMemoryChart(), renderOnlineOfflineChart(), renderOSChart()

### Community 32 - "loadBranchAssets"
Cohesion: 0.22
Nodes (10): exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), renderAssets(), renderBranchAssets() (+2 more)

### Community 33 - "main"
Cohesion: 0.20
Nodes (16): addBusinessUnit(), buildBranchOptionsHTML(), closeBranchesModal(), createBranch(), deleteBranch(), editBranch(), esc(), loadBranches() (+8 more)

### Community 34 - "SecuritySettings"
Cohesion: 0.22
Nodes (11): closeChangeUsernameModal(), closeEditUserModal(), createUser(), deleteUser(), getSelectValues(), loadUsers(), openUsersModal(), resetUserMFA() (+3 more)

### Community 35 - "main"
Cohesion: 0.14
Nodes (18): rustDeskExportConfig, rustDeskCLIConfig(), T, TestRustDeskCLIConfigDecodesExport(), TestRustDeskCLIConfigRejectsUnsafeValues(), ManageRustDesk(), findRustDeskBinary(), Duration (+10 more)

### Community 36 - "Cara Pasang Agent di Komputer"
Cohesion: 0.14
Nodes (14): Agent Connect → Register, Alur Kerja (Flow), Arsitektur & Kenapa Bisa Ringan, CPU Usage, Gambaran Besar, Heartbeat (Setiap 30 Detik), Hitungan Memori: Kok Bisa 1000 Device di 1 GB?, Jadi untuk 1000 device: (+6 more)

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

### Community 43 - "SecuritySettings"
Cohesion: 0.12
Nodes (22): buildDeviceTable(), connectWS(), filterDevices(), goPage(), handleSignal(), init(), isVersionNewer(), loadDevices() (+14 more)

### Community 46 - "Daftar API"
Cohesion: 0.20
Nodes (9): API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Error yang Mungkin Muncul, Pengelolaan MFA, PIC dan pemegang fisik aset (+1 more)

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.09
Nodes (31): envOr(), main(), listProcesses(), listProcesses(), Request, ResponseWriter, Server, hashPassword() (+23 more)

### Community 49 - "Pasang Server di VPS Linux (Tanpa Docker)"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 50 - "3. Halaman Devices"
Cohesion: 0.40
Nodes (5): ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Login gagal "invalid credentials", Masalah Server, ❌ Server tidak mau start, error "address already in use"

### Community 56 - "dialogs.test.cjs"
Cohesion: 0.20
Nodes (7): assert, createElement(), element(), fs, path, test, vm

### Community 57 - "dialogs.js"
Cohesion: 0.53
Nodes (5): appAlert(), appConfirm(), appDialog(), appDialogQueue, appPrompt()

## Knowledge Gaps
- **114 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig`, `RoleDefinition`, `credentialToken` (+109 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `NewDB()` connect `NewDB` to `Community 10`, `Community 2`, `Community 7`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`?**
  _High betweenness centrality (0.090) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `NewDB`, `device.go`, `Community 3`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`?**
  _High betweenness centrality (0.081) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 10`, `device.go`, `NewDB`, `4. Halaman Asset Inventory`, `.Close`, `ManualAsset`, `main`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `jsonError()` (e.g. with `.authorizeUserToken()` and `.handleActivities()`) actually correct?**
  _`jsonError()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig` to the rest of the system?**
  _114 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.062310949788263764 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.07368082368082368 - nodes in this community are weakly interconnected._