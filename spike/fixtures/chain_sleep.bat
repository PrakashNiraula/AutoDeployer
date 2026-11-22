@echo off
REM Starts SleepJava as a child. Used for tree-kill tests (P0-a1, P0-a2).
setlocal
cd /d "%~dp0"
if not exist SleepJava.class (
  echo SleepJava.class missing. Compile with: javac SleepJava.java
  exit /b 2
)
java -cp . SleepJava 600000
exit /b %ERRORLEVEL%
