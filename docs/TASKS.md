# Build order

Mirrors the task list. Phase 0 must pass on a Japanese-locale Windows machine before the runner (1c) is finalized.

| # | Task | Notes |
|---|------|-------|
| 3 | Scaffold repo | done in first commit |
| 4 | Phase 0 fixtures and harness | `spike/` |
| 5 | Run Phase 0 on JP Windows | human step, fills `spike/REPORT.md` |
| 6 | 1a: config, discovery, schema, path safety | |
| 7 | 1b: env validation, masking, plan, plan_id | |
| 8 | 1c: runner, encoding, timeout, cancel, detach | needs Phase 0 results |
| 9 | 1d: SQLite history, locks, reconcile, doctor | |
| 10 | 1e: CLI wiring, exit codes, integration tests | tag v0.1.0 |
| 11 | Phase 2: schedule, reconcile, notify, gc | |
| 12 | Phase 3: promote | |
| 13 | Phase 4: GUI (separate repo) | optional |

Open decision for the spike: when a non-detached run is cancelled, do detached services survive? Suggested default: yes (breakaway), verified by P0-a4 and P0-a5.
