@echo off
REM Prefer UTF-8 output (Java helper writes UTF-8 regardless of console CP).
setlocal
cd /d "%~dp0"
if exist OutUTF8.class (
  java -cp . OutUTF8
  exit /b %ERRORLEVEL%
)
REM Fallback: write UTF-8 via PowerShell
powershell -NoProfile -Command "[Console]::OutputEncoding = [Text.UTF8Encoding]::new($false); Write-Output '日本語テスト_utf8'; Write-Output 'ライン2_処理成功'"
exit /b %ERRORLEVEL%
