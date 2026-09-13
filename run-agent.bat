@echo off
title RemoteDesk Agent
echo ===================================================
echo   Memulai RemoteDesk Agent...
echo ===================================================

if exist "%~dp0rd-agent.exe" (
    "%~dp0rd-agent.exe" %*
) else if exist "%~dp0bin\rd-agent.exe" (
    "%~dp0bin\rd-agent.exe" %*
) else (
    echo Error: rd-agent.exe tidak ditemukan!
    echo Silakan jalankan 'scripts\build.ps1' untuk compile binary.
    pause
    exit /b 1
)

if %ERRORLEVEL% NEQ 0 (
    echo.
    echo Agent terhenti. Pastikan konfigurasi di agent.json sudah benar.
    pause
)
