# Graph Report - remote-desktop  (2026-10-10)

## Corpus Check
- 150 files · ~283,630 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1479 nodes · 2932 edges · 114 communities (97 shown, 17 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 412 edges (avg confidence: 0.79)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `18005a32`
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
- TestDeviceIdentityMigrationAndReopen
- TestHolderManagementPermissions
- vpn-tray.md
- SecuritySettings

## God Nodes (most connected - your core abstractions)
1. `Server` - 76 edges
2. `DB` - 72 edges
3. `jsonError()` - 63 edges
4. `getClaims()` - 56 edges
5. `jsonResp()` - 55 edges
6. `api()` - 51 edges
7. `NewDB()` - 50 edges
8. `showToast()` - 44 edges
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

## Communities (114 total, 17 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.13
Nodes (10): AgentConfig, CleanupOldExecutable(), executeShellCommand(), generateDeviceID(), Agent, Conn, Mutex, RWMutex (+2 more)

### Community 1 - "Community 1"
Cohesion: 0.06
Nodes (48): FS, addMacInstallers(), agentBinaryCandidates(), canAccessSwitch(), Request, ResponseWriter, DB, Server (+40 more)

### Community 3 - "Community 3"
Cohesion: 0.15
Nodes (6): Conn, DB, RWMutex, SignalMessage, Client, Hub

### Community 4 - "RemoteDesk"
Cohesion: 0.18
Nodes (11): Apa Itu RemoteDesk?, Build Agent untuk Linux dari Windows, Cara Build (Step by Step), Di Linux / macOS (Terminal), Di Windows (PowerShell), Dokumentasi Lengkap, Fitur, Hasil Build = File Apa? Ada di Mana? (+3 more)

### Community 5 - "Community 5"
Cohesion: 0.11
Nodes (18): Auto-Start (LaunchDaemon), Cara 1: Langsung Jalankan (Paling Cepat), Cara 1: Langsung Jalankan (Tes Dulu), Cara 2: Jalan di Background (Sederhana), Cara 2: Pakai Task Scheduler (Jalan Otomatis Waktu Startup), Cara 3: Pakai NSSM (Jadi Windows Service), Cara 3: Systemd Service (Paling Benar, Auto Restart), Cara 4: Pakai Config File (+10 more)

### Community 6 - "Community 6"
Cohesion: 0.15
Nodes (12): 1. Login, 2. Dashboard, 5. Halaman Remote Desktop, 6. Logout, Cara Cepat, Cara Pakai RemoteDesk, Cara Remote ke Komputer, Grafik (Tengah) (+4 more)

### Community 7 - "Community 7"
Cohesion: 0.20
Nodes (16): attachAgentRelay(), closeRelaySession(), createViewerRelay(), Conn, Duration, Mutex, Time, pipeRelay() (+8 more)

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
Cohesion: 0.08
Nodes (42): remoteDeskService, ChangeRequest, T, TestStoppedSupervisorTerminatesOwnedWorker(), enableTokenPrivilege(), exeToUTF16(), Handle, runSystemService() (+34 more)

### Community 17 - "device.go"
Cohesion: 0.12
Nodes (13): ManageRustDesk(), RawMessage, Time, AssetVerification, AuthLog, BlockedIPInfo, DeviceHeartbeat, NetworkScan (+5 more)

### Community 19 - "NewDB"
Cohesion: 0.07
Nodes (59): T, TestAPIKeyRotationIsExplicitAndPersistent(), TestBulkKeyMigrationRequiresPilotAndSkipsIneligible(), TestOldKeyWebSocketDoesNotAutoMigrate(), TestRetainShortLegacyKeyWithoutWeakeningNewKeys(), NewDB(), T, TestADHBranchIsolation() (+51 more)

### Community 20 - "qrcode.min.js"
Cohesion: 0.05
Nodes (27): remoteCommand, remoteFileTransfer, Hash, File, Time, T, TestRemoteFileNames(), TestRemoteFileRejectsInvalidTransfer() (+19 more)

### Community 22 - "db.go"
Cohesion: 0.16
Nodes (15): closeHandoverModal(), closeRelocateModal(), closeServiceModal(), isAssetUser(), loadAssetActivities(), mayReviewSwitch(), openAssetTimeline(), openSwitchHistory() (+7 more)

### Community 25 - "main"
Cohesion: 0.07
Nodes (14): assert, fs, path, setTimeout(), test, vm, assert, fs (+6 more)

### Community 26 - "validBranchType"
Cohesion: 0.06
Nodes (30): bindRemoteKeyboard(), closeDownloadAgentModal(), devices, doLogout(), loadHistoryLogs(), loadSecurityLogs(), manualAssets, openHistoryModal() (+22 more)

### Community 27 - "SecuritySettings"
Cohesion: 0.11
Nodes (18): exportAssets(), fmtBytes(), onSearchBranchAssets(), openDeviceModal(), openEditAssetModal(), openVerifyModal(), remoteFileRequest(), renderAssets() (+10 more)

### Community 28 - "loadDevices"
Cohesion: 0.14
Nodes (16): buildDeviceTable(), connectWS(), deviceResourceUsage(), deviceVPNBadge(), handleSignal(), isVersionNewer(), keyStatusLabel(), openReconfigureModal() (+8 more)

### Community 29 - "esc"
Cohesion: 0.05
Nodes (69): vpnClient, vpnLANInterface, vpnLANState, vpnLease, Mutex, Time, Config, T (+61 more)

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
Nodes (37): addBusinessUnit(), api(), buildBranchOptionsHTML(), closeBranchesModal(), closeChangeUsernameModal(), closeEditUserModal(), closeManualAssetModal(), createBranch() (+29 more)

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
Cohesion: 0.17
Nodes (9): Cara A: Build Sendiri dari Source Code (Perlu Go), Cara B: Pakai Docker (Untuk Server Saja), Cara C: Pakai Docker Compose (Paling Simple), Cara Dapat File Program-nya, Cara Install RemoteDesk, Checklist Setelah Install, Pakai Caddy (Paling Gampang, Auto SSL), Pasang HTTPS (Biar Aman) (+1 more)

### Community 42 - "parseRustDeskID"
Cohesion: 0.29
Nodes (4): parseRustDeskID(), T, TestParseRustDeskID(), getRustDeskID()

### Community 43 - "SecuritySettings"
Cohesion: 0.36
Nodes (11): authRequest(), cleanPasswordFixture(), Context, HandlerFunc, Server, T, securityServer(), TestCredentialTokenIntegrityAndAccountChanges() (+3 more)

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
Cohesion: 0.17
Nodes (17): HMENU, HWND, Handle, initVPNTray(), launchVPNTray(), vpnPanelURL(), appendMenuItem(), appendMenuSeparator() (+9 more)

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
Cohesion: 0.21
Nodes (14): SystemInfo, applicationUsage24Hours(), getRemoteClipboard(), CollectSystemInfo(), getAllIPs(), getCPUModel(), GetCPUUsage(), getDiskInfo() (+6 more)

### Community 61 - "README.md"
Cohesion: 0.20
Nodes (8): RawMessage, Request, ResponseWriter, Server, IsNewer(), parse(), T, TestIsNewer()

### Community 63 - "Hitungan Memori: Kok Bisa 1000 Device di 1 GB?"
Cohesion: 0.40
Nodes (4): Batas verifikasi, Deploy aplikasi, Deploy aplikasi dan transfer file — 0.2.56, Transfer file

### Community 64 - "TestRemoteInputProtectionCleanup"
Cohesion: 0.67
Nodes (3): T, TestRemoteInputProtectionCleanup(), TestRemoteProtectionDesktopTransitionReassertsBlockInput()

### Community 65 - "AssetSwitchRequest"
Cohesion: 0.13
Nodes (24): closeDeviceModal(), closeRustDeskManageModal(), closeVerificationModal(), deleteDevice(), deleteManualAsset(), filterDevices(), goPage(), init() (+16 more)

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
Cohesion: 0.32
Nodes (11): closeVPNModal(), handleVPNDialogKeydown(), onVPNDevicePickerChange(), openVPNModal(), refreshVPNState(), renderVPNLANControls(), renderVPNModal(), resetVPNLANForm() (+3 more)

### Community 88 - "Skenario: Server di VPS Linux, Agent di Komputer Windows"
Cohesion: 0.36
Nodes (5): Request, ResponseWriter, Server, tokenDigest(), UserClaims

### Community 89 - "vpn.test.cjs"
Cohesion: 0.22
Nodes (7): assert, fs, html, path, source, test, vm

### Community 90 - ".deployApplication"
Cohesion: 0.29
Nodes (5): applicationDeployment, deploymentExitStatus(), Agent, T, TestDeploymentExitStatus()

### Community 91 - ".handleDeployments"
Cohesion: 0.40
Nodes (5): 1. Upload `rd-server` ke VPS, 2. Jalankan server di VPS, 3. Jalankan agent di komputer Windows, Setelah Build, Terus Ngapain?, Skenario: Server di VPS Linux, Agent di Komputer Windows

### Community 93 - "detectApplications"
Cohesion: 0.29
Nodes (8): hashPassword(), Request, T, TestBreachScreeningAcrossPasswordRoutes(), TestBreachScreeningAtLogin(), TestPwnedPasswordRangePrivacyAndFailures(), Response, breachTransport

### Community 94 - "hashPassword"
Cohesion: 0.11
Nodes (21): Config, Mutex, RawMessage, Request, ResponseWriter, Server, Time, Server (+13 more)

### Community 95 - ".GetAssetVerificationsFiltered"
Cohesion: 0.25
Nodes (7): appSource, assert, fs, indexHtml, path, test, vm

### Community 96 - "README.md"
Cohesion: 0.27
Nodes (7): T, TestAdminCannotBypassMFAWithTrustedDevice(), TestMFAManagementRequiresReauthentication(), GenerateTOTPCode(), GenerateTOTPSecret(), Time, parseMFATicket()

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
Cohesion: 0.22
Nodes (9): Activity Log Device, Daftar API, Detail Satu Device, Hapus Device, List Semua Device, List Semua Group, Login, Statistik Dashboard (+1 more)

### Community 102 - "TestUnlinkDeviceManualAsset"
Cohesion: 0.29
Nodes (6): Pengukuran lokal, Perbaikan disconnect prematur WireGuard pilot (v0.2.68), Perubahan dan bukti, RemoteDesk: paket perbaikan untuk uji coba, Uji setelah penerapan yang diotorisasi, Verifikasi dan batas

### Community 103 - "New"
Cohesion: 0.40
Nodes (4): envOr(), main(), checkPwnedPassword(), New()

### Community 104 - "verifyPassword"
Cohesion: 0.40
Nodes (4): TestPasswordHashAndPolicy(), validatePassword(), verifyPassword(), credentialToken

### Community 105 - ".acceptNewPassword"
Cohesion: 0.47
Nodes (4): Context, Request, ResponseWriter, Server

### Community 106 - "Pasang Server di VPS Linux (Tanpa Docker)"
Cohesion: 0.40
Nodes (5): Langkah 1: Upload file rd-server ke VPS, Langkah 2: Beri izin execute, Langkah 3: Jalankan server (tes dulu), Langkah 4: Bikin Jalan Otomatis (Systemd), Pasang Server di VPS Linux (Tanpa Docker)

### Community 107 - "3. Halaman Devices"
Cohesion: 0.40
Nodes (5): 3. Halaman Devices, Cari Device, Filter, Hapus Device, Lihat Detail Device

### Community 108 - "4. Halaman Asset Inventory"
Cohesion: 0.50
Nodes (4): 4. Halaman Asset Inventory, Apa Bedanya dengan Halaman Devices?, Export ke CSV (Excel), Kolom yang Ditampilkan

### Community 109 - "Tips"
Cohesion: 0.50
Nodes (4): Keamanan, Monitoring Efektif, Organisasi Device yang Rapi, Tips

### Community 110 - "TestDeviceIdentityMigrationAndReopen"
Cohesion: 0.67
Nodes (3): T, TestDeviceIdentityMigrationAndReopen(), TestDeviceIdentityPersistenceValidationAndPermissions()

## Knowledge Gaps
- **209 isolated node(s):** `github.com/user/remote-desktop`, `Agent`, `Agent`, `rustDeskExportConfig`, `RoleDefinition` (+204 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **17 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `New` to `README.md`, `Community 1`, `main`, `.deployApplication`, `.handleDeployments`, `.heartbeatLoop`, `qrcode.min.js`, `AssetSwitchRequest`, `Skenario: Server di VPS Linux, Agent di Komputer Windows`, `NewDB`, `qrcode.min.js`, `detectApplications`, `Skenario: Server di VPS Linux, Agent di Komputer Windows`, `.deployApplication`, `esc`, `hashPassword`?**
  _High betweenness centrality (0.115) - this node is a cross-community bridge._
- **Why does `Server` connect `Community 1` to `Community 3`, `New`, `.heartbeatLoop`, `Daftar API`, `device.go`, `hashPassword`?**
  _High betweenness centrality (0.075) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `verifyPassword`, `device.go`, `NewDB`, `SecuritySettings`, `Device`, `.Close`, `ManualAsset`, `startRemote`?**
  _High betweenness centrality (0.062) - this node is a cross-community bridge._
- **Are the 15 inferred relationships involving `jsonError()` (e.g. with `.acceptNewPassword()` and `.authorizeUserToken()`) actually correct?**
  _`jsonError()` has 15 INFERRED edges - model-reasoned connections that need verification._
- **Are the 12 inferred relationships involving `getClaims()` (e.g. with `.handleActivities()` and `.handleAttachment()`) actually correct?**
  _`getClaims()` has 12 INFERRED edges - model-reasoned connections that need verification._
- **Are the 11 inferred relationships involving `jsonResp()` (e.g. with `.handleActivities()` and `.handleDeployments()`) actually correct?**
  _`jsonResp()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/user/remote-desktop`, `Agent`, `Agent` to the rest of the system?**
  _209 weakly-connected nodes found - possible documentation gaps or missing edges._