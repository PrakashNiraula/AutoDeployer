# Phase 0 spike

Fixtures and harness for plan section B (Japanese-locale Windows).  
Encoding decoder cases also run on Linux/macOS.

## Prerequisites (Windows)

1. Go 1.22+ on PATH (`go version`)
2. JDK with `javac` and `java` on PATH (for bat→java kill/detach cases)
3. Japanese locale recommended for live `out_cp932.bat` console tests; pre-encoded bytes cover decoder without locale

## One command

From this repo on Windows:

```bat
cd spike\harness
go test ./...
go run . -fixtures ..\fixtures
```

Or from repo root:

```bat
cd spike\harness && go test ./... && go run . -fixtures ..\fixtures
```

The harness prints a markdown PASS/FAIL/SKIP table for P0-a1..P0-c3.

## What each case needs

| ID | Needs Windows | Needs JDK | Notes |
|----|---------------|-----------|--------|
| P0-a1..a5 | yes | yes | tree-kill + detach |
| P0-b1..b5 | no | no | line-boundary decoder |
| P0-c1..c3 | yes | no | bat exit codes |

## Kill / detach mechanisms tried

- **`taskkill /PID /T /F`** — primary tree-kill used by the harness for timeout/cancel
- **Windows Job Object** — helpers in `harness/kill_windows.go`; create **without** `KILL_ON_JOB_CLOSE` for detach experiments
- **Detach** — `detach_start.bat` uses PowerShell `Start-Process` (breakaway-style); assert child alive after bat exits, then clean up

## After the run

1. Copy the printed table into `spike/REPORT.md`
2. Fill machine/locale/go version and go/no-go
3. Owner runs issue #7 on a JP Windows machine; reviewer gate is #8

## Linux quick check (encoding only)

```bash
cd spike/harness
go test ./...
go run . -fixtures ../fixtures
```

Windows-only cases show as `SKIP`.
