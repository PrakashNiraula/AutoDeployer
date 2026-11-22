AutoDeployer — Feasibility & Architecture Plan (v4 FINAL)

0. Changes from v3







Change



Why





v1 secrets = plain files readable by operators; hardening deferred



Simplifies ACL/service-account design





Removed from v1: dedicated scheduler account, ACL design, DPAPI/keystore, plan_id HMAC, env-set permission rules



Complexity without v1 need





Kept: log/plan masking, required_env validation, per-machine allowed envs



Cheap, high value





plan_id = plain content hash (not secret)



Binding without crypto ceremony





env get/list mask by default; --reveal; every env set audited in history



Usable + accountable





Cancel = GUI Running list → cancel <run-id> → tree-kill → mark cancelled, clear lock, close row



No polling-file cancel





Invariant: each run is its own OS process; closing GUI never kills it



Real detach





detach: true → Job Object without KILL_ON_JOB_CLOSE, or CREATE_BREAKAWAY_FROM_JOB; Phase 0 criterion



Child survives normal CLI exit





NDJSON events: append-only, one complete line per write, flush per line



No partial-line reads





output_code_page: auto = machine default only; no sniffing



Predictable





Promote-lock wait: 30s manual, 5 min scheduled (configurable)



Schedules survive short promotes





doctor read-only; doctor --fix reconciles stale runs/locks



Safer default





Lock/promote skip (exit 5/12) → history status=skipped + notify if enabled



Visibility for unattended





data/ backup = manual/IT; doctor prints path hint



No backup subsystem in v1





Schedules follow OS timezone; JP machines; history keeps stored offset if zone changes later



Clear time model





Added Deferred (A), Phase 0 checklist (B), Decisions log (C)



Hand-off ready

Unchanged: core/GUI split; CLI contract; app//project//data/; SQLite; mandatory env validation; OS scheduler → CLI; local time + offset; promote lock + rename retry/rollback; per-step encoding + line-boundary decode; fail-fast; path safety; AD_* vars; Phase 0 + fallbacks; Go core; GUI stack in Phase 4.



1. Assumptions and open questions







ID



Topic



Default (work unblocked)



Status





A1



Sample scripts



Stubs / “not available yet”



Assumed





A2



JP/garbled console sample



none → synthetic fixtures in Phase 0



Assumed





A3



Machines



Win10/11 + Server 2019/2022; JP locale on UAT/Prod



Assumed





A4



Task Scheduler UAT/Prod



Allowed



Assumed





A5



Air-gap



Possible → static Go binary



Assumed





A6



AV / EXE policy



May need IT allowlist



Assumed





A7



Scale



≤50 jobs, ≤10 machines, ≤20 ops, ≤200 runs/day



Assumed





A8



Team languages



Go for core; TS/React for GUI later



Assumed





A9



Scheduler identity



Same account as interactive ops (SYSTEM or installing user) until post-v1 hardening



Locked (v1 simplify)





A10



TZ



All machines Japan; schedules = OS local TZ



Locked

Accepted v1 security decision (explicit):  



v1 secrets are plain .env files under data/env/<job-id>/, readable by operators. Hardening is a later phase.



2. Build-vs-buy

Competitor: Task Scheduler + PowerShell wrapper (env load, run bat, append log, maybe CSV).







Need



TS + PS



AutoDeployer CLI





Clock trigger



Yes



Yes (TS → CLI)





Mandatory env check



Easy to skip



Exit 4 always





Encoding (JP / mixed)



Often garbled



Per-step + line-boundary





History query



Ad-hoc



SQLite





Locks + stale PID



Rare



Exclusive + stale detect





Tree-kill timeout/cancel



Hard



Job Object / taskkill /T





Safe project/ promote



Manual risk



Promote lock + rollback





Plan/confirm + plan_id



No



Yes

CLI is the product; GUI is optional. Justify core over wrapper: history, locks, encoding, tree-kill, promote, plan/confirm—not a pretty window.



3. System architecture

3.1 Components

