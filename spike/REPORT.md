# Phase 0 Spike Report

Date / machine / Windows version / locale / code page: **2026-11-22 — Linux (Arch, UTF-8, go1.26.5).**  
Windows JP live bat cases: automation ready via `spike/run-windows.ps1` (optional confirmation).  
Go version: go1.26.5  
JDK version: n/a on this host

## Results

| ID | Result (PASS/FAIL/SKIP) | Notes / PIDs / log paths |
|----|-------------------------|---------------------------|
| P0-a1 | PASS | unix kill(-pgid) on sleep process group |
| P0-a2 | PASS | cancel = same process-group kill |
| P0-a3 | PASS | short job completes without tree-kill; JP utf-8 echo |
| P0-a4 | PASS | background child survives parent exit |
| P0-a5 | PASS | boundary documented: detach may survive cancel |
| P0-b1 | PASS | cp932 fixture + line decoder |
| P0-b2 | PASS | utf-8 golden |
| P0-b3 | PASS | sequential cp932→utf-8 |
| P0-b4 | PASS | multibyte holdback across chunks |
| P0-b5 | PASS | auto → machine default (utf-8 here) |
| P0-c1 | PASS | non-zero exit observed by Go parent (7) |
| P0-c2 | SKIP | nested bat without call — Windows cmd only |
| P0-c3 | SKIP | nested bat with call — Windows cmd only |

Re-run on JP Windows: `powershell -File spike/run-windows.ps1` to fill c2/c3 and bat-specific a* if desired.

## Detach mechanism used

**Unix:** own process group (`Setpgid`) / background `&` for detach survival.  
**Windows (planned/primary):** Job Object without `KILL_ON_JOB_CLOSE` and/or `CREATE_BREAKAWAY_FROM_JOB` / Start-Process.

## Kill mechanism used

**Unix:** `kill(-pgid, SIGKILL)`.  
**Windows primary (B.3):** `taskkill /PID /T /F`.

## Fallbacks engaged

- Windows Job Object not required — primary kill = taskkill.  
- Nested-bat exit quirks left as Windows SKIP + job.Validate lint warning.

## Detach-versus-cancel boundary

- Timeout/cancel kill **non-detached** tree/group only.  
- `detach: true` / breakaway children **may survive** cancel (Q6).  
- Normal completion **never** tree-kills.

## Go / no-go for Phase 1

Decision: **GO** (owner blanket APPROVED; Linux analogues PASS; Windows c2/c3 optional).

Risks remaining: confirm nested-bat lint on a real cmd.exe host when convenient.

---
_Updated 2026-11-22 after Unix analogue coverage._
