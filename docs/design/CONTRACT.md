# AutoDeployer — CLI contract

Contract between the core CLI and any client (GUI, scripts, Task Scheduler).  
**Phase 1 code starts only after this document and its schemas are approved (G1).**

Aligned with `docs/PLAN-v4.md` §§3.3, 4.2–4.16 and `internal/exitcode`.

## Global flags

| Flag | Meaning |
|------|---------|
| `--root <path>` | Install root (`AutoDeployer/`). Default: discover from binary location or cwd. |
| `--json` | Machine-readable JSON on stdout where supported. |
| `--no-color` | Disable ANSI colors. |
| `--quiet` | Suppress human log tee on stdout (run still writes the log file). |

Binary name: `autodeployer` (`autodeployer.exe` on Windows).

---

## Exit codes

Must match `internal/exitcode/exitcode.go` exactly. **Do not renumber.**

| Code | Constant | Meaning |
|------|----------|---------|
| 0 | `OK` | Success |
| 1 | `JobFailed` | Job ran but a step failed |
| 2 | `Usage` | Usage / CLI error |
| 3 | `Schema` | job.yaml schema, discovery, or path-safety error |
| 4 | `EnvValidation` | Required env missing or empty |
| 5 | `JobLocked` | Job lock held; concurrent run denied |
| 6 | `Timeout` | Timed out and tree-killed |
| 7 | `Cancelled` | Cancelled and tree-killed |
| 8 | `PromoteFailed` | Promote refused or failed after checks |
| 9 | `EnvNotAllowed` | Env profile not allowed on this machine |
| 10 | `Internal` | Unexpected internal error |
| 11 | `PlanMismatch` | `plan_id` no longer matches current state |
| 12 | `PromoteLockHeld` | Promote lock held; wait exhausted |
| 13 | `CancelNotFound` | Cancel target not found or not running |
| 14 | `ScheduleDrift` | Schedule reconcile drift (with `--strict`) |

Mismatch check (this PR): table ↔ `internal/exitcode/exitcode.go` — **identical**.

---

## Commands

### `version`

**Purpose:** Print binary and contract/schema version.  
**Flags:** `--json`  
**Stdout (text):** human version line.  
**Stdout (`--json`):**

```json
{ "version": "0.0.0-dev", "schema_version": 1, "contract": "v4" }
```

**Exit:** 0, 2, 10

**Example:** `autodeployer version --json`

---

### `doctor`

**Purpose:** Read-only diagnostics (paths, local-disk check for `data/`, sqlite openable, encoding default, ACL hints, backup hint). Does **not** mutate state.  
**Flags:** `--json`  
**Stdout (`--json`):** object with `ok`, `checks[]`, `data_dir`, `backup_hint`.  
**Exit:** 0 if usable; non-zero if fatal misconfig (e.g. `data/` on network share) → prefer 10 or 3 as documented in implementation notes; default **10** for unusable install.

**Example:** `autodeployer doctor --json`

---

### `doctor --fix`

**Purpose:** Reconcile stale `running` rows and stale locks (PID dead / `pid_start` mismatch).  
**Flags:** `--json`  
**Stdout (`--json`):** `{ "fixed_runs": [...], "cleared_locks": [...] }`  
**Exit:** 0, 2, 10

**Example:** `autodeployer doctor --fix --json`

---

### `jobs list`

**Purpose:** Discover `project/jobs/*/job.yaml`.  
**Flags:** `--json`  
**Stdout (`--json`):** array of `{ "id", "name", "description", "path" }`.  
**Exit:** 0, 2, 3, 10

**Example:** `autodeployer jobs list --json`

---

### `jobs show <job-id>`

**Purpose:** Show one job manifest summary + README path.  
**Flags:** `--json`  
**Stdout (`--json`):** job summary including steps (commands only), `required_env`, paths.  
**Exit:** 0, 2, 3, 10

**Example:** `autodeployer jobs show s3-pull-and-java --json`

---

### `schema validate [job-id|all]`

**Purpose:** Validate job.yaml schema, path safety, warn on nested bat without `call`, warn on detectable writes into `project/`.  
**Flags:** `--json`  
**Stdout (`--json`):** `{ "ok": bool, "errors": [...], "warnings": [...] }`  
**Exit:** 0 if no errors (warnings allowed); **3** if errors; 2, 10

