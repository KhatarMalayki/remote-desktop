# Remote web input investigation — 2026-09-29

## Scope and evidence

Inspected source from the repositories below in a temporary directory. No upstream
executables, installers, scripts, or new dependencies were run or imported.
This is a bounded source comparison, not a security audit or a guarantee that an
entire repository has no backdoors. ManageEngine implementation was not inspected.

| Repository / pinned commit | Relevant source | Finding |
| --- | --- | --- |
| RustDesk `4812a9815bd3c6a93f3ad903f29504168c4930a1` | [windows.cc](https://github.com/rustdesk/rustdesk/blob/4812a9815bd3c6a93f3ad903f29504168c4930a1/src/platform/windows.cc#L278), [win_impl.rs](https://github.com/rustdesk/rustdesk/blob/4812a9815bd3c6a93f3ad903f29504168c4930a1/libs/enigo/src/win/win_impl.rs#L23) | Desktop selection requests object, hook, enumeration, switch and generic-write access; retains the selected desktop handle; uses native INPUT/SendInput. |
| MeshAgent `709f373d2ccb71b82945b56aaf8c9bc28c3fde8a` | [kvm.c](https://github.com/Ylianst/MeshAgent/blob/709f373d2ccb71b82945b56aaf8c9bc28c3fde8a/meshcore/KVM/Windows/kvm.c#L324), [input.c](https://github.com/Ylianst/MeshAgent/blob/709f373d2ccb71b82945b56aaf8c9bc28c3fde8a/meshcore/KVM/Windows/input.c#L391) | Same desktop access pattern; closes the previous handle after successful attachment; sends absolute virtual-desktop mouse input with button flags. |
| UltraVNC `0cd8b41aeb34e52e170b33e24411b3f805e303be` | [HelperFunctions.cpp](https://github.com/ultravnc/UltraVNC/blob/0cd8b41aeb34e52e170b33e24411b3f805e303be/winvnc/winvnc/HelperFunctions.cpp#L446) | Same expanded desktop access mask; explicit error paths for desktop selection. |

## Confirmed defects in our implementation

1. `prepareRemoteDesktop` requested only READOBJECTS, WRITEOBJECTS and
   SWITCHDESKTOP (`0x181`). All three inspected projects also request CREATEWINDOW,
   CREATEMENU, ENUMERATE, HOOKCONTROL and GENERIC_WRITE. This is a credible missing
   access requirement. The native comparison below reproduces its effect locally;
   the specific endpoint still requires verification.
2. The old function attempted to close the desktop handle while its thread still
   used it, ignoring the result. Microsoft documents that CloseDesktop fails in
   that situation. Repeating it for frames and events could leak handles.
3. Any SendInput error triggered a forced Winlogon reconnect for 15 seconds, even
   when the active desktop was Default. Input rejection does not establish that
   Windows is locked. Removed this inference and its stale force timer.
4. SendInput's captured syscall error was discarded by the dependency wrapper;
   another GetLastError call was used instead. Now use the error returned by the
   same native call. Zero/zero alone never proves UIPI.
5. SetCursorPos errors were ignored. Now clicks stop and report the positioning
   failure instead of silently clicking at an unknown position.
6. The remote screen did not identify the serving agent version. The server
   sidebar's version does not prove the executable serving the relay is updated.
   The live relay now supplies its own version beside FPS.

## Applied changes and limits

Expanded desktop access without changing ACLs, disabling UAC, or changing Windows
policy. Each pinned relay thread owns one selected desktop handle, closes replaced
handles and restores its original desktop before cleanup. SYSTEM/session
architecture is unchanged. Native failure reporting and version visibility improved.

Unit checks cover INPUT sizes on the build architecture and preservation of native
errors. A Windows desktop test exercises repeated attachment, handle closure and
restoration without moving the cursor or generating keystrokes/clicks. It skips
when no accessible interactive desktop exists. These checks are not an end-to-end
click test on SS-HO-EPSON.

## Native reproduction result

Executed `TestNativeInputDesktopProbe` on this Windows amd64 machine with
`RD_NATIVE_INPUT_PROBE=1`. The same process and pinned OS thread, on Default,
submitted an INPUT containing only MOUSEEVENTF_MOVE with zero displacement.
No mouse button or keyboard events were generated; identity/privilege was unchanged.

```text
old desktop access 0x181: Windows menolak input mouse (SendInput: 0, error: Access is denied., desktop: Default)
upstream desktop access 0x400001cf: <nil>
PASS
```

This demonstrates a local causal failure from the old desktop access mask, rather
than relying on a UIPI hypothesis or a successful build. It does not prove all
endpoint configurations behave identically. The probe is opt-in and skipped during
normal tests. Reproduce in PowerShell on an interactive Windows test machine:

```powershell
$env:RD_NATIVE_INPUT_PROBE = '1'
go test ./internal/agent -run '^TestNativeInputDesktopProbe$' -v -count=1
Remove-Item Env:RD_NATIVE_INPUT_PROBE
```

For endpoint verification, confirm the live relay says Agent 0.2.51, test left/right/
double-click and keyboard in a non-elevated application, then record the exact
input error and matching worker.log timestamp if it fails. A successful build or
unit test does not establish that the target is fixed. No endpoint logs or access
were available during this investigation.

## Microsoft references

- [SendInput return values and UIPI limitations](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-sendinput)
- [Desktop security and generic access mapping](https://learn.microsoft.com/en-us/windows/win32/winstation/desktop-security-and-access-rights)
- [CloseDesktop restrictions](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-closedesktop)
