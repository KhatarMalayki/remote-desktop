# Graph Report - remote-desktop  (2026-10-10)

## Corpus Check
- 159 files · ~302,772 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1551 nodes · 3053 edges · 117 communities (100 shown, 17 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 430 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9451a525`
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
- qrcode.min.js
- Remote disconnect investigation — 2026-09-30
- lockPolicyState
- ui-smoke.cjs
- detectApplications
- .heartbeatLoop
- .handleLockPolicy
- devices.test.cjs
- client.go
- qrcode.min.js
- executeLockPolicy
- Device
- key-rotation.test.cjs
- vpn.js
- Skenario: Server di VPS Linux, Agent di Komputer Windows
- vpn.test.cjs
- .deployApplication
- .handleDeployments
- .Read
- detectApplications
- hashPassword
- .GetAssetVerificationsFiltered
- README.md
- SecuritySettings
- .acceptNewPassword
- Rotasi API key bertahap (v0.2.62)
- arpNeighbors
- Daftar API
- TestUnlinkDeviceManualAsset
- New
- verifyPassword
- .acceptNewPassword
- Pasang Server di VPS Linux (Tanpa Docker)
- 3. Halaman Devices
- 4. Halaman Asset Inventory
- Tips
- vpn-tray.md
- Pasang di Linux
- TestDeviceIdentityMigrationAndReopen
- TestHolderManagementPermissions
- SecuritySettings

## God Nodes (most connected - your core abstractions)
1. `Server` - 76 edges
2. `DB` - 72 edges
3. `jsonError()` - 64 edges
4. `getClaims()` - 57 edges
5. `jsonResp()` - 56 edges
6. `api()` - 51 edges
7. `NewDB()` - 50 edges
8. `showToast()` - 45 edges
9. `NewHub()` - 31 edges
10. `Agent` - 26 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewAgent()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `TestStoppedSupervisorTerminatesOwnedWorker()` --calls--> `Command`  [INFERRED]
  cmd/agent/system_service_stop_windows_test.go → internal/vpn/pilot.go
- `main()` --calls--> `LoadConfig()`  [INFERRED]
  cmd/agent/main.go → internal/agent/client.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/server/main.go → internal/server/server.go
- `TestNewAgentPersistsGeneratedDeviceID()` --calls--> `NewAgent()`  [INFERRED]
  internal/agent/client_test.go → internal/agent/client.go

## Import Cycles
- None detected.

## Communities (117 total, 17 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.13
Nodes (10): AgentConfig, CleanupOldExecutable(), executeShellCommand(), generateDeviceID(), Agent, Conn, Mutex, RWMutex (+2 more)

### Community 1 - "Community 1"
Cohesion: 0.06
Nodes (50): FS, addMacInstallers(), agentBinaryCandidates(), canAccessSwitch(), Request, ResponseWriter, DB, Server (+42 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (6): Conn, DB, RWMutex, SignalMessage, Client, Hub

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.15
Nodes (13): Auto-Start (LaunchDaemon), Cara 1: Langsung Jalankan (Paling Cepat), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara Pasang Agent di Komputer, Deploy ke Banyak Komputer Sekaligus, FAQ Agent, Installer dari dashboard (+5 more)

### Community 6 - "Community 6"
Cohesion: 0.12
Nodes (16): 1. Login, 2. Dashboard, 5. Halaman Remote Desktop, 6. Logout, Cara Cepat, Cara Pakai RemoteDesk, Cara Remote ke Komputer, Grafik (Tengah) (+8 more)

### Community 7 - "Community 7"
Cohesion: 0.20
Nodes (16): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+8 more)

### Community 8 - "Community 8"
Cohesion: 0.09
Nodes (21): ❌ Agent connect tapi device tidak muncul di dashboard, ❌ Agent sering putus dan reconnect, ❌ Agent tidak mau connect, "connection failed", ❌ API return "unauthorized" (401), ❌ Buka browser tapi halaman tidak muncul, ❌ Container tidak start, ❌ Data hilang setelah restart container, ❌ Error "gcc not found" waktu build server (+13 more)

### Community 9 - "Community 9"
Cohesion: 0.10
Nodes (26): checkAuth(), closeChangePasswordModal(), closeChangeUsernameModal(), closeMFAModal(), configureRustDesk(), copyDownloadAgentLink(), copyMFASecret(), doLogin() (+18 more)

### Community 10 - "Community 10"
Cohesion: 0.11
Nodes (18): Activity Log Device, API Reference, Cara 1: Pakai Token dari Login, Cara 2: Pakai API Key Langsung, Cara Autentikasi, Contoh Script: Monitoring Otomatis, Daftar API, Detail Satu Device (+10 more)

### Community 11 - "Community 11"
Cohesion: 0.21
Nodes (9): remoteFileTransfer, Hash, File, Time, T, TestRemoteFileNames(), TestRemoteFileRejectsInvalidTransfer(), TestRemoteFileTransfer() (+1 more)

### Community 17 - "device.go"
Cohesion: 0.12
Nodes (13): ManageRustDesk(), RawMessage, Time, AssetActivity, BlockedIPInfo, Branch, DeviceHeartbeat, NetworkScan (+5 more)

### Community 19 - "NewDB"
Cohesion: 0.07
Nodes (59): T, TestAPIKeyRotationIsExplicitAndPersistent(), TestBulkKeyMigrationRequiresPilotAndSkipsIneligible(), TestOldKeyWebSocketDoesNotAutoMigrate(), TestRetainShortLegacyKeyWithoutWeakeningNewKeys(), NewDB(), T, TestADHBranchIsolation() (+51 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.05
Nodes (45): remoteCommand, getRemoteClipboard(), remoteDesktop, Rectangle, handleRemoteInput(), macInputPermission(), macSendKey(), newRemoteDesktop() (+37 more)

### Community 22 - "db.go"
Cohesion: 0.16
Nodes (15): closeHandoverModal(), closeRelocateModal(), closeServiceModal(), isAssetUser(), loadAssetActivities(), mayReviewSwitch(), openAssetTimeline(), openSwitchHistory() (+7 more)

### Community 23 - ".Close"
Cohesion: 0.17
Nodes (3): validBranchType(), splitBranches(), AssetVerification

### Community 25 - "main"
Cohesion: 0.07
Nodes (14): assert, fs, path, setTimeout(), test, vm, assert, fs (+6 more)

### Community 26 - "validBranchType"
Cohesion: 0.06
Nodes (27): bindRemoteKeyboard(), closeDownloadAgentModal(), devices, loadHistoryLogs(), loadSecurityLogs(), manualAssets, openHistoryModal(), openRemoteTerminalFromSession() (+19 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.11
Nodes (18): exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), remoteFileRequest(), renderAssets() (+10 more)

### Community 28 - "loadDevices"
Cohesion: 0.14
Nodes (16): buildDeviceTable(), connectWS(), deviceResourceUsage(), deviceVPNBadge(), handleSignal(), isVersionNewer(), keyStatusLabel(), openReconfigureModal() (+8 more)

### Community 29 - "esc"
Cohesion: 0.05
Nodes (65): vpnClient, vpnLANInterface, vpnLANState, vpnLease, Mutex, Time, Config, T (+57 more)

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
Cohesion: 0.12
Nodes (32): addBusinessUnit(), api(), buildBranchOptionsHTML(), closeBranchesModal(), closeEditUserModal(), createBranch(), createUser(), deleteBranch() (+24 more)

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
Cohesion: 0.31
Nodes (14): hashPassword(), authRequest(), cleanPasswordFixture(), Context, HandlerFunc, Server, T, securityServer() (+6 more)

### Community 46 - "Daftar API"
Cohesion: 0.21
Nodes (7): Request, ResponseWriter, Server, keyID(), TestAPIKeyRotationAuthorizationAndValidation(), validAPIKey(), apiKeyState

### Community 47 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.31
Nodes (4): lockPolicyState, lockValue, mockLockStore, windowsLockStore

### Community 48 - "README.md"
Cohesion: 0.29
Nodes (5): assert, fs, path, test, vm

### Community 49 - "AssetSwitchRequest"
Cohesion: 0.42
Nodes (7): lockPolicyStore, executeLockPolicy(), T, TestExecuteLockPolicyApplyAndRestoreWithDrift(), TestExecuteLockPolicyAuditDoesNotMutate(), TestExecuteLockPolicyIdempotentAndSaveFailure(), TestExecuteLockPolicyRefusesWhenExternalManagerConflict()

### Community 50 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.09
Nodes (27): HMENU, HWND, Agent, createVPNSelfPipe(), Agent, Duration, Handle, readVPNSelfPipe() (+19 more)

### Community 52 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.29
Nodes (6): Local input, Native references checked, Privacy / layar hitam, Remote session controls (0.2.52), Send file, Verification

### Community 56 - "dialogs.test.cjs"
Cohesion: 0.11
Nodes (15): assert, createElement(), element(), fs, path, test, vm, assert (+7 more)

### Community 57 - "dialogs.js"
Cohesion: 0.53
Nodes (5): appAlert(), appConfirm(), appDialog(), appDialogQueue, appPrompt()

### Community 58 - "main"
Cohesion: 0.27
Nodes (8): browseRemoteFiles(), deploymentDevices, loadDeployments(), openDeployments(), openFileManager(), renderDeployTargets(), submitDeployment(), supportsDeployment()

### Community 59 - "process_usage_windows.go"
Cohesion: 0.21
Nodes (14): SystemInfo, applicationUsage24Hours(), getRemoteClipboard(), CollectSystemInfo(), getAllIPs(), getCPUModel(), GetCPUUsage(), getDiskInfo() (+6 more)

### Community 61 - "README.md"
Cohesion: 0.20
Nodes (8): RawMessage, Request, ResponseWriter, Server, IsNewer(), parse(), T, TestIsNewer()

### Community 62 - "startRemote"
Cohesion: 0.17
Nodes (11): app, assets, fs, html, http, index, path, root (+3 more)

### Community 63 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.40
Nodes (4): Batas verifikasi, Deploy aplikasi, Deploy aplikasi dan transfer file — 0.2.56, Transfer file

### Community 64 - "TestRemoteInputProtectionCleanup"
Cohesion: 0.67
Nodes (3): T, TestRemoteInputProtectionCleanup(), TestRemoteProtectionDesktopTransitionReassertsBlockInput()

### Community 65 - "AssetSwitchRequest"
Cohesion: 0.12
Nodes (26): closeDeviceModal(), closeManualAssetModal(), closeRustDeskManageModal(), closeVerificationModal(), deleteDevice(), deleteManualAsset(), filterDevices(), goPage() (+18 more)

### Community 68 - ".deployApplication"
Cohesion: 0.42
Nodes (7): boundedEndpointText(), collectEndpointReport(), installedMachineApplications(), managementBlocker(), openMachineSCM(), readLockValue(), Mgr

### Community 69 - ".handleDeployments"
Cohesion: 0.32
Nodes (4): RawMessage, Request, ResponseWriter, Server

### Community 70 - "qrcode.min.js"
Cohesion: 0.32
Nodes (4): detectApplications(), DB, validTrackedApplications(), TrackedApplication

### Community 72 - "Remote disconnect investigation — 2026-09-30"
Cohesion: 0.50
Nodes (3): Remote disconnect investigation — 2026-09-30, Verification and limits, Verified in source

### Community 73 - "lockPolicyState"
Cohesion: 0.20
Nodes (6): assert, fs, idleSource, source, test, vm

### Community 74 - "ui-smoke.cjs"
Cohesion: 0.22
Nodes (8): assert, { createRequire }, fixture, fs, http, path, responses, root

### Community 75 - "detectApplications"
Cohesion: 0.60
Nodes (4): T, TestAgentBinaryPlatformIsolation(), TestMacAgentPackage(), TestMacInstallerShellSyntax()

### Community 76 - ".heartbeatLoop"
Cohesion: 0.31
Nodes (7): activeApplications(), enrichProcesses(), isSystemProcess(), processIcon(), listProcesses(), listProcesses(), ProcessInfo

### Community 77 - ".handleLockPolicy"
Cohesion: 0.29
Nodes (6): Advertise LAN pilot (0.2.69), Operation, Safety and cleanup, Scope and prerequisites, Upstream references reviewed, Verification

### Community 78 - "devices.test.cjs"
Cohesion: 0.25
Nodes (6): assert, fs, path, source, test, vm

### Community 79 - "client.go"
Cohesion: 0.23
Nodes (10): envOr(), main(), LoadConfig(), T, TestLoadConfigAcceptsUTF8BOM(), TestNewAgentPersistsGeneratedDeviceID(), T, TestAgentConnectEncodesSpecialKeyAndDeviceID() (+2 more)

### Community 80 - "qrcode.min.js"
Cohesion: 0.12
Nodes (21): remoteScreenState, B, Image, captureRemoteFrame(), pasteRemoteText(), remotePasteKeys(), T, TestPasteRemoteTextValidationAndSequence() (+13 more)

### Community 81 - "executeLockPolicy"
Cohesion: 0.27
Nodes (4): remotePrivacyWindow, Int64, remoteProtection, startRemotePrivacy()

### Community 85 - "Device"
Cohesion: 0.31
Nodes (6): migrate(), scanDevice(), scanDeviceRows(), Device, Rows, scanner

### Community 86 - "key-rotation.test.cjs"
Cohesion: 0.22
Nodes (7): assert, fs, path, rotation, source, test, vm

### Community 87 - "vpn.js"
Cohesion: 0.21
Nodes (20): closeVPNModal(), getVPNDevice(), handleVPNDialogKeydown(), loadVPNMaster(), loadVPNSelfPolicy(), onVPNDevicePickerChange(), openVPNMaster(), openVPNModal() (+12 more)

### Community 88 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.23
Nodes (19): Server, T, lanServer(), lanSession(), TestVPNLANAPIRequiresAdminAndSendsApprovedConfiguration(), TestVPNLANAuthorizationAndCollision(), TestVPNLANBaselineFirewallIsDefaultDeny(), TestVPNLANPartialSetupAndCleanupFailure() (+11 more)

### Community 89 - "vpn.test.cjs"
Cohesion: 0.36
Nodes (5): Request, ResponseWriter, Server, tokenDigest(), UserClaims

### Community 90 - ".deployApplication"
Cohesion: 0.29
Nodes (5): applicationDeployment, deploymentExitStatus(), Agent, T, TestDeploymentExitStatus()

### Community 91 - ".handleDeployments"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 93 - "detectApplications"
Cohesion: 0.27
Nodes (7): T, TestAdminCannotBypassMFAWithTrustedDevice(), TestMFAManagementRequiresReauthentication(), GenerateTOTPCode(), GenerateTOTPSecret(), Time, parseMFATicket()

### Community 94 - "hashPassword"
Cohesion: 0.06
Nodes (34): remoteDeskService, ChangeRequest, T, TestStoppedSupervisorTerminatesOwnedWorker(), enableTokenPrivilege(), exeToUTF16(), Handle, runSystemService() (+26 more)

### Community 95 - ".GetAssetVerificationsFiltered"
Cohesion: 0.25
Nodes (7): appSource, assert, fs, indexHtml, path, test, vm

### Community 96 - "README.md"
Cohesion: 0.25
Nodes (7): assets, fs, http, index, modal, path, root

### Community 98 - ".acceptNewPassword"
Cohesion: 0.17
Nodes (5): assert, fs, source, test, vm

### Community 99 - "Rotasi API key bertahap (v0.2.62)"
Cohesion: 0.50
Nodes (3): Rotasi API key bertahap (v0.2.62), Upgrade dari implementasi lama, Verifikasi

### Community 100 - "arpNeighbors"
Cohesion: 0.60
Nodes (4): arpNeighbors(), ScanLocalNetwork(), IP, NetworkScanHost

### Community 101 - "Daftar API"
Cohesion: 0.32
Nodes (6): Request, T, TestBreachScreeningAcrossPasswordRoutes(), TestPwnedPasswordRangePrivacyAndFailures(), Response, breachTransport

### Community 102 - "TestUnlinkDeviceManualAsset"
Cohesion: 0.29
Nodes (6): Pengukuran lokal, Perbaikan disconnect prematur WireGuard pilot (v0.2.68), Perubahan dan bukti, RemoteDesk: paket perbaikan untuk uji coba, Uji setelah penerapan yang diotorisasi, Verifikasi dan batas

### Community 106 - "Pasang Server di VPS Linux (Tanpa Docker)"
Cohesion: 0.40
Nodes (4): envOr(), main(), checkPwnedPassword(), New()

### Community 107 - "3. Halaman Devices"
Cohesion: 0.40
Nodes (5): 3. Halaman Devices, Cari Device, Filter, Hapus Device, Lihat Detail Device

### Community 108 - "4. Halaman Asset Inventory"
Cohesion: 0.50
Nodes (4): 4. Halaman Asset Inventory, Apa Bedanya dengan Halaman Devices?, Export ke CSV (Excel), Kolom yang Ditampilkan

### Community 109 - "Tips"
Cohesion: 0.47
Nodes (4): Context, Request, ResponseWriter, Server

### Community 112 - "vpn-tray.md"
Cohesion: 0.50
Nodes (3): Admin policy, Local boundary and verification, Windows VPN tray

### Community 113 - "Pasang di Linux"
Cohesion: 0.40
Nodes (5): Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File, Pasang di Linux

### Community 114 - "TestDeviceIdentityMigrationAndReopen"
Cohesion: 0.67
Nodes (3): T, TestDeviceIdentityMigrationAndReopen(), TestDeviceIdentityPersistenceValidationAndPermissions()

## Knowledge Gaps
- **232 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `Agent`, `rustDeskExportConfig`, `Agent` (+227 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **17 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Pasang Server di VPS Linux (Tanpa Docker)` to `Community 1`, `main`, `.deployApplication`, `.handleDeployments`, `Community 11`, `.heartbeatLoop`, `SecuritySettings`, `qrcode.min.js`, `AssetSwitchRequest`, `Skenario: Server di VPS Linux, Agent di Komputer Windows`, `NewDB`, `detectApplications`, `Skenario: Server di VPS Linux, Agent di Komputer Windows`, `vpn.test.cjs`, `.deployApplication`, `esc`?**
  _High betweenness centrality (0.112) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 3`, `Pasang Server di VPS Linux (Tanpa Docker)`, `.heartbeatLoop`, `Daftar API`, `device.go`, `hashPassword`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **Why does `NewDB()` connect `NewDB` to `Community 2`, `Community 7`, `Pasang Server di VPS Linux (Tanpa Docker)`, `SecuritySettings`, `TestDeviceIdentityMigrationAndReopen`, `Device`, `.Close`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **Are the 16 inferred relationships involving `jsonError()` (e.g. with `.acceptNewPassword()` and `.authorizeUserToken()`) actually correct?**
  _`jsonError()` has 16 INFERRED edges - model-reasoned connections that need verification._
- **Are the 13 inferred relationships involving `getClaims()` (e.g. with `.handleActivities()` and `.handleAttachment()`) actually correct?**
  _`getClaims()` has 13 INFERRED edges - model-reasoned connections that need verification._
- **Are the 12 inferred relationships involving `jsonResp()` (e.g. with `.handleActivities()` and `.handleDeployments()`) actually correct?**
  _`jsonResp()` has 12 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `Agent` to the rest of the system?**
  _232 weakly-connected nodes found - possible documentation gaps or missing edges._