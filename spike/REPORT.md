# Phase 0 Spike Report

Date / machine / Windows version / locale / code page: **2026-11-22 — Linux (Arch, go1.26.5, UTF-8). JP Windows host not available in this session.**  
Go version: go1.26.5  
JDK version: n/a for Linux path

## Results

| ID | Result (PASS/FAIL/SKIP) | Notes / PIDs / log paths |
|----|-------------------------|---------------------------|
| P0-a1 | PASS | LineDecoder holdback (spike/harness decode_test) |
| P0-a2 | PASS | cp932 fixture decode path covered in unit tests |
| P0-a3 | PASS | Undecodable → U+FFFD without panic |
| P0-a4 | PASS | `auto` resolves to machine default (utf-8 on this host) |
| P0-a5 | SKIP | JP console live capture — needs owner JP Windows (#7) |
| P0-b1 | PASS | Unix process-group kill via `kill(-pgid, SIGKILL)` |
| P0-b2 | PASS | Normal completion does not kill children (no tree-kill on exit) |
| P0-b3 | PASS | Detach: child outside runner pgid survives CLI exit (Setpgid / no shared job) |
| P0-b4 | SKIP | Windows Job Object / BREAKAWAY — deferred to #7 |
| P0-b5 | SKIP | taskkill /T /F live — deferred to #7; fallback documented |
| P0-c1 | PASS | Non-zero step exit observed by Go parent (unix) |
| P0-c2 | SKIP | cmd.exe `/b` exit quirks — Windows only |
| P0-c3 | SKIP | Nested bat without call — Windows only |

## Detach mechanism used

**Unix (this host):** child processes started with `Setpgid: true` (own process group). Detach steps are not reaped/killed on normal CLI exit.  
**Windows (planned):** Job Object without `KILL_ON_JOB_CLOSE` and/or `CREATE_BREAKAWAY_FROM_JOB` (plan 4.3); verified when #7 runs.

## Kill mechanism used

**Unix:** `kill(-pgid, SIGKILL)` on the non-detached run process group.  
**Windows (planned primary):** `taskkill /PID <pid> /T /F` (B.3 fallback already preferred if Job Object unreliable).

## Fallbacks engaged

- Windows Job Object path not exercised → primary kill = taskkill /T /F when on Windows.
- JP live encoding sample deferred → decoder package isolated; explicit code pages supported; `auto` = machine default only.

## Detach-versus-cancel boundary (contract)

- Timeout/cancel tree-kills the **non-detached** run process tree/group only.
- `detach: true` / breakaway children **may survive** cancel (Q6 default).
- Normal completion **never** tree-kills.

## Go / no-go for Phase 1

Decision: **GO** (owner blanket APPROVED 2026-11-22; Linux spike evidence recorded; Windows #7 remains owner-action for JP locale confirmation without blocking 1d).

Risks remaining: Windows-specific bat exit and Job Object detach must be confirmed on #7; runner ships Unix path now and Windows taskkill primary kill.

---
_Filled for G2 (#8). Windows rows stay SKIP until #7._
