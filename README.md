# AutoDeployer

A per-machine ops console core: register jobs (ordinary `.bat` / `.sh` scripts plus a `job.yaml`), pick an environment, review a plan, run with live logs, schedule, and review History later.

**The CLI is the product. A GUI is an optional, separate thin client.**

Status: scaffold. See `docs/PLAN-v4.md` for the full plan (v4 FINAL) and `docs/TASKS.md` for the build order.

## Layout

```
app/        binaries (built output, not committed)
project/    REPLACED on promote: job definitions, scripts, env templates
data/       machine-local, NEVER replaced or committed: env, logs, history.db, schedules, locks
cmd/        Go entry points
internal/   Go packages
spike/      Phase 0 fixtures and report (run on a Japanese-locale Windows machine)
docs/       plan, contract, decisions
```

## Build

```
go build -o app/autodeployer ./cmd/autodeployer
go test ./...
./app/autodeployer version
```

## Key rules (v4)

- v1 secrets are plain `.env` files in `data/env/<job-id>/`, readable by operators. Hardening is deferred.
- Real secrets never live in `project/`; only `*.example` templates do.
- Every run is a separate OS process. Closing a GUI never kills a run.
- Promote replaces `project/` only and never touches `data/`.
