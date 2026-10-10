# Graph Report - remote-desktop  (2026-10-04)

## Corpus Check
- 130 files · ~200,414 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1331 nodes · 2646 edges · 107 communities (91 shown, 16 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 340 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d26e4c60`
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
- process_usage_windows.go
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
- Remote disconnect investigation — 2026-09-30
- IsNewer
- ui-smoke.cjs
- detectApplications
- .heartbeatLoop
- .handleLockPolicy
- devices.test.cjs
- client.go
- qrcode.min.js
- lockPolicyState
- .accountState
- SecuritySettings
- executeLockPolicy
- .Read
- key-rotation.test.cjs
- vpn.js
- detectApplications
- init
- arpNeighbors
- password_breach_test.go
- 3. Halaman Devices
- 5. Halaman Remote Desktop
- key_rotation_test.go
- TestUnlinkDeviceManualAsset
- GenerateTOTPCode
- AssetSwitchRequest
- Pasang di macOS
- Rotasi API key bertahap (v0.2.62)
- TestAPIKeyRotationIsExplicitAndPersistent
- TestLockPolicyEndpointAuthorizationAndQueue
- .handleLockPolicy
- main
- TestHolderManagementPermissions
- TestVPNPolicyRequiresAdminAndValidation
- Branch

## God Nodes (most connected - your core abstractions)
1. `Server` - 76 edges
2. `DB` - 72 edges
3. `jsonError()` - 63 edges
4. `getClaims()` - 56 edges
5. `jsonResp()` - 55 edges
6. `api()` - 51 edges
7. `NewDB()` - 47 edges
8. `showToast()` - 44 edges
9. `Agent` - 26 edges
10. `NewHub()` - 23 edges

## Surprising Connections (you probably didn't know these)
- `TestStoppedSupervisorTerminatesOwnedWorker()` --calls--> `Command`  [INFERRED]
  cmd/agent/system_service_stop_windows_test.go → internal/vpn/pilot.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `TestAgentKeyRecoveryAfterRejectedConnection()` --calls--> `LoadConfig()`  [INFERRED]
  internal/agent/key_rotation_test.go → internal/agent/client.go

## Import Cycles
- None detected.

## Communities (107 total, 16 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.14
Nodes (7): CleanupOldExecutable(), executeShellCommand(), Agent, Conn, Mutex, RWMutex, FormatBytes()

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (43): FS, addMacInstallers(), agentBinaryCandidates(), canAccessSwitch(), Request, ResponseWriter, Server, Request (+35 more)

### Community 3 - "Community 3"
Cohesion: 0.10
Nodes (11): Conn, DB, RWMutex, checkPwnedPassword(), Context, Request, ResponseWriter, Server (+3 more)

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.11
Nodes (14): Cara 1: Langsung Jalankan (Paling Cepat), Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File, Cara Pasang Agent di Komputer (+6 more)

### Community 6 - "Community 6"
Cohesion: 0.13
Nodes (15): 1. Login, 2. Dashboard, 4. Halaman Asset Inventory, 6. Logout, Apa Bedanya dengan Halaman Devices?, Cara Pakai RemoteDesk, Export ke CSV (Excel), Grafik (Tengah) (+7 more)

### Community 7 - "Community 7"
Cohesion: 0.19
Nodes (17): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+9 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.13
Nodes (20): checkAuth(), closeChangePasswordModal(), closeMFAModal(), configureRustDesk(), copyDownloadAgentLink(), copyMFASecret(), doLogin(), doLoginMFA() (+12 more)

### Community 10 - "Community 10"
Cohesion: 0.20
Nodes (9): API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Error yang Mungkin Muncul, Pengelolaan MFA, PIC dan pemegang fisik aset (+1 more)

### Community 11 - "Community 11"
Cohesion: 0.07
Nodes (38): getRemoteClipboard(), remoteDesktop, Rectangle, handleRemoteInput(), macInputPermission(), macSendKey(), newRemoteDesktop(), releaseRemoteInputs() (+30 more)

### Community 17 - "device.go"
Cohesion: 0.14
Nodes (9): ManageRustDesk(), RawMessage, AuthLog, DeviceHeartbeat, RoleDefinition, RustDeskCommand, RustDeskResult, SecuritySettings (+1 more)

### Community 19 - "NewDB"
Cohesion: 0.26
Nodes (19): NewDB(), T, TestADHBranchIsolation(), TestAgentPackageDownload(), TestAgentRegistrationDoesNotAutoUpdate(), TestAgentVersionEndpoint(), TestAuditLogFilters(), TestAuthLogs() (+11 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.17
Nodes (13): deleteManualAsset(), exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), remoteFileRequest() (+5 more)

### Community 22 - "db.go"
Cohesion: 0.16
Nodes (15): closeHandoverModal(), closeRelocateModal(), closeServiceModal(), isAssetUser(), loadAssetActivities(), mayReviewSwitch(), openAssetTimeline(), openSwitchHistory() (+7 more)

### Community 23 - ".Close"
Cohesion: 0.18
Nodes (3): validBranchType(), splitBranches(), AssetVerification

### Community 25 - "main"
Cohesion: 0.07
Nodes (14): assert, fs, path, setTimeout(), test, vm, assert, fs (+6 more)

### Community 26 - "validBranchType"
Cohesion: 0.06
Nodes (29): bindRemoteKeyboard(), closeDownloadAgentModal(), devices, doLogout(), loadHistoryLogs(), loadSecurityLogs(), manualAssets, openHistoryModal() (+21 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.06
Nodes (31): remoteCommand, remoteFileTransfer, remoteScreenState, File, Hash, Image, captureRemoteFrame(), pasteRemoteText() (+23 more)

### Community 28 - "loadDevices"
Cohesion: 0.20
Nodes (12): closeDeviceModal(), closeManualAssetModal(), closeRustDeskManageModal(), closeVerificationModal(), deleteDevice(), loadBranchAssets(), onBranchChange(), saveDeviceMeta() (+4 more)

### Community 29 - "esc"
Cohesion: 0.36
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 30 - "renderCharts"
Cohesion: 0.29
Nodes (9): collectEndpointReport(), runLockPolicy(), runLockPolicy(), ApplicationDetection, EndpointReport, EndpointState, InstalledApplication, LockPolicyRequest (+1 more)

### Community 31 - "init"
Cohesion: 0.33
Nodes (11): CHART_COLORS, chartDefaults(), openProcessList(), processIcon(), renderApplicationRuntimeChart(), renderApplicationsChart(), renderCharts(), renderDiskChart() (+3 more)

### Community 32 - "loadBranchAssets"
Cohesion: 0.26
Nodes (9): addTrackedApplicationRow(), onEndpointDevicePickerChange(), openEndpointDevice(), openPolicyManagerModal(), openTrackedApplications(), refreshEndpointDevice(), showEndpointModal(), submitLockPolicy() (+1 more)

### Community 33 - "main"
Cohesion: 0.11
Nodes (35): addBusinessUnit(), api(), buildBranchOptionsHTML(), closeBranchesModal(), closeChangeUsernameModal(), closeEditUserModal(), createBranch(), createUser() (+27 more)

### Community 34 - "SecuritySettings"
Cohesion: 0.29
Nodes (6): Applied changes and limits, Confirmed defects in our implementation, Microsoft references, Native reproduction result, Remote web input investigation — 2026-09-29, Scope and evidence

### Community 35 - "main"
Cohesion: 0.18
Nodes (15): rustDeskExportConfig, rustDeskCLIConfig(), T, TestRustDeskCLIConfigDecodesExport(), TestRustDeskCLIConfigRejectsUnsafeValues(), findRustDeskBinary(), Duration, Service (+7 more)

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
Cohesion: 0.32
Nodes (4): RawMessage, Request, ResponseWriter, Server

### Community 46 - "Daftar API"
Cohesion: 0.27
Nodes (4): remotePrivacyWindow, Int64, remoteProtection, startRemotePrivacy()

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.16
Nodes (24): hashPassword(), authRequest(), cleanPasswordFixture(), Context, HandlerFunc, T, securityServer(), TestCredentialTokenIntegrityAndAccountChanges() (+16 more)

### Community 48 - "README.md"
Cohesion: 0.29
Nodes (5): assert, fs, path, test, vm

### Community 49 - "AssetSwitchRequest"
Cohesion: 0.05
Nodes (50): vpnClient, vpnLease, HMENU, HWND, Agent, Mutex, Time, Config (+42 more)

### Community 50 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.12
Nodes (21): remoteDeskService, ChangeRequest, T, TestStoppedSupervisorTerminatesOwnedWorker(), enableTokenPrivilege(), exeToUTF16(), runSystemService(), startConsoleSystemWorker() (+13 more)

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

### Community 59 - "process_usage_windows.go"
Cohesion: 0.23
Nodes (14): SystemInfo, applicationUsage24Hours(), getRemoteClipboard(), CollectSystemInfo(), getAllIPs(), getCPUModel(), GetCPUUsage(), getDiskInfo() (+6 more)

### Community 61 - "README.md"
Cohesion: 0.23
Nodes (13): T, TestDeploymentAuthorizationTransferAndResult(), NewHub(), T, TestRegistrationDoesNotBootstrapRustDeskWhenAlreadyDetected(), TestRegistrationDoesNotBootstrapUnsupportedAgent(), TestRegistrationQueuesRustDeskBootstrap(), TestRegistrationRepairsRustDeskOnceAfterAgentUpgrade() (+5 more)

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
Cohesion: 0.13
Nodes (19): buildDeviceTable(), deviceResourceUsage(), filterDevices(), goPage(), isVersionNewer(), keyStatusLabel(), loadDevices(), openReconfigureModal() (+11 more)

### Community 68 - ".deployApplication"
Cohesion: 0.22
Nodes (9): Activity Log Device, Daftar API, Detail Satu Device, Hapus Device, List Semua Device, List Semua Group, Login, Statistik Dashboard (+1 more)

### Community 69 - ".handleDeployments"
Cohesion: 0.21
Nodes (7): Request, ResponseWriter, Server, keyID(), TestAPIKeyRotationAuthorizationAndValidation(), validAPIKey(), apiKeyState

### Community 70 - "AssetSwitchRequest"
Cohesion: 0.19
Nodes (6): Time, DB, AssetActivity, AssetAttachment, BlockedIPInfo, User

### Community 72 - "Remote disconnect investigation — 2026-09-30"
Cohesion: 0.50
Nodes (3): Remote disconnect investigation — 2026-09-30, Verification and limits, Verified in source

### Community 73 - "IsNewer"
Cohesion: 0.20
Nodes (8): RawMessage, Request, ResponseWriter, Server, IsNewer(), parse(), T, TestIsNewer()

### Community 74 - "ui-smoke.cjs"
Cohesion: 0.22
Nodes (8): assert, { createRequire }, fixture, fs, http, path, responses, root

### Community 75 - "detectApplications"
Cohesion: 0.60
Nodes (4): T, TestAgentBinaryPlatformIsolation(), TestMacAgentPackage(), TestMacInstallerShellSyntax()

### Community 76 - ".heartbeatLoop"
Cohesion: 0.26
Nodes (7): activeApplications(), enrichProcesses(), isSystemProcess(), processIcon(), listProcesses(), listProcesses(), ProcessInfo

### Community 77 - ".handleLockPolicy"
Cohesion: 0.29
Nodes (5): applicationDeployment, deploymentExitStatus(), Agent, T, TestDeploymentExitStatus()

### Community 78 - "devices.test.cjs"
Cohesion: 0.25
Nodes (6): assert, fs, path, source, test, vm

### Community 79 - "client.go"
Cohesion: 0.29
Nodes (9): AgentConfig, envOr(), main(), generateDeviceID(), LoadConfig(), NewAgent(), T, TestLoadConfigAcceptsUTF8BOM() (+1 more)

### Community 80 - "qrcode.min.js"
Cohesion: 0.23
Nodes (6): a(), b(), d(), g(), r(), s()

### Community 81 - "lockPolicyState"
Cohesion: 0.31
Nodes (4): lockPolicyState, lockValue, mockLockStore, windowsLockStore

### Community 82 - ".accountState"
Cohesion: 0.40
Nodes (4): Request, ResponseWriter, Server, UserClaims

### Community 83 - "SecuritySettings"
Cohesion: 0.36
Nodes (9): T, TestAdminEditUserAndMultiBranchADH(), TestBranchADHAndHeadOfficeSPVHolderWorkflow(), TestHolderCannotAccessUnrelatedRoutes(), TestInitialAssignmentAttachmentAndTimeline(), TestMasterRoleAndITSupportPermissions(), TestOwnershipAndSwitchWorkflow(), TestServiceRequestAndAssignedToWorkflow() (+1 more)

### Community 84 - "executeLockPolicy"
Cohesion: 0.42
Nodes (7): lockPolicyStore, executeLockPolicy(), T, TestExecuteLockPolicyApplyAndRestoreWithDrift(), TestExecuteLockPolicyAuditDoesNotMutate(), TestExecuteLockPolicyIdempotentAndSaveFailure(), TestExecuteLockPolicyRefusesWhenExternalManagerConflict()

### Community 85 - ".Read"
Cohesion: 0.42
Nodes (7): boundedEndpointText(), collectEndpointReport(), installedMachineApplications(), managementBlocker(), openMachineSCM(), readLockValue(), Mgr

### Community 86 - "key-rotation.test.cjs"
Cohesion: 0.22
Nodes (7): assert, fs, path, rotation, source, test, vm

### Community 87 - "vpn.js"
Cohesion: 0.36
Nodes (7): onVPNDevicePickerChange(), openVPNModal(), refreshVPNState(), renderVPNModal(), triggerVPN(), updateVPNDevicePicker(), vpnPilotState

### Community 88 - "detectApplications"
Cohesion: 0.32
Nodes (4): detectApplications(), DB, validTrackedApplications(), TrackedApplication

### Community 89 - "init"
Cohesion: 0.32
Nodes (8): connectWS(), handleSignal(), init(), loadGroups(), loadStats(), openRemoteFromLocation(), showPage(), updateRemoteDeviceList()

### Community 90 - "arpNeighbors"
Cohesion: 0.47
Nodes (5): arpNeighbors(), ScanLocalNetwork(), IP, NetworkScan, NetworkScanHost

### Community 91 - "password_breach_test.go"
Cohesion: 0.40
Nodes (4): Request, TestPwnedPasswordRangePrivacyAndFailures(), Response, breachTransport

### Community 92 - "3. Halaman Devices"
Cohesion: 0.40
Nodes (5): 3. Halaman Devices, Cari Device, Filter, Hapus Device, Lihat Detail Device

### Community 93 - "5. Halaman Remote Desktop"
Cohesion: 0.40
Nodes (5): 5. Halaman Remote Desktop, Cara Cepat, Cara Remote ke Komputer, Putus Koneksi, Yang Perlu Diketahui

### Community 94 - "key_rotation_test.go"
Cohesion: 0.60
Nodes (4): T, TestAgentConnectEncodesSpecialKeyAndDeviceID(), TestAgentKeyRecoveryAfterRejectedConnection(), TestAgentReconfigureKeepsRecoveryKey()

### Community 95 - "TestUnlinkDeviceManualAsset"
Cohesion: 0.60
Nodes (4): T, TestMatchManualAssetPriorityAndDuplicateWarning(), TestRelinkManualAssetOnHostnameChange(), TestUnlinkDeviceManualAsset()

### Community 96 - "GenerateTOTPCode"
Cohesion: 0.40
Nodes (3): GenerateTOTPCode(), Time, parseMFATicket()

### Community 98 - "Pasang di macOS"
Cohesion: 0.50
Nodes (4): Auto-Start (LaunchDaemon), Installer dari dashboard, Langsung Jalankan, Pasang di macOS

### Community 99 - "Rotasi API key bertahap (v0.2.62)"
Cohesion: 0.50
Nodes (3): Rotasi API key bertahap (v0.2.62), Upgrade dari implementasi lama, Verifikasi

### Community 100 - "TestAPIKeyRotationIsExplicitAndPersistent"
Cohesion: 0.67
Nodes (3): T, TestAPIKeyRotationIsExplicitAndPersistent(), TestOldKeyWebSocketDoesNotAutoMigrate()

### Community 101 - "TestLockPolicyEndpointAuthorizationAndQueue"
Cohesion: 0.67
Nodes (3): T, TestLockPolicyEndpointAuthorizationAndQueue(), TestTrackedApplicationsAPIAndDetection()

## Knowledge Gaps
- **173 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `Agent`, `rustDeskExportConfig`, `RoleDefinition` (+168 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **16 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 1` to `GenerateTOTPCode`, `main`, `Community 3`, `main`, `SecuritySettings`, `.heartbeatLoop`, `.handleLockPolicy`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`, `.accountState`, `NewDB`, `executeLockPolicy`, `.Read`, `SecuritySettings`, `README.md`?**
  _High betweenness centrality (0.115) - this node is a cross-community bridge._
- **Why does `NewDB()` connect `NewDB` to `Community 1`, `Community 2`, `TestAPIKeyRotationIsExplicitAndPersistent`, `TestLockPolicyEndpointAuthorizationAndQueue`, `Community 7`, `Hitungan Memori: Kok Bisa 1000 Device di 1 GB?`, `README.md`, `SecuritySettings`, `.Close`, `esc`, `TestUnlinkDeviceManualAsset`?**
  _High betweenness centrality (0.074) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 3`, `.handleDeployments`, `.heartbeatLoop`, `Skenario: Server di VPS Linux, Agent di Komputer Windows`, `arpNeighbors`, `README.md`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Are the 15 inferred relationships involving `jsonError()` (e.g. with `.acceptNewPassword()` and `.authorizeUserToken()`) actually correct?**
  _`jsonError()` has 15 INFERRED edges - model-reasoned connections that need verification._
- **Are the 12 inferred relationships involving `getClaims()` (e.g. with `.handleActivities()` and `.handleAttachment()`) actually correct?**
  _`getClaims()` has 12 INFERRED edges - model-reasoned connections that need verification._
- **Are the 11 inferred relationships involving `jsonResp()` (e.g. with `.handleActivities()` and `.handleDeployments()`) actually correct?**
  _`jsonResp()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `Agent` to the rest of the system?**
  _173 weakly-connected nodes found - possible documentation gaps or missing edges._