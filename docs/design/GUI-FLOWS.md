# GUI information architecture and flows

Thin-client GUI over the AutoDeployer CLI. The GUI **never** spawns `.bat` / `.sh` / Java itself.  
Vision: `docs/design/VISION.md`. Commands: `docs/design/CONTRACT.md`. Visual: `docs/design/VISUAL.md`.

## Product metaphor

**Each job is a project** (like a GitHub repository):

- Opening a job lands on its **`README.md` homepage** (rendered markdown).
- A **folder tree** shows `job.yaml`, `scripts/`, `env/*.example`, etc.
- Selecting a file opens a **code / markdown viewer** (view-only for scripts and manifests; real editing stays in the operator’s IDE on disk).
- **Schedules**, **Runs**, and **Env** for that job live as tabs inside the job workspace.
- The top-level **Jobs** screen is a **catalog of projects**, with last-run and schedule badges.

Global areas remain: **Running** (Kill), **History** (all jobs), **Deployments**, **Doctor**.

## Invariant

> Every run is a **separate OS process**. Closing or crashing the GUI **never** stops a run. The only intentional stop is **Kill** → `autodeployer cancel <run-id>`. The log file under `data/logs/` is the source of truth.

---

## Navigation map

```text
┌─ AutoDeployer ── machine ── env chip Dev|UAT|Prod ─────────────┐
│ Jobs │ Running │ History │ Deployments │ Doctor                  │
└──────────────────────────────────────────────────────────────────┘

Jobs catalog ──open──► Job workspace
                         Overview │ Files │ Schedules │ Runs │ Env
                         ┌──────┬─────────────────────────────────┐
                         │ tree │  README home  OR  file viewer   │
                         └──────┴─────────────────────────────────┘
```

### Global frames (Figma)

| Frame | Purpose | Backing |
|-------|---------|---------|
| `Jobs catalog` | All jobs as projects: name, blurb, last run, schedule badge | `jobs list`, `history list`, `schedule list` |
| `Running` | Cross-job `status=running` + Kill | `history list`, `cancel` |
| `History and log` | Cross-job history + log viewer | `history list/show`, `logs tail` |
| `Live log` | Tail while watching a run | log file / `logs tail` |
| `Deployments` | Promote / version history | promote JSON / deployments table |
| `Doctor` | Health + backup path hint | `doctor`, `doctor --fix` |
| `Plan confirmation` | Modal before run | `plan --json` |
| `Prod confirmation` | Type env name | GUI gate → `run` |

### Job workspace frames (Figma)

| Frame | Purpose | Backing |
|-------|---------|---------|
| `Job / Overview` | **Homepage:** rendered `README.md` + Run / Plan | read `project/jobs/<id>/README.md`; `plan` / `run` |
| `Job / Files` | **GitHub-like:** folder tree + file viewer | read files under `project/jobs/<id>/` only |
| `Job / Schedules` | Schedules for this job + drift | `schedule list/set/delete`, `schedule reconcile` |
| `Job / Runs` | History filtered to this job | `history list --job` |
| `Job / Env` | Env editor for this job | `env list\|get\|set` |

---

## File viewer rules

| Kind | Display | Edit in GUI? |
|------|---------|--------------|
| `README.md` / `*.md` | Rendered markdown (Overview default; also in Files) | No — IDE on disk |
| `job.yaml`, `.bat`, `.sh`, other text | Monospace code viewer (UTF-8; Japanese OK) | No — IDE on disk |
| `env/*.example` | Code viewer | No |
| `data/env/.../.env.*` | Via **Job / Env** only (masked / `--reveal`) | Yes — `env set` |

Path safety: GUI may only open paths under `project/jobs/<job-id>/` for the Files tree (same rule as CLI path safety).

---

## States

| State | Where it shows |
|-------|----------------|
| empty catalog | Jobs catalog with CTA to add `project/jobs/<id>/` |
| empty tree / missing README | Overview: “Add README.md”; Files: empty folder message |
| loading | Any CLI call in flight |
| error | Readable CLI message (exit 2/3/10…) |
| running / skipped / failed | Badges on catalog, Job / Runs, History |
| validation blocked | Plan / Env — exit 4 / 9 |
| plan_id mismatch | After Confirm — exit 11 |
| drift | Job / Schedules after reconcile |

---

## Flows

### 1. Browse jobs (catalog)

1. Open **Jobs catalog**.  
2. `jobs list --json` + enrich with last run (`history list`) and schedules (`schedule list`).  
3. Each row looks like a project card: id/name, short description, **last run** status, **schedule** badge (or “unscheduled”).  
4. Click row → **Job / Overview**.

### 2. Job homepage (README)

1. Default tab **Overview**.  
2. GUI reads and renders `project/jobs/<id>/README.md`.  
3. Actions: **Plan**, **Run** (env from top chip / allow-list).  
4. Missing README → empty state with path hint.

### 3. Explore files (GitHub-like)

1. Tab **Files**.  
2. Left: recursive tree of job folder (`job.yaml`, `README.md`, `scripts/…`, `env/*.example`).  
3. Click file → right pane code or markdown viewer.  
4. No script execution from the viewer.

### 4. Run with confirmation

1. From Overview (or Files): **Run** / **Plan**.  
2. `plan <job> --env <p> --json` → **Plan confirmation** (masked secrets).  
3. Confirm → if prod, **Prod confirmation** (type env name).  
4. Start detached: `run … --trigger gui --plan-id … --yes --json-meta …`.  
5. Optional navigate to **Running** / **Live log**. GUI close does not kill.

### 5. Kill

1. **Running** (global) or deep-link from Job / Runs if still running.  
2. **Kill** → `cancel <run-id>`.

### 6. History two days later

1. Global **History** or **Job / Runs**.  
2. Open scheduled run → full UTF-8 log (Japanese OK).  
3. Skipped rows show `skip_reason`.

### 7. Env for a job

1. **Job / Env**.  
2. `env list/get/set` with mask / Reveal.  
3. Audit on set (no secret value stored).

### 8. Schedules for a job

1. **Job / Schedules** — filter `schedule list` to this `job_id`.  
2. Create/edit/delete via `schedule set|delete`.  
3. **Reconcile** → `schedule reconcile`; show drift for this job’s tasks.

### 9. Promote / Doctor

Unchanged globally: **Deployments**, **Doctor** (`doctor`, backup hint for `data/`).

---

## Action → CLI / filesystem map

| UI action | Backing |
|-----------|---------|
| Refresh catalog | `jobs list --json` (+ history/schedule for badges) |
| Open Overview | read `…/README.md` |
| Open file in tree | read path under `project/jobs/<id>/` |
| Validate | `schema validate <id> --json` |
| Plan / Run / Kill | `plan` / `run` / `cancel` |
| Job runs | `history list --job <id>` |
| Env | `env list\|get\|set` |
| Schedules | `schedule list\|set\|delete`, `schedule reconcile` |
| Doctor | `doctor [--fix]` |

---

## Out of scope (design + v1 GUI)

- In-GUI editor for `.bat` / `job.yaml` (use IDE).  
- Git UI inside the app (promote/folder replace first).  
- Executing scripts from the file viewer.