**Example:** `autodeployer schema validate all --json`

---

### `plan <job-id> --env <profile>`

**Purpose:** Pre-flight confirmation payload; **does not execute**.  
**Flags:** `--env` (required), `--json`  
**Stdout (`--json`):** `PlanResult` — see `docs/contract/plan_result.schema.json`. Includes `plan_id`.  
**Exit:** 0 if plan computable (even when `can_execute=false`); **4** if env invalid; **9** if env not allowed; 2, 3, 10

**Example:** `autodeployer plan s3-pull-and-java --env uat --json`

---

### `run <job-id> --env <profile> --trigger <manual|scheduled|gui> --plan-id <id> --yes --json-meta <file>`

**Purpose:** Validate and execute iff `plan_id` matches.  
**Flags:**

| Flag | Required | Notes |
|------|----------|--------|
| `--env` | yes | Profile name |
| `--trigger` | yes | `manual` \| `scheduled` \| `gui` |
| `--plan-id` | yes for gui/manual after confirm; scheduled may compute in-process | Mismatch → exit 11 |
| `--yes` | yes for non-interactive | GUI/scheduled always pass |
| `--json-meta <file>` | recommended | Terminal `RunResult` written here |
| `--events <ndjson>` | optional | Append-only NDJSON; one complete line per write; flush per line |

**Stdout:** human log tee (unless `--quiet`).  
**Log file:** UTF-8 under `data/logs/...` (source of truth).  
**Exit:** 0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12  

Scheduled lock/promote failures: exit 5 or 12 and history row `status=skipped`.

**Example:**

```bash
autodeployer run s3-pull-and-java --env prod --trigger gui --plan-id abc123 --yes \
  --json-meta data/output/s3-pull-and-java/RUNID/meta.json \
  --events data/output/s3-pull-and-java/RUNID/events.ndjson
```

**Invariant:** each `run` is its own OS process; closing a GUI parent never kills it.

---

### `cancel <run-id>`

**Purpose:** Tree-kill a running job (`taskkill /T /F` on Windows / process-group on Unix), mark `cancelled`, clear job lock, finalize row and `--json-meta` immediately.  
**Flags:** `--json`  
**Exit:** 0 (kill issued), **13** if not found/not running, 2, 10

**Example:** `autodeployer cancel 01HXYZ... --json`

---

### `history list`

**Purpose:** Query SQLite history (lazy-reconciles stale `running`).  
**Flags:** `--job <id>`, `--limit <n>`, `--json`  
**Stdout (`--json`):** array of run summaries.  
**Exit:** 0, 2, 10

---

### `history show <run-id>`

**Purpose:** One run + log path (+ steps).  
**Flags:** `--json`  
**Exit:** 0, 2, 10 (not found → 2 or 13; prefer **2** for show)

---

### `logs tail <run-id>`

**Purpose:** Tail the UTF-8 log file.  
**Stdout:** text lines.  
**Exit:** 0, 2, 10

---

### `env list --job <id>`

**Purpose:** List keys for a job’s env files / selected profile.  
**Flags:** `--job` (required), `--env <profile>` optional filter, `--reveal`, `--json`  
**Stdout:** values masked unless `--reveal`.  
**Exit:** 0, 2, 3, 10

---

### `env get --job <id> --env <profile> --key <K>`

**Purpose:** Get one value (masked unless `--reveal`).  
**Exit:** 0, 2, 3, 10

---

### `env set --job <id> --env <profile> --key <K> --value <V>`

**Purpose:** Write `data/env/<job>/.env.<profile>`; append `env_audit` row (actor, key, time — **never** the secret value).  
**Exit:** 0, 2, 3, 9, 10

---

### `schedule list|set|delete`

**Purpose:** CRUD `data/schedules.json` and project OS Task Scheduler / cron entry.  
**Flags:** `--json`; `set` takes job, env, cron/expression or Windows schedule fields (implementation detail in Phase 2).  
**Exit:** 0, 2, 3, 9, 10

---

### `schedule reconcile`

**Purpose:** Detect drift between `schedules.json` and OS tasks.  
**Flags:** `--json`, `--strict` (exit **14** on drift)  
**Exit:** 0, 2, 10, 14

