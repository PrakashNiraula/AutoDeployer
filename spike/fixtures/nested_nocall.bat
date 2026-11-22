@echo off
REM Nested bat WITHOUT call — classic %ERRORLEVEL% pitfall.
cd /d "%~dp0"
nested_inner.bat
echo after_nocall ERRORLEVEL=%ERRORLEVEL%
exit /b %ERRORLEVEL%
