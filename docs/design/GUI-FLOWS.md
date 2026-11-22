# GUI information architecture and flows

Thin-client GUI over the AutoDeployer CLI. The GUI **never** spawns `.bat` / `.sh` / Java itself.  
Frame names below are the inventory for Figma (issue #4). Vision: `docs/design/VISION.md`. Commands: `docs/design/CONTRACT.md`.

## Invariant

> Every run is a **separate OS process**. Closing or crashing the GUI **never** stops a run. The only intentional stop is **Kill** → `autodeployer cancel <run-id>`. The log file under `data/logs/` is the source of truth.

---

## Screen inventory and navigation

```text
┌─────────────────────────────────────────────────────────────┐
│ AutoDeployer   [machine]   env chip: Dev|UAT|Prod (allowed) │
├──────────┬──────────────────────────────────────────────────┤
│ Jobs     │  main content                                    │
│ Running  │                                                  │
│ History  │                                                  │
│ Env      │                                                  │
│ Schedules│                                                  │
│ Deploys  │                                                  │
└──────────┴──────────────────────────────────────────────────┘
```

| Frame name (Figma) | Purpose | Primary CLI |
|--------------------|---------|-------------|
| `Jobs` | Job list + README preview + Run | `jobs list`, `jobs show`, then `plan` |
| `Plan confirmation` | Pre-run summary (masked secrets) | `plan … --json` |
| `Prod confirmation` | Type env name before run | (GUI only; then `run`) |
| `Running` | Live `status=running` + Kill | `history list`, `cancel` |
| `Live log` | Tail current or selected run | `logs tail` / read log file |
| `History and log` | Past runs + open log | `history list`, `history show` |
| `Env editor` | View/edit `data/env` via CLI | `env list\|get\|set` |
| `Schedules` | CRUD + reconcile drift | `schedule *`, `schedule reconcile` |
| `Deployments` | Promote results / versions | `promote` (ops), `history`/deployments query |

Nav is always available; modal overlays: Plan confirmation, Prod confirmation.

---

## States (every screen)

| State | Meaning | Typical CLI signal |
|-------|---------|-------------------|
| empty | No jobs / no runs / no schedules | empty JSON array |
| loading | Waiting on CLI | subprocess in flight |
| error | CLI failed (usage, internal, etc.) | exit 2, 3, 10 + stderr |
| running | Active run(s) | `status=running` |
| skipped | Run did not execute | `status=skipped`, `skip_reason` |
| blocked by validation | Cannot start | exit **4** env; exit **9** env not allowed |
| plan_id mismatch | Stale confirm | exit **11** |
| promote lock | Wait exhausted | exit **12** |
| job locked | Concurrent denied | exit **5** |

---

## Flows

### 1. Run with confirmation

1. **Jobs** — select job; choose env from machine allow-list; click **Run**.  
2. GUI: `autodeployer plan <job> --env <p> --json` → show **Plan confirmation** (steps, workdir, timeout, masked env, active runs, project version, `plan_id`).  
3. User confirms.  
4. If profile is `prod` → go to flow 2; else continue.  
5. GUI starts a **detached** OS process:  
   `autodeployer run <job> --env <p> --trigger gui --plan-id <id> --yes --json-meta <file> [--events <file>]`  
6. Navigate to **Running** / **Live log** (tail log file). Closing the GUI leaves the process running.

### 2. Prod type-the-env-name confirmation

1. After plan confirm, if `env_profile == prod` (or machine marks profile as “strict”), show **Prod confirmation**.  
2. User must type the exact env name (`prod`). Button disabled until match.  
3. Then start `run` as in flow 1 step 5.  
4. Visual chrome for Prod is distinct (see D4).

### 3. Kill from Running

1. **Running** lists rows where `status=running` (`history list --json` filtered, or equivalent).  
2. User clicks **Kill** on a row.  
3. GUI: `autodeployer cancel <run-id> --json`.  
4. On success, row becomes `cancelled` (exit path 7); lock cleared.  
5. Closing the GUI does **not** cancel.

### 4. History two days later

1. Open **History and log**.  
2. `autodeployer history list --limit 100 --json`.  
3. User opens a scheduled run from ~2 days ago → `history show` + open `log_path`.  
4. Log viewer renders UTF-8 including Japanese sample lines.  
5. Skipped runs show status + `skip_reason` (`job_lock` / `promote_lock`).

### 5. Edit env (mask + reveal)

1. **Env editor** — pick job + profile.  
2. `autodeployer env list --job <id> --env <p> --json` (masked).  
3. **Reveal** → `env get … --reveal` (or list with `--reveal`) after explicit toggle.  
4. Save key → `autodeployer env set --job <id> --env <p> --key K --value V` (audited; value not stored in audit).  
5. Missing required keys surface on next **plan** as exit 4 / `can_execute=false`.

### 6. Create schedule and see reconcile drift

1. **Schedules** — create: `schedule set` (job, env, cron, enabled).  
2. Later, OS task deleted manually or renamed → **Reconcile**: `schedule reconcile --json`.  
3. UI shows drift; with `--strict`, exit 14.  
4. User repairs via `schedule set` / `delete`.

### 7. View a promote result

1. **Deployments** — show last `project-version.json` + deployment history (from DB / doctor).  
2. Ops may run promote from CLI; GUI can offer “Show last promote” and orphan schedule warnings from promote JSON.  
3. Promote while running is refused (exit 8) — surface as error state; never hot-swap.

---

## Action → CLI map (complete)

| UI action | CLI |
|-----------|-----|
| Refresh jobs | `jobs list --json` |
| Open job README | `jobs show <id> --json` (+ read README path) |
| Validate job | `schema validate <id> --json` |
| Plan before run | `plan <id> --env <p> --json` |
| Start run | `run <id> --env <p> --trigger gui --plan-id <id> --yes --json-meta <f>` |
| List running | `history list --json` (filter `running`) |
| Kill | `cancel <run-id> --json` |
| Tail log | `logs tail <run-id>` or read file |
| History | `history list` / `history show` |
| Env list/get/set | `env list\|get\|set …` |
| Schedules | `schedule list\|set\|delete` |
| Reconcile | `schedule reconcile --json` |
| Doctor / backup hint | `doctor --json` |
| Fix stale | `doctor --fix --json` |

---

## Screen details

### Jobs
- List id, name, short description.  
- Detail: README markdown preview, steps summary, required_env.  
- Empty: “No jobs in project/jobs — add a folder + job.yaml”.  
- Error: schema exit 3 message readable.

### Plan confirmation
- Renders `PlanResult` fields; secrets only `***`.  
- Disable Confirm if `can_execute=false` or env validation not ok.  
- Show active runs warning.

### Prod confirmation
- Distinct danger styling; type-env gate.

### Running
- Columns: job, env, started (canonical local+offset), run_id short, Kill.  
- Empty: “No runs in progress”.

### Live log / History and log
- Monospace; Japanese sample: `日本語テスト_utf8`.  
- Status badges: succeeded / failed / timeout / cancelled / skipped / running.

### Env editor
- Masked by default; Reveal toggle; never show secrets on Plan screen.

### Schedules
- Table + drift panel after reconcile.

### Deployments
- Version hash, time, actor, status; link to orphan schedule report text.

---

## Error callouts (must be designed)

| Exit | User-facing summary |
|------|---------------------|
| 4 | “Missing or empty required environment keys: …” — fix in Env editor |
| 5 | “Job already running” / History may show skipped for schedules |
| 11 | “Plan expired or files changed — run Plan again” |
| 12 | “Promote in progress — try again” / skipped for schedules |
| 9 | “This machine does not allow that environment” |

---

## Out of scope here
Pixels, colors, Figma components — issue #4.  
GUI implementation — Phase 4 / issue #17.
