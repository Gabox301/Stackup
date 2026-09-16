# Archive Report: stackup-cli — Detection + IDE Config Generation

- Change: `stackup-cli`
- Project: `stackup`
- Archived to: `openspec/changes/archive/2026-09-12-stackup-cli/`
- Archive date: 2026-09-12 (ISO)
- Artifact store mode: `hybrid` (filesystem + Engram)
- Engram topic: `sdd/stackup-cli/archive-report`

## Final State at Close (terminal record)

Per the Final-State Authority hierarchy, this report describes the change AT CLOSE.
`apply-progress` and `verify-report` are intermediate snapshots; where the
orchestrator's explicit final-state facts outrank them, the final facts are reported.

- Verify verdict: `pass_with_warnings`, 0 blockers, 0 criticals.
- Requirements: 6/6, scenarios: 12/12 (at verification time).
- Tests: 43 PASS / 0 FAIL across 7 packages (`go test ./... -count=1`).
- Static checks: `gofmt`, `go vet`, `golangci-lint` clean.
- Canonical harnesses: exit codes 2/3/4 reproduced on the built binary; all
  `--format json` payloads parseable.
- NO code changes after verify. The only archive-time change is the
  verify-directed documentation fold-in (W1/W2) described below — spec text only.

## Task Completion Gate

- Persisted tasks artifact (`tasks.md`, now archived): 19/19 `[x]`, 0 unchecked.
- Native SDD status at archive time: `taskProgress 19/19 allComplete`,
  `dependencies.archive: ready`, `nextRecommended: archive`,
  `actionContext.mode: repo-local` (no workspace-planning guard triggered).
- Authoritative task count is 19/19 per `verify-report` (Engram #1775); any
  divergent per-slice counting in intermediate notes is superseded.
- No stale-checkbox reconciliation was needed; the gate passed on first inspection.

## Verify Warnings Folded In (W1/W2, owned by archive)

`verify-report` (Engram #1775) closed with two spec-silence WARNINGs and directed
archive to document them in the `cli-ux` spec. Both were folded into the delta
spec BEFORE the mechanical sync, so the archived delta and the new main spec
carry them byte-identically (SHA256 `F41E8B89…D142` on both copies):

- W1 — TTY-prompt-decline exits 0: new scenario `TTY decline aborts with success
  status` documents that declining the TTY confirmation (EOF or anything-but-yes)
  aborts with zero writes, prints `apply aborted (no writes performed)`, and
  exits 0. An explicit user abort is neither ok-with-writes nor an error nor a
  non-interactive block.
- W2 — Fresh-vs-overwrite distinction: new scenario `Fresh files apply without
  confirmation` documents that the confirmation gate and the non-interactive
  block (exit 3) apply only to pending overwrites (`Changed && !IsNew`); fresh
  files write without confirmation even under `--non-interactive`, which is
  therefore not a global write blocker.

Both additions are documentation of verified behavior (covered by
`TestApplyConfirmDialogOnTTY` and the fresh-apply harness); no behavior, code,
or test changed, so no re-verify was required.

## Specs Synced (source of truth)

Greenfield change: `openspec/specs/` was empty, so each delta spec was copied
mechanically (shell `Copy-Item`, never model Read/Write) as a full spec.
`rules.archive` ("warn before merging destructive deltas") was checked: the
change contains no REMOVED/RENAMED sections, so the merge is purely additive
and no warning was warranted. Native `sdd-archive-compose` was not applicable
(no canonical spec existed to compose against).

| Domain | Action | Details |
|--------|--------|---------|
| cli-ux | Created | 2 requirements, 6 scenarios (4 verified + W1/W2 doc scenarios) → `openspec/specs/cli-ux/spec.md` |
| config-generation | Created | 2 requirements, 4 scenarios → `openspec/specs/config-generation/spec.md` |
| stack-detection | Created | 2 requirements, 4 scenarios → `openspec/specs/stack-detection/spec.md` |

Byte-identity readback (shell, independent of the model): `git diff --no-index`
on each delta→main pair returned empty output with exit 0, and SHA256 pairs
matched (`F41E8B89…` / `AD83B1F0…` / `0D1E1D6F…`). NOTE: this Windows host has no
`diff(1)` binary, so `git diff --no-index` plus SHA256 was used as the native
structural readback; verbatim outputs are recorded in the phase result.

## Archive Contents

- proposal.md ✅
- specs/cli-ux/spec.md ✅ (with W1/W2)
- specs/config-generation/spec.md ✅
- specs/stack-detection/spec.md ✅
- design.md ✅
- tasks.md ✅ (19/19 complete)
- apply-progress.md ✅
- verify-report.md ✅
- exploration.md ✅ (supplementary)
- research.md ✅ (formal `blocked` rev2 — see lineage)
- archive-report.md ✅ (this file, additive after the move)

Move readback: `git mv` exit 0, source absent afterwards; `git diff --no-index
--stat` snapshot-vs-destination empty (exit 0); all 10 file SHA256 hashes
identical between pre-move snapshot and destination.

## Source of Truth Updated

- `openspec/specs/cli-ux/spec.md` (new, incl. W1/W2)
- `openspec/specs/config-generation/spec.md` (new)
- `openspec/specs/stack-detection/spec.md` (new)

## Delivery State (pending maintainer decision, NOT resolved)

- 5 LOCAL commits stacked on `master`: `4d034ec` (PR1 detect), `f400882` (PR2
  generate+merge), `e399107` (PR3 diff+apply), `2f1dc2b` (PR4 CLI),
  `67c74a5` (PR5 TUI+goldens).
- NO git remote configured (`git remote -v` empty). NO PRs opened.
- `size:exception` will be needed at PR-open time for PR1 (~969), PR2 (~1354),
  PR4 (~928), PR5 (~672) lines. Recorded as a pending maintainer decision.

## Research Lineage

Formal `sdd-research` stayed `blocked` at revision 2 (Engram #1757; file
`research.md` rev2, `proposal_ready: false`, admission denied — child runtime
had no evidence tooling). It was superseded by direct Firecrawl evidence
gathering (Engram #1766, topic `sdd/stackup-cli/research-firecrawl`, 5 lanes:
devin/kiro/cursor/JSONC-in-Go/TUI), which informed proposal/spec/design. A
future reader must not expect a `done` formal research artifact for this change.

## Traceability (Engram observation IDs actually read)

| Artifact | Observation ID | Topic |
|----------|---------------|-------|
| proposal | #1767 | sdd/stackup-cli/proposal |
| spec (all domains) | #1769 | sdd/stackup-cli/spec |
| design | #1770 | sdd/stackup-cli/design |
| tasks (slice-4 snapshot; file is authoritative, 19/19) | #1771 | sdd/stackup-cli/tasks |
| apply-progress (final, slices 1–5) | #1772 | sdd/stackup-cli/apply-progress |
| verify-report | #1775 | sdd/stackup-cli/verify-report |
| research (formal, blocked rev2) | #1757 | sdd/stackup-cli/research |
| research-firecrawl (effective evidence) | #1766 | sdd/stackup-cli/research-firecrawl |
| explore | #1755 | sdd/stackup-cli/explore |
| delivery decision (chained, stacked-to-main) | #1756 | delivery decision |

## SDD Cycle Complete

The change has been fully planned, implemented, verified, and archived.
Next action (outside SDD): configure the git remote and open the 5 stacked PRs
with the recorded `size:exception` decisions.