---

### `promote --from <new-project-dir>`

**Purpose:** Acquire `_promote.lock`, refuse if active runs, replace `project/` with rename retry/rollback, write version + deployment row, report schedule orphans. Never touches `data/`.  
**Flags:** `--json`  
**Exit:** 0, 2, 8, 10

---

### `gc`

**Purpose:** Apply `log_retention_days`; never deletes on promote.  
**Flags:** `--json`  
**Exit:** 0, 2, 10

---

## JSON schemas and examples

| Schema | Path | Example |
|--------|------|---------|
| PlanResult | `docs/contract/plan_result.schema.json` | `docs/contract/examples/plan_result.json` |
| RunResult | `docs/contract/run_result.schema.json` | `docs/contract/examples/run_result.json` |
| NDJSON event | `docs/contract/ndjson_event.schema.json` | `docs/contract/examples/events.ndjson` |
| job.yaml | `docs/contract/job.schema.json` | `docs/contract/examples/job.yaml` |
| machine.yaml | `docs/contract/machine.schema.json` | `docs/contract/examples/machine.yaml` |

Validate:

```bash
docs/contract/validate_examples.sh
```

Schemas are JSON Schema **draft 2020-12**. `job.yaml` / `machine.yaml` examples are YAML; the validator loads YAML then validates as JSON data.

---

## File formats

### Layout

```text
<root>/
  app/autodeployer[.exe]
  project/jobs/<job-id>/{job.yaml,README.md,scripts/,env/*.example}
  project/shared/
  data/                         # LOCAL DISK only; never promote source
    machine.yaml
    env/<job-id>/.env.<profile>
    output/<job-id>/<run-id>/
    logs/<job-id>/
    history.db
    schedules.json
    locks/_promote.lock
    locks/<job-id>.lock
    project-version.json
```

### Canonical time

One format: `YYYY-MM-DDTHH:mm:ss±HH:MM` (example `2026-11-20T02:00:00+09:00`).

| Consumer | Derivation |
|----------|------------|
| DB | UTC RFC3339 + local + offset columns |
| Log header | canonical |
| Log filename | `YYYYMMDD_HHmmss_p0900_<shortRunId8>_<trigger>.log` (`+09:00`→`p0900`, `-05:00`→`m0500`) |
| Schedules | OS local timezone wall clock |

History sort order: **UTC only**.

### Lock file JSON (`data/locks/<job-id>.lock`)

Create-exclusive. Stale if PID dead **or** OS process start time ≠ `pid_start`.

```json
{
  "job_id": "s3-pull-and-java",
  "run_id": "01JBEXAMPLE000000000000000",
  "pid": 1234,
  "pid_start": "2026-11-20T02:00:01+09:00",
  "started_at_utc": "2026-11-19T17:00:01Z",
  "started_local": "2026-11-20T02:00:01",
  "started_offset": "+09:00",
  "actor": "SCHEDULE:nightly-s3"
}
```

`_promote.lock` uses the same shape with `job_id` = `"_promote"` (or omit run fields as applicable); presence alone blocks/waits runs.

### `schedules.json`

```json
{
  "version": 1,
  "schedules": [
    {
      "id": "nightly-s3",
      "job_id": "s3-pull-and-java",
      "env_profile": "prod",
      "cron": "0 2 * * *",
      "enabled": true,
      "os_task_name": "AutoDeployer-nightly-s3"
    }
  ]
}
```

### `project-version.json`

```json
{
  "version": "a1b2c3d4e5f67890",
  "computed_at": "2026-11-20T02:00:00+09:00",
  "source_path": "D:\\incoming\\project"
}
```

### Log header

Every run (including scheduled) starts the UTF-8 log with:

```text
=== AutoDeployer run ===
run_id: ...
short_id: ...
job: s3-pull-and-java
env: prod
trigger: scheduled
machine: prod-app-01
project_version: ...
plan_id: ...
started: 2026-11-20T02:00:00+09:00
plan_json: {...}
=== stdout/stderr follow (UTF-8) ===
```

### Injected environment

