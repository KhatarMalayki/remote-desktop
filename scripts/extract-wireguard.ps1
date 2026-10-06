param(
    [string]$DestinationDir = (Join-Path $PSScriptRoot "..\prebuilt-agent")
)
$ErrorActionPreference = 'Stop'
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("wg-extract-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tempDir | Out-Null
try {
    $wgMsi = Join-Path $tempDir "wireguard-amd64.msi"
    Invoke-WebRequest "https://download.wireguard.com/windows-client/wireguard-amd64-0.5.3.msi" -OutFile $wgMsi -UseBasicParsing
    $expectedMsiHash = "76FCEC042C5989C5B816CD32EAED1E5B1C3B998A4B1C9ECA55F299E3314EF7E4"
    $actualMsiHash = (Get-FileHash -Algorithm SHA256 $wgMsi).Hash
    if ($actualMsiHash -ne $expectedMsiHash) { throw "WireGuard MSI hash mismatch: $actualMsiHash" }
    $msiSig = Get-AuthenticodeSignature -FilePath $wgMsi
    if ($msiSig.Status -ne "Valid" -or -not $msiSig.SignerCertificate.Subject.Contains("WireGuard")) { throw "WireGuard MSI signature invalid: $($msiSig.Status)" }

    $csharp = @"
using System;
using System.IO;
using System.Runtime.InteropServices;
public class WgMsiStreamDumper {
    [DllImport("msi.dll", CharSet = CharSet.Unicode)] static extern uint MsiOpenDatabase(string path, IntPtr persist, out IntPtr db);
    [DllImport("msi.dll", CharSet = CharSet.Unicode)] static extern uint MsiDatabaseOpenView(IntPtr db, string query, out IntPtr view);
    [DllImport("msi.dll")] static extern uint MsiViewExecute(IntPtr view, IntPtr record);
    [DllImport("msi.dll")] static extern uint MsiViewFetch(IntPtr view, out IntPtr record);
    [DllImport("msi.dll")] static extern uint MsiRecordReadStream(IntPtr record, uint field, byte[] buffer, ref uint count);
    [DllImport("msi.dll")] static extern uint MsiCloseHandle(IntPtr handle);
    public static void ExtractStream(string msi, string streamName, string outPath) {
        IntPtr db, view, record;
        if (MsiOpenDatabase(msi, IntPtr.Zero, out db) != 0) throw new Exception("open db failed");
        string sql = string.Format("SELECT Data FROM _Streams WHERE Name = '{0}'", streamName);
        if (MsiDatabaseOpenView(db, sql, out view) != 0) throw new Exception("open view failed: " + sql);
        MsiViewExecute(view, IntPtr.Zero);
        if (MsiViewFetch(view, out record) != 0) throw new Exception("fetch failed");
        using (var fs = File.Create(outPath)) {
            byte[] buf = new byte[65536]; uint read = (uint)buf.Length;
            while (MsiRecordReadStream(record, 1, buf, ref read) == 0 && read > 0) { fs.Write(buf, 0, (int)read); read = (uint)buf.Length; }
        }
        MsiCloseHandle(record); MsiCloseHandle(view); MsiCloseHandle(db);
    }
}
"@
    if (-not ("WgMsiStreamDumper" -as [type])) {
        Add-Type -TypeDefinition $csharp -Language CSharp
    }
    $wgCab = Join-Path $tempDir "wg-cab.cab"
    [WgMsiStreamDumper]::ExtractStream($wgMsi, "cab1.cab", $wgCab)
    $wgExtracted = Join-Path $tempDir "wg-extracted"
    New-Item -ItemType Directory -Force -Path $wgExtracted | Out-Null
    & expand.exe -R $wgCab -F:* $wgExtracted | Out-Null
    $wgExe = Get-ChildItem -Path $wgExtracted -Recurse -File -Filter "wireguard.exe" | Select-Object -First 1
    if (-not $wgExe) { throw "wireguard.exe not found in MSI" }
    $wgSig = Get-AuthenticodeSignature -FilePath $wgExe.FullName
    if ($wgSig.Status -ne "Valid" -or -not $wgSig.SignerCertificate.Subject.Contains("WireGuard")) { throw "wireguard.exe signature invalid: $($wgSig.Status)" }
    New-Item -ItemType Directory -Force -Path $DestinationDir | Out-Null
    Copy-Item -LiteralPath $wgExe.FullName -Destination (Join-Path $DestinationDir "wireguard.exe") -Force
} finally {
    Remove-Item -Recurse -Force -LiteralPath $tempDir -ErrorAction SilentlyContinue
}
