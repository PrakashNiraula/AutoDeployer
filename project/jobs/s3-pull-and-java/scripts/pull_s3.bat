@echo off
rem STUB: replace with the real S3/SFTP pull.
chcp 932 >nul
echo SFTP接続開始: host=%SFTP_HOST% user=%SFTP_USER%
echo バケット: %S3_BUCKET%
echo 出力先: %AD_JOB_OUTPUT_DIR%
echo ファイル取得完了
exit /b 0
