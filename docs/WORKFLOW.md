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

## Merge rules (hard)

These exist because earlier PRs (#18 to #30) were merged within seconds, with no review and with red CI.

1. **The builder merges only after the reviewer approves.** Owner instruction (2026-11-22): the owner does not relay messages or merge; the reviewer reviews and says OK, and the builder does the rest. So the builder may merge its own PR only when the reviewer comment `PM REVIEW: APPROVED` exists (rule 3), the checks satisfy rule 2, and every condition the reviewer wrote has been met. Without that comment the builder must not merge.
2. **CI must be green on both ubuntu-latest and windows-latest.** A red check means the PR is not mergeable, whatever else is true.
   *Exception while `main` itself is red (otherwise no fix could ever merge):* the builder may merge a PR whose checks are still red only if all three hold: (a) it introduces no new failing test (every failure also fails on `main`, or was hidden because the build did not compile); (b) every remaining failure is tracked by an open issue that the PR description lists; (c) the reviewer comment says "merge exception granted" and names those issues. The exception ends as soon as `main` is green.
3. **A reviewer comment is required.** A PR is mergeable only after a comment that starts with `PM REVIEW: APPROVED` and ends with the signature line `-- Claude (PM)`. The builder must never write that phrase or signature. `PM REVIEW: CHANGES REQUESTED` means fix and ask again.
4. **Stop the line.** While CI on `main` is red, the only PRs allowed are the ones that make it green.
5. **Done means verified.** An issue is closed only when every acceptance criterion is met and its evidence is on the PR. Items that need the owner (real Windows machine, real scripts) stay open until the owner confirms them; the builder must not close or skip them.
6. **Never move or delete a release tag.** Fix forward with a new tag and honest release notes.

## Autopilot loop (builder)

The builder works without waiting for the owner, in this loop:

1. Take the lowest-numbered `status:ready` issue. Branch, build, open a PR (template, `Refs #N` or `Closes #N` only if every criterion is met).
2. **Poll GitHub for the reviewer's verdict** (for example `gh pr view <n> --comments` every few minutes); do not wait for the owner to relay it.
3. `PM REVIEW: CHANGES REQUESTED`: fix, push, and wait for a new verdict. `PM REVIEW: APPROVED`: meet any conditions it lists, then merge, close the issue only if all criteria are met, and go back to step 1.
4. Stop and ask the owner only for: issues labelled `owner-action`, decisions labelled `needs-owner-confirmation` that are not delegated in the issue, changing a locked decision, and anything the reviewer marks "owner decision needed".

## Status labels

`status:ready` · `status:in-progress` · `status:in-review` · `status:changes-requested` · `status:blocked` · `status:done`

## Gates (no later work starts until the gate passes)

| Gate | Passes when |
|------|-------------|
| G1 Design | Vision, contract spec, GUI flows and the Figma designs are reviewed by the reviewer **and explicitly confirmed by the owner** (see "Owner confirmation checkpoints"). No development until the owner comments `APPROVED` on the G1 issue (#5). |
| G2 Spike | Phase 0 report exists, every P0 case is PASS or has a documented fallback, and go/no-go is "go". The runner (issue 1c) starts only after G2. |
| G3 Core | CLI-only operation works end to end and the invariants in plan section 7 pass. |

## Owner confirmation checkpoints

Design comes first and is done in **Figma**. Development never starts on its own.

1. **After design (G1).** When D1 to D4 are finished and reviewed, the builder stops and posts a confirmation request on issue #5 containing: the Figma link, links to the design docs, a short summary of what will be built, the open questions with their defaults, and the exact line: `Reply APPROVED to start development, or list changes.`
2. The builder does **not** open any Phase 1 branch until the owner replies `APPROVED` on #5. Any other reply means changes: the builder updates the designs and asks again.
3. **After the spike (G2).** Before the runner (issue #13) starts, the builder posts the same kind of request on #8 with the spike report summary and waits for `APPROVED`.
4. **Before each later phase** (2, 3, 4) the builder posts a short summary on the phase issue and waits for `APPROVED`.
5. Changing a locked decision, a contract shape or an exit code after approval requires a new owner confirmation.

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
