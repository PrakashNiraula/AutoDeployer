# Working agreement

## Roles

| Role | Who | Does |
|------|-----|------|
| Builder | Cursor | Design and code. One issue per branch, one PR per issue. |
| Project manager / reviewer | Claude | Writes and orders issues, reviews every PR against its acceptance criteria and `docs/PLAN-v4.md`, runs the gates. Does not write the feature code. |
| Owner | Prakash | Final approval, merges, provides the Japanese-locale Windows machine for Phase 0 and real scripts. |

## Flow

1. Pick the lowest-numbered issue labelled `status:ready` in the current milestone.
2. Branch: `<issue-number>-<short-slug>` (for example `12-job-discovery`).
3. Build only what the issue's acceptance criteria say. Anything extra becomes a new issue.
4. Open a PR using the template. Link the issue with `Closes #N`. Set the label `status:in-review`.
5. Reviewer checks the PR. Outcome is `status:changes-requested` (with a list of fixes) or approval.
6. Owner merges after approval. The issue closes and the next one becomes `status:ready`.

## Status labels

`status:ready` · `status:in-progress` · `status:in-review` · `status:changes-requested` · `status:blocked` · `status:done`

## Gates (no later work starts until the gate passes)

| Gate | Passes when |
|------|-------------|
| G1 Design | Vision, contract spec and GUI designs are reviewed and signed off. Phase 1 may start after the contract spec (D2). |
| G2 Spike | Phase 0 report exists, every P0 case is PASS or has a documented fallback, and go/no-go is "go". The runner (issue 1c) starts only after G2. |
| G3 Core | CLI-only operation works end to end and the invariants in plan section 7 pass. |

## Definition of done (every issue)

- Acceptance criteria all met, each one demonstrable.
- Tests added or updated and passing (`go test ./...`, `go vet ./...`).
- No secrets, real env files, logs or `data/` contents committed.
- Contract changes (CLI flags, JSON shapes, exit codes, file formats) are reflected in `docs/design/CONTRACT.md` in the same PR.
- Windows-specific behavior is marked as tested on Windows or flagged as untested.

## Reviewer checklist

- Does the PR do what the issue asks, and nothing else?
- Do locked decisions in `docs/PLAN-v4.md` (section C) still hold?
- Are exit codes and JSON shapes unchanged unless the contract doc changed with them?
- Are Windows edge cases handled: encoding, process tree, locking, rename?
- Are failures reported with readable messages for non-developers?
