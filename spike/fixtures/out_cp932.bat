@echo off
REM Emit Japanese text using the console OEM/ANSI code page (often cp932 on JP Windows).
REM The harness also has a pre-encoded cp932 byte fixture for Linux decoder tests.
chcp 932 >nul
echo 日本語テスト_cp932
echo ライン2_取得完了
exit /b 0
