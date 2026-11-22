# s3-pull-and-java

Pulls files from SFTP/S3, then runs a Java class under Tomcat.

**Stub scripts.** Replace `scripts/pull_s3.bat` and `scripts/run_java.bat` with the real ones.

Script conventions:
- Write outputs only under `%AD_JOB_OUTPUT_DIR%`, never into `project/`.
- Call nested batch files with `call` and finish with `exit /b <code>` so failures reach AutoDeployer.
- Real values live in `data/env/s3-pull-and-java/.env.<profile>`; copy from `env/*.example`.