flowchart TB
  Core[autodeployer-core Go CLI]
  GUI[optional thin GUI]
  TS[Task Scheduler]
  Project[project/]
  Data[data/ local disk]
  GUI -->|"subprocess only"| Core
  TS --> Core
  Operator[Operator CLI] --> Core
  Core --> Project
  Core --> Data







Piece



Role





Core



All execution, validation, history, locks, promote, schedule helpers, cancel





GUI



Browse, plan confirm, start run process, Running+Kill, tail logs, env via CLI





OS scheduler



Invokes CLI on a clock

Invariant — run process detach:
Every run is a separate OS process. The GUI only starts it (and may pass handles/files). Closing or crashing the GUI never terminates a run. Cancel/Kill is the only intentional stop. Test required (§7).

Core: Go 1.22+, app/autodeployer.exe.
GUI: choose in Phase 4 (Tauri or Electron + React).

3.2 Process model





One OS process per run (manual, gui, scheduled).



SQLite WAL on local data/ only.



Locks: create-exclusive under data/locks/.



Live truth: log file on disk; optional NDJSON events file.

3.3 CLI contract

Commands







Command



Notes





version



Binary + schema





doctor [--json]



Read-only diagnostics + data/ path backup hint





doctor --fix [--json]



Reconcile stale running rows + stale locks





jobs list|show



Discovery





schema validate [job|all]



Schema, path safety, bat call warnings, project-write warnings





plan <job> --env <p> [--json]



PlanResult + plan_id (content hash)





run <job> --env <p> --trigger manual|scheduled|gui --plan-id <id> --yes --json-meta <file> [--events <ndjson>]



Execute if plan_id matches





cancel <run-id> [--json]



Tree-kill; mark cancelled; clear lock; close row





history list|show



Lazy reconcile of stale running





logs tail <run-id>



Tail UTF-8 log





env list --job <id> [--reveal] [--json]



Masked unless --reveal





env get --job --env --key [--reveal]



Masked unless --reveal





env set --job --env --key --value



Writes plain file; audit row in history





schedule list|set|delete



CRUD + OS task





schedule reconcile [--json] [--strict]



Drift detection





promote --from <dir> [--json]



Promote lock + replace





gc



Retention

Flags: --root, --json, --no-color, --quiet.

Exit codes







Code



Meaning





0



Success





1



Job failed





2



Usage error





3



Schema / path safety





4



Env validation





5



Job lock (concurrent)





6



Timeout





7



Cancelled





8



Promote failed / refused after checks





9



Env not allowed on machine





10



Internal





11



plan_id mismatch





12



Promote lock (wait exhausted)





13



Cancel: not found / not running





14



Schedule reconcile drift (--strict)

Outputs





Log file UTF-8: source of truth.



--json-meta <file>: terminal RunResult (and updates on end states).



--events <ndjson>: append-only; one complete JSON object + \n per write; flush every line; readers never see partial lines.



No protocol on stderr.



4. Detailed specs

4.1 Layout

