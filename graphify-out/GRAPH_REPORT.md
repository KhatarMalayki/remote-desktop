# Graph Report - remote-desktop  (2026-09-30)

## Corpus Check
- 102 files · ~145,216 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1093 nodes · 2246 edges · 73 communities (63 shown, 10 thin omitted)
- Extraction: 88% EXTRACTED · 12% INFERRED · 0% AMBIGUOUS · INFERRED: 264 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `20385698`
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
- AssetSwitchRequest
- TestRemoteDestinationNoOverwrite
- .deployApplication
- .handleDeployments
- AssetSwitchRequest
- main
- Remote disconnect investigation — 2026-09-30

## God Nodes (most connected - your core abstractions)
1. `Server` - 73 edges
2. `DB` - 72 edges
3. `jsonError()` - 60 edges
4. `getClaims()` - 54 edges
5. `jsonResp()` - 54 edges
6. `api()` - 49 edges
7. `NewDB()` - 45 edges
8. `showToast()` - 42 edges
9. `Agent` - 24 edges
10. `NewHub()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `TestExecuteLockPolicyIdempotentAndSaveFailure()` --calls--> `New()`  [INFERRED]
  internal/agent/endpoint_test.go → internal/server/server.go
- `listProcesses()` --calls--> `New()`  [INFERRED]
  internal/agent/process_other.go → internal/server/server.go

## Import Cycles
- None detected.

## Communities (73 total, 10 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.05
Nodes (41): AgentConfig, SystemInfo, envOr(), main(), CleanupOldExecutable(), executeShellCommand(), generateDeviceID(), Agent (+33 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (40): FS, canAccessSwitch(), Request, ResponseWriter, Server, validatePassword(), Request, ResponseWriter (+32 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (3): DB, Time, SecuritySettings

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
Cohesion: 0.19
Nodes (17): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+9 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.12
Nodes (21): checkAuth(), closeChangePasswordModal(), closeMFAModal(), configureRustDesk(), copyDownloadAgentLink(), copyMFASecret(), doLogin(), doLoginMFA() (+13 more)

### Community 10 - "Community 10"
Cohesion: 0.20
Nodes (9): API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Error yang Mungkin Muncul, Pengelolaan MFA, PIC dan pemegang fisik aset (+1 more)

### Community 11 - "Community 11"
Cohesion: 0.09
Nodes (35): remoteDeskService, ChangeRequest, enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker(), startSecureDesktopRelay(), startUserDesktopRelay() (+27 more)

### Community 17 - "device.go"
Cohesion: 0.11
Nodes (14): ManageRustDesk(), RawMessage, Time, AssetActivity, AuthLog, BlockedIPInfo, Branch, DeviceHeartbeat (+6 more)

### Community 19 - "NewDB"
Cohesion: 0.09
Nodes (50): hashPassword(), NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuditLogFilters() (+42 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.18
Nodes (12): exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), remoteFileRequest(), renderAssets() (+4 more)

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
Nodes (26): closeDownloadAgentModal(), devices, doLogout(), loadHistoryLogs(), loadSecurityLogs(), manualAssets, openHistoryModal(), openRemoteTerminalFromSession() (+18 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.06
Nodes (26): remoteCommand, remoteFileTransfer, remoteScreenState, File, Hash, Image, captureRemoteFrame(), encodeRemoteFrame() (+18 more)

### Community 28 - "loadDevices"
Cohesion: 0.10
Nodes (27): closeDeviceModal(), closeManualAssetModal(), closeRustDeskManageModal(), closeVerificationModal(), connectWS(), deleteDevice(), deleteManualAsset(), filterDevices() (+19 more)

### Community 29 - "esc"
Cohesion: 0.36
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 30 - "renderCharts"
Cohesion: 0.07
Nodes (34): lockPolicyState, lockPolicyStore, lockValue, mockLockStore, windowsLockStore, executeLockPolicy(), Agent, collectEndpointReport() (+26 more)

### Community 31 - "init"
Cohesion: 0.33
Nodes (11): CHART_COLORS, chartDefaults(), openProcessList(), processIcon(), renderApplicationRuntimeChart(), renderApplicationsChart(), renderCharts(), renderDiskChart() (+3 more)

### Community 32 - "loadBranchAssets"
Cohesion: 0.26
Nodes (9): addTrackedApplicationRow(), onEndpointDevicePickerChange(), openEndpointDevice(), openPolicyManagerModal(), openTrackedApplications(), refreshEndpointDevice(), showEndpointModal(), submitLockPolicy() (+1 more)

### Community 33 - "main"
Cohesion: 0.16
Nodes (24): addBusinessUnit(), api(), buildBranchOptionsHTML(), closeBranchesModal(), createBranch(), deleteBranch(), editBranch(), esc() (+16 more)

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
Cohesion: 0.28
Nodes (9): buildDeviceTable(), isVersionNewer(), osIcon(), parseAgentVersion(), renderDevices(), renderRecentDevices(), timeAgo(), updateDeviceAgent() (+1 more)

### Community 46 - "Daftar API"
Cohesion: 0.27
Nodes (4): remotePrivacyWindow, Int64, remoteProtection, startRemotePrivacy()

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.10
Nodes (27): Request, ResponseWriter, Server, authRequest(), HandlerFunc, T, securityServer(), TestCredentialTokenIntegrityAndAccountChanges() (+19 more)

### Community 48 - "README.md"
Cohesion: 0.29
Nodes (5): assert, fs, path, test, vm

### Community 50 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.22
Nodes (9): Activity Log Device, Daftar API, Detail Satu Device, Hapus Device, List Semua Device, List Semua Group, Login, Statistik Dashboard (+1 more)

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
Cohesion: 0.27
Nodes (8): browseRemoteFiles(), deploymentDevices, loadDeployments(), openDeployments(), openFileManager(), renderDeployTargets(), submitDeployment(), supportsDeployment()

### Community 61 - "README.md"
Cohesion: 0.23
Nodes (6): a(), b(), d(), g(), r(), s()

### Community 62 - "startRemote"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 63 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.40
Nodes (4): Batas verifikasi, Deploy aplikasi, Deploy aplikasi dan transfer file — 0.2.56, Transfer file

### Community 64 - "TestRemoteInputProtectionCleanup"
Cohesion: 0.67
Nodes (3): T, TestRemoteInputProtectionCleanup(), TestRemoteProtectionDesktopTransitionReassertsBlockInput()

### Community 65 - "AssetSwitchRequest"
Cohesion: 0.22
Nodes (11): closeChangeUsernameModal(), closeEditUserModal(), createUser(), deleteUser(), getSelectValues(), loadUsers(), openUsersModal(), resetUserMFA() (+3 more)

### Community 68 - ".deployApplication"
Cohesion: 0.29
Nodes (5): applicationDeployment, deploymentExitStatus(), Agent, T, TestDeploymentExitStatus()

### Community 69 - ".handleDeployments"
Cohesion: 0.32
Nodes (4): RawMessage, Request, ResponseWriter, Server

### Community 70 - "AssetSwitchRequest"
Cohesion: 0.22
Nodes (3): DB, AssetAttachment, AssetSwitchRequest

### Community 72 - "Remote disconnect investigation — 2026-09-30"
Cohesion: 0.50
Nodes (3): Remote disconnect investigation — 2026-09-30, Verification and limits, Verified in source

## Knowledge Gaps
- **143 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `Agent`, `rustDeskExportConfig`, `RoleDefinition` (+138 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **10 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `Community 0`, `main`, `.deployApplication`, `.handleDeployments`, `main`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`, `NewDB`, `SecuritySettings`, `renderCharts`?**
  _High betweenness centrality (0.118) - this node is a cross-community bridge._
- **Why does `NewDB()` connect `NewDB` to `Community 1`, `Community 2`, `Community 7`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`, `.Close`, `esc`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 0`, `device.go`, `Community 3`, `NewDB`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Are the 12 inferred relationships involving `jsonError()` (e.g. with `.authorizeUserToken()` and `.handleActivities()`) actually correct?**
  _`jsonError()` has 12 INFERRED edges - model-reasoned connections that need verification._
- **Are the 10 inferred relationships involving `getClaims()` (e.g. with `.handleActivities()` and `.handleAttachment()`) actually correct?**
  _`getClaims()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `jsonResp()` (e.g. with `.handleActivities()` and `.handleDeployments()`) actually correct?**
  _`jsonResp()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `Agent` to the rest of the system?**
  _143 weakly-connected nodes found - possible documentation gaps or missing edges._