@echo off
REM Nested bat WITH call + exit /b — correct propagation.
cd /d "%~dp0"
call nested_inner.bat
echo after_call ERRORLEVEL=%ERRORLEVEL%
exit /b %ERRORLEVEL%