AutoDeployer/
  app/
    autodeployer.exe
    autodeployer-gui.exe          # optional
  project/                        # replace on promote
    jobs/<job-id>/
      job.yaml
      README.md
      scripts/
      env/*.example
    shared/
  data/                           # LOCAL DISK; never from promote source
    machine.yaml
    env/<job-id>/.env.<profile>   # plain secrets; operators may read
    output/<job-id>/<run-id>/
    logs/<job-id>/
    history.db
    schedules.json
    locks/_promote.lock
    locks/<job-id>.lock
    project-version.json

Backup (v1): Manual/IT owns backup of data/. doctor prints absolute data/ path and a one-line hint: “Back up this directory (history, env, schedules).”

4.2 Canonical time

Canonical: YYYY-MM-DDTHH:mm:ss±HH:MM e.g. 2026-11-20T02:00:00+09:00.







Use



Form





DB



UTC RFC3339 + local + offset columns derived from canonical





Log header



canonical





Filename



YYYYMMDD_HHmmss_p0900_<shortRunId8>_<trigger>.log





Schedules



OS local timezone wall clock





History if OS TZ changes later



Keep stored offset/UTC; do not rewrite old rows

Japan: DST not used; keep UTC sort; minimal DST tests only.

4.3 job.yaml (schema_version: 1)

schema_version: 1
id: s3-pull-and-java
name: S3 Pull then Tomcat Java
description: ...
workdir: scripts
shell: auto
encoding:
  output_code_page: auto       # = machine default; no sniffing
steps:
  - name: Pull
    command: pull_s3.bat
    output_code_page: cp932
  - name: Java
    command: run_java.bat
    output_code_page: utf-8
  - name: Start service
    command: start_tomcat.bat
    detach: true
required_env: [SFTP_HOST, SFTP_USER, SFTP_PASSWORD, TOMCAT_HOME, JAVA_CLASS]
secrets: [SFTP_PASSWORD]       # ⊆ required_env; used for masking only
timeout_minutes: 60
allow_concurrent: false
log_paths: []

detach: true: step may start a long-lived child. Implementation: Job Object without KILL_ON_JOB_CLOSE, and/or child created with CREATE_BREAKAWAY_FROM_JOB. Normal CLI exit must not kill detached children. Timeout/cancel still tree-kills the non-detached run tree (taskkill /T /F / process group).

Tree-kill: only on timeout or cancel—never on normal completion.

Path safety: reject .. or resolution outside project/jobs/<job-id>/ (exit 3).

4.4 Injected env







Var



Meaning





AD_ROOT



Install root





AD_DATA_DIR



data/





AD_JOB_OUTPUT_DIR



data/output/<job-id>/<run-id>/ (created)





AD_RUN_ID



Run id





AD_ENV



Profile





AD_JOB_ID



Job id

Scripts write under AD_JOB_OUTPUT_DIR, never into project/.

4.5 Env validation and masking





Load only data/env/<job-id>/.env.<profile> (plain files).



All required_env present and non-empty → else exit 4.



No silent .example use (Dev opt-in --allow-example-env only if machine allows).



allowed_envs in machine.yaml → else exit 9.



secrets ⊆ required_env (schema error).



Auto-mask key names matching (?i)(PASSWORD|SECRET|TOKEN|KEY).



Min mask length default 4 (machine.yaml).



Mask in logs + plan UI/JSON.



env list/get: masked unless --reveal.



env set: write file; insert audit event in SQLite (actor = OS user, job, key, timestamp; not the secret value).

4.6 plan_id

Plain content hash of: normalized job.yaml + project_version + env profile name + resolved env key/values. Not secret.
run --plan-id recomputes; mismatch → exit 11.
Scheduled: plan+run in same process (no GUI TOCTOU).

4.7 Plan / confirmation JSON

Includes: job, env, machine, steps+commands+code pages+detach, workdir, timeout, env validation with masked values, active runs, project_version, plan_id.
GUI confirm; Prod types env name; then run --plan-id … --trigger gui --yes as separate process.
Scheduled: plan JSON in log header.

4.8 SQLite (sketch)

CREATE TABLE runs (
  id TEXT PRIMARY KEY,
  short_id TEXT NOT NULL,
  job_id TEXT NOT NULL,
  env_profile TEXT NOT NULL,
  trigger TEXT NOT NULL,          -- manual|scheduled|gui
  actor TEXT,
  status TEXT NOT NULL,           -- running|succeeded|failed|timeout|cancelled|skipped
  skip_reason TEXT,               -- lock|promote_lock|...
  exit_code INTEGER,
  plan_id TEXT,
  started_at_utc TEXT NOT NULL,
  started_local TEXT NOT NULL,
  started_offset TEXT NOT NULL,
  ended_at_utc TEXT,
  ended_local TEXT,
  ended_offset TEXT,
  log_path TEXT,
  project_version TEXT,
  plan_json TEXT,
  error_summary TEXT,
  pid INTEGER
);

CREATE TABLE run_steps (
  run_id TEXT NOT NULL,
  step_index INTEGER NOT NULL,
  name TEXT NOT NULL,
  command TEXT NOT NULL,
  output_code_page TEXT,
  detach INTEGER NOT NULL DEFAULT 0,
  exit_code INTEGER,
  duration_ms INTEGER,
  started_local TEXT,
  ended_local TEXT,
  PRIMARY KEY (run_id, step_index)
);

CREATE TABLE env_audit (
  id TEXT PRIMARY KEY,
  at_utc TEXT NOT NULL,
  at_local TEXT NOT NULL,
  at_offset TEXT NOT NULL,
  actor TEXT,
  job_id TEXT NOT NULL,
  env_profile TEXT NOT NULL,
  key TEXT NOT NULL
  -- value NEVER stored
);

CREATE TABLE deployments ( /* as v3 */ );

