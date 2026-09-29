# Graph Report - remote-desktop  (2026-09-29)

## Corpus Check
- 80 files · ~76,635 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 963 nodes · 2026 edges · 65 communities (55 shown, 10 thin omitted)
- Extraction: 89% EXTRACTED · 11% INFERRED · 0% AMBIGUOUS · INFERRED: 231 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `559509ff`
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
- remote_desktop_other.go
- Daftar API
- Hitungan Memori: Kok Bisa 1000 Device di 1 GB?
- README.md
- AssetSwitchRequest
- Skenario: Server di VPS Linux, Agent di Komputer Windows
- authentication.md
- Hitungan Memori: Kok Bisa 1000 Device di 1 GB?
- dialogs.test.cjs
- dialogs.js
- main
- remoteProtection
- README.md
- startRemote
- Hitungan Memori: Kok Bisa 1000 Device di 1 GB?
- TestRemoteInputProtectionCleanup

## God Nodes (most connected - your core abstractions)
1. `Server` - 73 edges
2. `DB` - 72 edges
3. `jsonError()` - 57 edges
4. `getClaims()` - 51 edges
5. `jsonResp()` - 51 edges
6. `api()` - 49 edges
7. `showToast()` - 42 edges
8. `NewDB()` - 41 edges
9. `Agent` - 24 edges
10. `userAllowsBranch()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `listProcesses()` --calls--> `New()`  [INFERRED]
  internal/agent/process_other.go → internal/server/server.go
- `installRustDesk()` --calls--> `New()`  [INFERRED]
  internal/agent/rustdesk_manage_windows.go → internal/server/server.go

## Import Cycles
- None detected.

## Communities (65 total, 10 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (37): AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), executeShellCommand(), generateDeviceID(), Agent (+29 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (42): FS, canAccessSwitch(), Request, ResponseWriter, Server, validatePassword(), verifyPassword(), Request (+34 more)

### Community 3 - "Community 3"
Cohesion: 0.10
Nodes (11): Conn, DB, RWMutex, rustDeskBootstrapSetting(), IsNewer(), parse(), T, TestIsNewer() (+3 more)

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.17
Nodes (8): Auto-Start (LaunchDaemon), Cara Pasang Agent di Komputer, Deploy ke Banyak Komputer Sekaligus, FAQ Agent, Langsung Jalankan, Pasang di macOS, Penjelasan Parameter Agent, Yang Kamu Butuhkan Sebelum Mulai

### Community 6 - "Community 6"
Cohesion: 0.08
Nodes (25): 1. Login, 2. Dashboard, 3. Halaman Devices, 4. Halaman Asset Inventory, 5. Halaman Remote Desktop, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Cepat (+17 more)

### Community 7 - "Community 7"
Cohesion: 0.32
Nodes (11): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+3 more)

### Community 8 - "Community 8"
Cohesion: 0.17
Nodes (11): ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server, ❌ Error "go: command not found", ❌ Klik Connect tapi layar tidak muncul, Masalah Build, Masalah Docker, Masalah Remote Desktop (+3 more)

### Community 9 - "Community 9"
Cohesion: 0.09
Nodes (40): api(), checkAuth(), closeChangePasswordModal(), closeChangeUsernameModal(), closeEditUserModal(), closeMFAModal(), configureRustDesk(), copyDownloadAgentLink() (+32 more)

### Community 10 - "Community 10"
Cohesion: 0.11
Nodes (18): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+10 more)

### Community 11 - "Community 11"
Cohesion: 0.09
Nodes (35): remoteDeskService, ChangeRequest, enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker(), startSecureDesktopRelay(), startUserDesktopRelay() (+27 more)

### Community 17 - "device.go"
Cohesion: 0.11
Nodes (13): ManageRustDesk(), Time, AssetActivity, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan (+5 more)

### Community 19 - "NewDB"
Cohesion: 0.10
Nodes (47): NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuditLogFilters(), TestAuthLogs() (+39 more)

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
Cohesion: 0.16
Nodes (7): assert, fs, path, Socket, test, vm, { webcrypto }

### Community 26 - "validBranchType"
Cohesion: 0.06
Nodes (20): closeDownloadAgentModal(), devices, loadHistoryLogs(), loadRoles(), loadSecurityLogs(), manualAssets, openHistoryModal(), openRemoteTerminalFromSession() (+12 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.06
Nodes (26): remoteCommand, remoteFileTransfer, remoteScreenState, File, Hash, Image, captureRemoteFrame(), encodeRemoteFrame() (+18 more)

### Community 28 - "loadDevices"
Cohesion: 0.16
Nodes (18): closeDeviceModal(), closeRustDeskManageModal(), closeVerificationModal(), deleteDevice(), filterDevices(), goPage(), init(), loadBranchAssets() (+10 more)

### Community 29 - "esc"
Cohesion: 0.36
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 30 - "renderCharts"
Cohesion: 0.22
Nodes (3): DB, AssetAttachment, AssetSwitchRequest

### Community 31 - "init"
Cohesion: 0.29
Nodes (12): CHART_COLORS, chartDefaults(), loadStats(), openProcessList(), processIcon(), renderApplicationRuntimeChart(), renderApplicationsChart(), renderCharts() (+4 more)

### Community 32 - "loadBranchAssets"
Cohesion: 0.17
Nodes (13): deleteManualAsset(), exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), remoteFileRequest() (+5 more)

### Community 33 - "main"
Cohesion: 0.17
Nodes (18): addBusinessUnit(), buildBranchOptionsHTML(), closeBranchesModal(), closeManualAssetModal(), createBranch(), deleteBranch(), editBranch(), esc() (+10 more)

### Community 34 - "SecuritySettings"
Cohesion: 0.29
Nodes (6): Applied changes and limits, Confirmed defects in our implementation, Microsoft references, Native reproduction result, Remote web input investigation — 2026-09-29, Scope and evidence

### Community 35 - "main"
Cohesion: 0.18
Nodes (15): rustDeskExportConfig, rustDeskCLIConfig(), T, TestRustDeskCLIConfigDecodesExport(), TestRustDeskCLIConfigRejectsUnsafeValues(), findRustDeskBinary(), Duration, installRustDesk() (+7 more)

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
Cohesion: 0.16
Nodes (14): buildDeviceTable(), connectWS(), handleSignal(), isVersionNewer(), osIcon(), parseAgentVersion(), quickRemote(), renderDevices() (+6 more)

### Community 46 - "Daftar API"
Cohesion: 0.27
Nodes (4): remotePrivacyWindow, Int64, remoteProtection, startRemotePrivacy()

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.10
Nodes (28): envOr(), main(), Request, ResponseWriter, Server, hashPassword(), authRequest(), HandlerFunc (+20 more)

### Community 48 - "README.md"
Cohesion: 0.40
Nodes (5): Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File, Pasang di Linux

### Community 49 - "AssetSwitchRequest"
Cohesion: 0.40
Nodes (5): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ Info hardware salah/kosong, Masalah Agent

### Community 50 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 52 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.29
Nodes (6): Local input, Native references checked, Privacy / layar hitam, Remote session controls (0.2.52), Send file, Verification

### Community 56 - "dialogs.test.cjs"
Cohesion: 0.20
Nodes (7): assert, createElement(), element(), fs, path, test, vm

### Community 57 - "dialogs.js"
Cohesion: 0.53
Nodes (5): appAlert(), appConfirm(), appDialog(), appDialogQueue, appPrompt()

### Community 58 - "main"
Cohesion: 0.40
Nodes (5): ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Login gagal "invalid credentials", Masalah Server, ❌ Server tidak mau start, error "address already in use"

### Community 61 - "README.md"
Cohesion: 0.50
Nodes (4): Cara 1: Langsung Jalankan (Paling Cepat), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Pasang di Windows

### Community 62 - "startRemote"
Cohesion: 0.60
Nodes (5): renderRemoteProtection(), setRemoteControls(), setRemoteStatus(), startRemote(), stopRemote()

### Community 64 - "TestRemoteInputProtectionCleanup"
Cohesion: 0.67
Nodes (3): T, TestRemoteInputProtectionCleanup(), TestRemoteProtectionDesktopTransitionReassertsBlockInput()

## Knowledge Gaps
- **131 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig`, `RoleDefinition`, `credentialToken` (+126 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?` to `Community 0`, `Community 1`, `main`, `NewDB`, `SecuritySettings`?**
  _High betweenness centrality (0.099) - this node is a cross-community bridge._
- **Why does `NewDB()` connect `NewDB` to `Community 2`, `esc`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`?**
  _High betweenness centrality (0.081) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 0`, `device.go`, `Community 3`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`?**
  _High betweenness centrality (0.076) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `jsonError()` (e.g. with `.authorizeUserToken()` and `.handleActivities()`) actually correct?**
  _`jsonError()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `rustDeskExportConfig` to the rest of the system?**
  _131 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05888376856118792 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.07078565980167811 - nodes in this community are weakly interconnected._