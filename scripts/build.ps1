Write-Host "Building RemoteDesk..." -ForegroundColor Cyan

$ErrorActionPreference = "Stop"

if (!(Test-Path "bin")) { New-Item -ItemType Directory -Path "bin" | Out-Null }

Write-Host "=> Building server..."
$env:CGO_ENABLED = "1"
go build -ldflags="-s -w" -o bin/rd-server.exe ./cmd/server

Write-Host "=> Building agent (windows)..."
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o bin/rd-agent.exe ./cmd/agent

Write-Host ""
Write-Host "Build complete!" -ForegroundColor Green
Get-ChildItem bin/ | Format-Table Name, Length