4.9 Locks





Job lock: create-exclusive data/locks/<job-id>.lock with {run_id,pid,started_at_utc,...}; stale = PID dead or start-time mismatch.



_promote.lock: promote acquires first, then checks active runs; run waits while promote lock held.

Wait defaults (machine.yaml overrides):







Trigger



Default wait on _promote.lock





manual / gui



30s





scheduled



5 minutes

Then exit 12 if still held → record skipped + reason promote_lock + notify if enabled.
Exit 5 (job lock) on scheduled → same: skipped, reason job_lock, notify if enabled.

4.10 Cancel / Kill





GUI Running = status=running from SQLite.



Kill → autodeployer cancel <run-id>.



Core: resolve PID from lock/DB → Windows taskkill /PID <pid> /T /F (or Job Object kill for non-detached tree); Unix process-group kill.



Set status=cancelled, exit_code path 7, end timestamps, clear job lock, finalize --json-meta / row immediately after kill (no side-channel cancel file).



Exit 13 if not running.

4.11 Encoding





Per step output_code_page; default job; else machine.



auto = machine default code page only; no content sniffing.



Decode on line boundaries; hold incomplete multibyte across read chunks; flush incomplete at step end as replacement chars.



Logs always UTF-8 on disk.

4.12 Promote





Exclusive _promote.lock.



No active runs (else release, exit 8).



Hash old/new; rename project→project.bak-…; move new in; retry/backoff on sharing violation; rollback bak on failure.



Write project-version.json + deployment row; schedule reconcile report.



Never touch data/.

4.13 Scheduling

OS Task Scheduler / cron → CLI.
Missed while off: skip (no backfill).
Overlap: exit 5 → skipped row.
schedule reconcile for drift.

4.14 Notify (Phase 2)

machine.yaml webhook and/or SMTP. Fire on failed, timeout, cancelled, and skipped (lock/promote) when enabled.

4.15 Retention / doctor





log_retention_days (default 30); gc; never on promote.  



doctor: read-only (disk local?, sqlite, paths, backup hint).  



doctor --fix: stale running + stale locks.  



Lazy reconcile also in history list and next run.

4.16 machine.yaml (example)

machine_name: uat-app-01
allowed_envs: [uat]
default_env: uat
default_output_code_page: cp932   # used when auto
env_file_encoding: utf-8
log_retention_days: 30
allow_example_env: false
promote_wait_seconds_manual: 30
promote_wait_seconds_scheduled: 300
secret_min_mask_length: 4
notify:
  on_failure: true
  on_skipped: true
  webhook_url: ""



5. Edge-case handling







Edge



Handling





Promote vs run



_promote.lock first; runs wait then skip/12





Rename in use



Retry; rollback; exit 8





Multibyte split



Line-boundary holdback





auto encoding



Machine CP only





Detach + CLI exit



No KILL_ON_JOB_CLOSE / breakaway; child lives





Timeout/cancel



Tree-kill non-detached tree





GUI close



Run process keeps going





plan_id mismatch



Exit 11





Scheduled lock/promote



History skipped + notify





Network data/



doctor fails; block runs in strict mode





Short secrets



No mask if len < min





Path escape



Exit 3





Stale running



doctor --fix / history list / next run



6. Phased build plan

Phase 0 — Spike (1–2 weeks) — see B below

Pass/fail on JP Windows: (a) tree-kill + detach survival, (b) encoding, (c) bat exit codes. Fallbacks if fail: taskkill /T primary; wrapper cmd for exits; encoding module isolation.

