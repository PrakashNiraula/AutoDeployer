# Build order

**GitHub issues are the source of truth.** This page is only a map. Rules are in `docs/WORKFLOW.md`.

Builder: Cursor. Reviewer and project manager: Claude. Owner: Prakash.

| Order | Issue | Milestone | Builder | Blocked by |
|-------|-------|-----------|---------|------------|
| 1 | #1 D1 Design vision brief | D - Design | Cursor | none |
| 2 | #2 D2 CLI contract spec and JSON schemas | D - Design | Cursor | none |
| 3 | #3 D3 GUI information architecture and flows | D - Design | Cursor | #1 |
| 4 | #4 D4 GUI visual design | D - Design | Cursor | #3 |
| 5 | #5 D5 Design review gate (G1) | D - Design | Reviewer | #1 to #4 |
| parallel | #6 P0-1 Spike fixtures and harness | Phase 0 | Cursor | none |
| parallel | #9 Owner: real scripts and Japanese console sample | Phase 0 | Owner | none |
| 6 | #7 P0-2 Run the spike on Japanese-locale Windows | Phase 0 | Owner | #6 |
| 7 | #8 P0-3 Spike review gate (G2) | Phase 0 | Reviewer | #7 |
| 8 | #10 1a Config, discovery, schema, path safety | Phase 1 | Cursor | #5 |
| 9 | #11 1b Env validation, masking, plan_id | Phase 1 | Cursor | #10 |
| 10 | #12 1c SQLite history, locks, reconcile, doctor | Phase 1 | Cursor | #11 |
| 11 | #13 1d Runner, encoding, timeout, cancel, detach | Phase 1 | Cursor | #12, #8 |
| 12 | #14 1e CLI wiring, integration tests, v0.1.0 (gate G3) | Phase 1 | Cursor | #13 |
| 13 | #15 Phase 2 Scheduling and notify | Phase 2 | Cursor | #14 |
| 14 | #16 Phase 3 Promote | Phase 3 | Cursor | #14 |
| 15 | #17 Phase 4 GUI (separate repo) | Phase 4 | Cursor | #14, #4 |

Open question for the spike: when a run is cancelled, do detached services survive? Suggested default: yes (breakaway), verified in #6 and #7, recorded in the contract.
