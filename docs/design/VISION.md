# AutoDeployer — Design vision

Shared vision for every later design and code decision. Locked decisions live in `docs/PLAN-v4.md` section C; this brief explains *why* and *for whom*.

## Users and contexts

| Who | Where | What they need |
|-----|--------|----------------|
| Dev operator | Laptop (often Mac or Windows) | Run the same jobs as UAT/Prod with a **Dev** env profile; edit scripts and `job.yaml` in any IDE; see live logs; never accidentally select Prod |
| UAT operator | UAT application server | Validate promote-ready jobs with UAT secrets; schedule test runs; inspect History after overnight jobs |
| Prod operator | Production server | Confirm before every manual run (type the env name); trust that scheduled jobs leave durable logs; promote `project/` without wiping history or secrets |
| Unattended schedule | No human watching | Task Scheduler / cron calls the CLI; same validation, logging, and history as a manual run; failures and skips must be visible later in History (and notified in Phase 2) |

Multiple people may use the same machine install. Collaboration is file-based (`project/` + plain `data/env/`), not a central fleet console.

## The problem (operators' words)

- Maintenance `.bat` / shell scripts are **scattered**; nobody knows which copy is current.
- There is **no shared history**: “Did the S3 pull run last night? What did it print?”
- **Env mix-ups** are easy: Dev hostnames and Prod passwords live in the same tribal knowledge.
- **Promote mistakes**: copying a whole tree can wipe machine logs or overwrite Prod `.env` with Dev values—or leave half-updated scripts while a job is running.
- Terminal one-offs (DB backup, export/import, SFTP pull, Tomcat Java invoke) never become a **repeatable, reviewed** runbook.

## Product principles

1. **CLI is the product.** `autodeployer` on disk is enough for Prod. The GUI is an **optional thin client** over the same files and CLI—never a second executor.
2. **One execution path.** Manual, scheduled, and GUI-started runs all call `autodeployer run …` with the same validation, logging, locks, and history.
3. **Safe by default.** Missing/empty `required_env` blocks the run; plan/confirmation before execute; Prod needs typed confirmation in the GUI; promote refuses while jobs run; secrets masked in logs and plan output.
4. **Portable project, local data.** `project/` is replaceable and promotable; `data/` (env, logs, history, schedules) stays on the machine and is never replaced by promote.
5. **Ordinary scripts win.** Jobs are folders + `job.yaml` + `.bat`/`.sh`. No app code change to add a job.
6. **Japanese text is first-class.** Encoding is explicit per step; logs are UTF-8; the GUI log viewer must render Japanese correctly.

## Non-goals for v1

From plan section A (Deferred) and out-of-scope:

| Non-goal | Notes |
|----------|--------|
| Dedicated scheduler service account / ACL lockdown on `data/env/` | v1: plain `.env` files readable by operators |
| DPAPI / OS keystore | Later hardening |
| Signed / HMAC `plan_id` | Plain content hash only |
| Env-set RBAC | Audit `env set` in history; no permission matrix |
| Git-based promote | Folder replace first |
| Hot-swap `project/` while a job runs | Promote lock refuses |
| Central fleet / remote agents | Per-machine install |
| Replacing Tomcat, Java, SFTP, or DB tools | We orchestrate them |
| Full CI/CD product | Ops runbooks, not app pipelines |
| Automated `data/` backup agent | Manual / IT-owned; `doctor` prints the path |
| Parameters / retries in `job.yaml` | Reserved for later |
| GUI as required runtime | CLI-only Prod is valid |

## Success criteria (as demos)

Each item from plan section 8, rewritten as something an owner can watch:

| # | Demo |
|---|------|
| 1 | Add a new job by creating `project/jobs/my-job/` with `job.yaml` + scripts only; `autodeployer jobs list` shows it—no rebuild of the binary. |
| 2 | Run the same job on Dev and UAT changing only `data/env/<job>/.env.*`; scripts and `job.yaml` identical. |
| 3 | Two days after a scheduled run, open History (CLI or GUI) and open the full UTF-8 log for that run. |
| 4 | Run `autodeployer promote --from <new-project>` and show `data/logs` and `data/env` unchanged; with a run active, promote exits refused. |
| 5 | Delete a required env key and run: process never starts; exit code 4. |
| 6 | Start a run from a GUI simulator, kill the GUI process: the run PID and log keep going. |
| 7 | From Running, Kill → `cancel` → process tree dead, status `cancelled`. |
| 8 | On Japanese Windows (Phase 0), a job with cp932 then UTF-8 steps produces a readable UTF-8 log with Japanese text. |
| 9 | Operate Prod for a week using **only** the CLI and Task Scheduler—no GUI installed. |

## Glossary

| Term | Meaning |
|------|---------|
| **Job** | A folder under `project/jobs/<id>/` with `job.yaml`, scripts, README, and env *templates* (`*.example`). |
| **Env profile** | Named environment such as `dev`, `uat`, `prod`. Real values live in `data/env/<job-id>/.env.<profile>`. |
| **Run** | One execution of a job for a profile: its own OS process, log file, and history row. |
| **Plan** | Pre-flight summary (`autodeployer plan`): steps, workdir, timeout, masked env check, active runs, project version—no execute. |
| **plan_id** | Plain content hash of job + project version + resolved env + profile. `run --plan-id` fails (exit 11) if state changed. Not a secret. |
| **Promote** | Replace `project/` with a new tree via `autodeployer promote`; never touches `data/`; refuses if runs are active. |
| **Detach** | A step with `detach: true` may start a long-lived child that **survives** normal CLI exit; timeout/cancel still kill the non-detached tree. |
| **Skipped run** | Scheduled (or waiting) run that did not execute because of job lock (exit 5) or promote lock (exit 12); recorded in history with `status=skipped` and a reason. |

## GUI UX principles

- **Confirm before run.** Always show the plan summary; execution starts only after confirm (CLI: interactive or `--yes` after GUI confirm).
- **Prod is harder to mistake.** Selecting Prod requires typing the environment name; Prod chrome is visually distinct from Dev/UAT (see D4).
- **Closing the GUI never loses or kills a run.** Each run is a separate OS process; the log file on disk is the source of truth; cancel only via Kill → `autodeployer cancel`.
- **Japanese is first-class.** Sample and production logs may contain Japanese; fonts and log views must render them; encoding mistakes are treated as product bugs.
- **Thin client.** Every action maps to a documented CLI command; the GUI never spawns `.bat`/`.sh`/Java itself.
- **Readable failures.** Exit codes and messages are for operators, not only developers (e.g. missing env key named clearly).

## Day in the life (one page)

**Tuesday 02:00 (machine local, Japan).** Task Scheduler starts:

`autodeployer run s3-pull-and-java --env prod --trigger scheduled --yes …`

The job validates Prod env, writes a UTF-8 log under `data/logs/…`, records a history row, finishes (or fails / skips with reason).

**Thursday morning.** An operator opens the GUI (or `autodeployer history list`). They filter or scroll to Wednesday night’s scheduled run for `s3-pull-and-java` / `prod`, open the log, and read the full output—including Japanese lines from a Java step—without asking who was on the server at 02:00. If it had been skipped for a lock, History shows `skipped` and why. If it failed, Phase 2 would already have sent a webhook; either way the evidence is on disk under `data/`, not in someone’s scrollback buffer.

That is the product: **scattered terminal work becomes one catalog, one runner, and durable history—CLI-first, GUI optional.**