Phase 1 — Headless core

Discover, validate, plan/plan_id, run, cancel, env CLI+audit, locks, promote-wait behavior hooks (lock file react), encoding, AD_*, json-meta, NDJSON flush-per-line, SQLite, doctor / doctor --fix, samples.

Exit: CLI-only operable; invariants tested without GUI.

Phase 2 — Schedule + notify

OS tasks, reconcile, skipped history+notify, gc.

Phase 3 — Promote

Full promote tool + race/rename tests.

Phase 4 — Optional GUI

Plan confirm, start detached run process, Running+Kill, log tail, History, env via CLI. Stack chosen here.



7. Test plan







Suite



Cases





Detach invariant



Start run from “GUI simulator”; kill parent; run PID still alive; log grows





Cancel



cancel → tree dead; status cancelled; lock gone; exit 7 path





Detach step



After normal CLI exit, detached child alive; after cancel, non-detach tree dead





Encoding



Per-step cp932/utf-8; chunk split multibyte; auto=machine CP





plan_id



Env change → exit 11





Locks / promote wait



Manual 30s / scheduled 5min; skipped rows





Promote



Refuse if running; rollback on share violation





Env



mask default; --reveal; env set audit without value





Path / AD_*



Reject ..; output dir created





doctor



read-only no writes; --fix clears stale





NDJSON



Reader only sees complete lines





Notify P2



skipped + failed webhooks



8. Success criteria





New job = folder + job.yaml + scripts only  



Same job across envs via data/env only  



Scheduled run from 2 days ago in History with full log  



Promote never touches data/; refuses when running  



Missing env blocks before start  



CLI-only Prod viable; GUI optional  



Closing GUI never kills runs  



Kill via cancel works from Running list  



Phase 0 encoding/kill/exit criteria passed (or documented fallback engaged)  



v1 secrets plain files (accepted); masking still works



A. Deferred (post-v1)







Item



Reason deferred





Dedicated scheduler service account



v1 uses same account as ops; simplifies install





ACL lockdown on data/env/



Explicit: operators may read plain secrets in v1





DPAPI / OS keystore



Hardening after workflow proven





plan_id HMAC / signed plans



Plain hash enough for TOCTOU binding





Env set permission / RBAC



All operators with filesystem access can set; audit only





Git-based promote



Folder replace first





Hot-swap project/ while running



Unsafe; promote lock stays





Central fleet / remote agent



Per-machine install





Parameters / retries in job.yaml



Schema reserved later





Full CI/CD product



Out of scope





Automated data/ backup agent



IT/manual in v1





Secret isolation (ops cannot read)



Conflicts with plain-file v1 decision



B. Phase 0 spike checklist

Duration: 1–2 weeks · Machine: Japanese-locale Windows · Owner: core engineer · Output: spike report (template below) checked into repo.

B.1 Fixtures to create







Fixture



Description





chain_sleep.bat → SleepJava



Bat starts java SleepJava (sleep 5–10 min); used for kill tests





detach_start.bat



Starts long-lived child via start / breakaway pattern; prints child PID to file





out_cp932.bat



Writes Japanese text using OEM/CP932 console encoding





out_utf8.bat or small Java



Writes UTF-8 Japanese lines





mixed_job



Step1 cp932 fixture, step2 utf-8 fixture





chunk_split.bin / harness



Feed decoder bytes that split a multibyte sequence across two reads





nested_nocall.bat



Calls another bat without call; demonstrates exit-code pitfall





nested_call.bat



Correct call + exit /b





fail_step1.bat



Exits non-zero for fail-fast

B.2 Exact test cases







ID



Test



Pass criteria





P0-a1



Timeout kills bat→java tree



After timeout, java PID gone; no orphan java





P0-a2



Cancel kills tree



Same as a1 via cancel API/taskkill /T /F





P0-a3



Normal success does not tree-kill



After successful short job, no erroneous kills





P0-a4



Detach survive CLI exit



Job Object without KILL_ON_JOB_CLOSE or breakaway: after CLI exits 0, detached child still alive





