# Run on Japanese-locale Windows to complete P0-a*/P0-c* bat cases.
# Usage (from repo root):
#   go run ./spike/harness -fixtures ./spike/fixtures
#   OR: powershell -File spike/run-windows.ps1

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot\..
Write-Host "Locale: $((Get-WinSystemLocale).Name)  OEMCP: $((chcp.com) -replace '[^\d]','')"
go version
java -version 2>&1 | Select-Object -First 1
go run ./spike/harness -fixtures ./spike/fixtures
Write-Host "Paste the table into spike/REPORT.md Results section."