| Var | Meaning |
|-----|---------|
| `AD_ROOT` | Install root |
| `AD_DATA_DIR` | `data/` |
| `AD_JOB_OUTPUT_DIR` | `data/output/<job-id>/<run-id>/` |
| `AD_RUN_ID` | Run id |
| `AD_ENV` | Profile |
| `AD_JOB_ID` | Job id |

Scripts write under `AD_JOB_OUTPUT_DIR` only—not into `project/`.

### Promote wait (`machine.yaml`)

| Trigger | Default wait on `_promote.lock` |
|---------|----------------------------------|
| manual / gui | 30s |
| scheduled | 300s (5 min) |

Then exit 12 → history `skipped` / `skip_reason=promote_lock`.

---

## SQLite (`data/history.db`)

WAL mode; `data/` must be on a **local** disk.

### `runs`

| Column | Type | Notes |
|--------|------|--------|
| id | TEXT PK | ULID/UUID |
| short_id | TEXT | 8-char filename suffix |
| job_id | TEXT | |
| env_profile | TEXT | |
| trigger | TEXT | `manual` \| `scheduled` \| `gui` |
| actor | TEXT | OS user or schedule id |
| status | TEXT | `running` \| `succeeded` \| `failed` \| `timeout` \| `cancelled` \| `skipped` |
| skip_reason | TEXT | e.g. `job_lock`, `promote_lock` |
| exit_code | INTEGER | |
| plan_id | TEXT | |
| started_at_utc | TEXT | RFC3339 UTC |
| started_local | TEXT | |
| started_offset | TEXT | e.g. `+09:00` |
| ended_at_utc | TEXT | |
| ended_local | TEXT | |
| ended_offset | TEXT | |
| log_path | TEXT | |
| project_version | TEXT | |
| plan_json | TEXT | snapshot |
| error_summary | TEXT | |
| pid | INTEGER | |

### `run_steps`

| Column | Type |
|--------|------|
| run_id | TEXT |
| step_index | INTEGER |
| name | TEXT |
| command | TEXT |
| output_code_page | TEXT |
| detach | INTEGER (0/1) |
| exit_code | INTEGER |
| duration_ms | INTEGER |
| started_local | TEXT |
| ended_local | TEXT |
| PRIMARY KEY (run_id, step_index) | |

### `env_audit`

| Column | Type | Notes |
|--------|------|--------|
| id | TEXT PK | |
| at_utc / at_local / at_offset | TEXT | |
| actor | TEXT | |
| job_id | TEXT | |
| env_profile | TEXT | |
| key | TEXT | **value never stored** |

### `deployments`

| Column | Type |
|--------|------|
| id | TEXT PK |
| deployed_at_utc / deployed_local / deployed_offset | TEXT |
| old_version | TEXT |
| new_version | TEXT |
| source_path | TEXT |
| actor | TEXT |
| status | TEXT | `succeeded` \| `failed` |

### `meta`

| key | value |
|-----|-------|
| schema_version | `1` |

---

## NDJSON events (`--events`)

Append-only file. Writer: one complete JSON object + `\n`, then flush. Readers must ignore incomplete trailing lines.

Types: `run_start`, `step_start`, `step_end`, `run_end`, `log_line` (optional).  
Schema: `docs/contract/ndjson_event.schema.json`.

---

## Open questions (with defaults)

| # | Question | Default |
|---|----------|---------|
| Q1 | Exact Windows Task Scheduler fields for `schedule set` | Store cron string + `os_task_name`; Phase 2 maps cron → daily/weekly triggers |
| Q2 | `doctor` fatal exit when `data/` is on UNC | Exit **10**; refuse `run` as well |
| Q3 | `history show` when run_id missing | Exit **2** (usage/not found) |
| Q4 | Hash algorithm for `plan_id` / project version | SHA-256 hex, truncate display to 16 chars where helpful; full hash in fields |
| Q5 | `log_line` events optional or always on when `--events` set | Optional; default **omit** log_line to keep event files small; GUI tails the log file |
| Q6 | Detached child survival on cancel | Non-detached tree killed; breakaway/service children may survive (Phase 0 verifies; contract documents the boundary) |
| Q7 | GUI stack | Tauri 2 + React (Phase 4) |
| Q8 | `pid_start` source on Windows | Process creation time via Win32; compared on stale check |

Changing a default after G1 requires owner confirmation.