P0-a5



Cancel does not require killing breakaway service if out of tree—but non-detached tree dead



Document boundary; non-detach PIDs dead





P0-b1



CP932 Japanese → UTF-8 log



Golden file match; readable 日本語





P0-b2



UTF-8 step on JP machine



Golden match





P0-b3



Sequential cp932 then utf-8



Both segments correct in one log





P0-b4



Multibyte split across chunks



No mojibake; holdback works





P0-b5



auto uses machine CP



Equals explicit machine default; no sniffing





P0-c1



Fail-fast



Step1 fail → step2 not run





P0-c2



Nested bat without call



Document observed exit code; lint warning justified





P0-c3



Nested with call + exit /b



Go parent sees non-zero correctly

B.3 Fallbacks if fail







Failure



Change





Job Object unreliable



Primary kill = taskkill /PID /T /F; Job Object optional





Detach still dies



Mandate CREATE_BREAKAWAY_FROM_JOB / service pattern in docs + helper





Encoding hard



Isolate decoder package; force explicit code pages (disallow auto except machine default)





Exit codes lie



Ship wrapper .cmd template AutoDeployer generates around user scripts

B.4 Report template

# Phase 0 Spike Report
Date / machine / Windows version / locale / code page:
Go version:

## Results
| ID | Result (PASS/FAIL) | Notes / PIDs / log paths |
|----|--------------------|---------------------------|
| P0-a1 |  |  |
| ... |  |  |

## Detach mechanism used
(KILL_ON_JOB_CLOSE off / BREAKAWAY / other):

## Kill mechanism used
(Job Object / taskkill /T /F / both):

## Fallbacks engaged
(none / list):

## Go / no-go for Phase 1
Decision:
Risks remaining:



C. Decisions log







Decision



Owner



Status





Core = Go headless CLI; GUI optional thin client



Product



Locked





Layout app/ project/ data/; promote replaces project/ only



Product



Locked





First install excludes source data/



Product



Locked





SQLite history; logs on disk



Product



Locked





OS scheduler invokes CLI



Product



Locked





Mandatory required_env validation



Product



Locked





Allowed envs per machine



Product



Locked





v1 secrets = plain files; operators may read



Product



Locked





Mask secrets in logs/plan; --reveal on env get/list



Product



Locked





plan_id = plain content hash



Product



Locked





Per-step encoding; line-boundary decode; auto=machine CP



Product



Locked





Tree-kill only timeout/cancel; detach without KILL_ON_JOB_CLOSE



Product



Locked





Run = separate OS process; GUI close never kills



Product



Locked





Cancel via CLI + Running/Kill UI



Product



Locked





Promote lock + rename retry/rollback



Product



Locked





Promote wait 30s manual / 5min scheduled



Product



Locked





NDJSON flush-per-line



Product



Locked





doctor read-only; --fix repairs



Product



Locked





Skipped runs recorded + notify (P2)



Product



Locked





data/ local disk; backup manual/IT



Product



Locked





Schedules = OS TZ; JP machines



Product



Locked





Phase 0 before Phase 1



Product



Locked





Failure notify in Phase 2



Product



Locked





GUI stack (Tauri vs Electron)



Eng



Open → default Tauri 2 + React at Phase 4





Sample real bats



Ops



Open → default stubs until pasted





Console encoding samples



Ops



Open → default synthetic fixtures





Exact Windows versions



Ops



Open → default Win10/11 + Server 2019/2022





Task Scheduler allowed



Ops



Open → default yes; if no → Windows Service wrapper fallback





Air-gap / AV allowlist



Ops



Open → default static binary + IT allowlist as needed





Scale numbers



Ops



Open → default ≤50/10/20/200





Notify channel (webhook vs SMTP)



Ops



Open → default webhook URL in machine.yaml





Phase 0 machine access



Eng/IT



Open → default use any JP-locale Win VM; block only if zero Win access (then spike on available Win)



v4 is final for planning. Engineers start at Phase 0 using checklist B. No further planning rounds required unless a Phase 0 FAIL forces a documented fallback from B.3.