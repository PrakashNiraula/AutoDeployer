@echo off
REM Starts a long-lived child and records its PID, then exits 0.
REM Used for detach survival tests (P0-a4).
setlocal
cd /d "%~dp0"
set "PIDFILE=%~dp0detach_child.pid"
if exist "%PIDFILE%" del "%PIDFILE%"

REM Start java sleep in a new process; capture PID via wmic/powershell
powershell -NoProfile -Command ^
  "$p = Start-Process -FilePath 'java' -ArgumentList '-cp','.','SleepJava','600000' -PassThru -WindowStyle Hidden; Set-Content -Path '%PIDFILE%' -Value $p.Id; Write-Output $p.Id"

if not exist "%PIDFILE%" (
  echo failed to write detach_child.pid
  exit /b 1
)
echo detach child pid:
type "%PIDFILE%"
exit /b 0
