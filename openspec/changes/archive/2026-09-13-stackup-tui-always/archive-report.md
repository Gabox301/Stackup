# Archive Report: stackup-tui-always — TUI presence for every command

- Change: `stackup-tui-always`
- Project: `stackup`
- Archived to: `openspec/changes/archive/2026-09-13-stackup-tui-always/`
- Archive date: 2026-09-13 (ISO)
- Artifact store mode: `hybrid` (filesystem + Engram)
- Engram topic: `sdd/stackup-tui-always/archive-report`

## Final State (authoritative at close)

- Verify: `pass` (clean, NOT pass_with_warnings), 0 blockers, 0 criticals, 5/5 requirements, 12/12 scenarios (6 delta + 6 canonical). NO code changes after verify.
- Tasks: 15/15 `[x]` (see Task Completion Gate reconciliation below).
- Delivery: LOCAL commits only (slices + progress chores), NO remote, NO PRs opened.
- Size exceptions recorded and PENDING at PR time: slice 1 (~823), slice 2 (~419, 19 over), slice 3 (~492).
- Pre-existing workdir hygiene (NOT this change, left exactly as found, do NOT touch): untracked `stackup` binary (user's own build), untracked `package.json`/`package-lock.json` strays, uncommitted `go.mod`/`go.sum` tidy, untracked `README.md`.

## Artifacts Read (traceability)

| Artifact | Filesystem | Engram observation |
|----------|-----------|-------------------|
| proposal | `openspec/changes/stackup-tui-always/proposal.md` (pre-move path) | #1790 `sdd/stackup-tui-always/proposal` |
| spec delta | `openspec/changes/stackup-tui-always/specs/cli-ux/spec.md` (pre-move path) | #1791 `sdd/stackup-tui-always/spec` |
| design | `openspec/changes/stackup-tui-always/design.md` (pre-move path) | #1792 `sdd/stackup-tui-always/design` |
| tasks | `openspec/changes/stackup-tui-always/tasks.md` (pre-move path, 15/15 `[x]`) | #1793 `sdd/stackup-tui-always/tasks` (stale, see below) |
| apply-progress | `openspec/changes/stackup-tui-always/apply-progress.md` (pre-move path) | #1794 `sdd/stackup-tui-always/apply-progress` (title: `stackup-tui-always slices 1-3 COMPLETE...`) |
| verify-report | `openspec/changes/stackup-tui-always/verify-report.md` (pre-move path) | #1795 `sdd/stackup-tui-always/verify-report` |
| canonical spec | `openspec/specs/cli-ux/spec.md` (pre-composition) | n/a (filesystem source of truth) |

All filesystem paths above refer to pre-move locations; post-move they live under `openspec/changes/archive/2026-09-13-stackup-tui-always/`.

## Task Completion Gate Reconciliation (exceptional, with proof)

The filesystem `tasks.md` shows 15/15 `[x]`; Engram observation #1793 still shows all boxes unchecked (`- [ ]`). #1793 is a stale pre-apply planning snapshot (created 2026-09-13 02:53:30, before `sdd-apply` marked completion in the file). Reconciliation reason: the orchestrator's launch prompt declares the explicit final-state fact "Authoritative counts: 15/15 tasks [x]", and completion is proven by (a) the filesystem `tasks.md` with 15 `[x]` and 0 `[ ]`, (b) `apply-progress` (file + #1794) documenting slices 1-3 complete with all tasks 1.1-4.1 `[x]`, and (c) `verify-report` (file + #1795) recording "Tasks total 15, complete 15, incomplete 0". No task content was altered by archive; the stale Engram planning copy is left untouched as history. The archived audit trail contains zero unchecked implementation tasks.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| cli-ux | Updated | 3 ADDED, 0 modified, 0 removed, 0 renamed |

Composition (native, mandatory path — no model Read/Edit merge):

```bash
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/cli-ux/spec.md" \
  --delta "openspec/changes/stackup-tui-always/specs/cli-ux/spec.md" \
  --output "openspec/specs/cli-ux/spec.md.compose-tmp" \
&& mv "openspec/specs/cli-ux/spec.md.compose-tmp" "openspec/specs/cli-ux/spec.md"
```

- Exit: 0 (zero exit is the only composition evidence; the `.compose-tmp` + `mv` kept the write atomic).
- Appended requirements: `TUI presence per command` (TTY-shows, apply-confirm-unchanged), `TUI gate rule` (piped-stays-text, json-never, yes-flag-stays-text), `Fallback and write gating` (launch-failure-fallback).
- Both pre-existing canonical requirements (`Core commands with preview and machine output`, `Non-interactive safety with replaceable TUI`) preserved verbatim with all 6 canonical scenarios.
- Delta is ADDED-only: no destructive content, so the `openspec/config.yaml` archive rule ("Warn before merging destructive deltas") required no warning and no confirmation.

## Archive Move (mechanical, shell-only)

- `git mv openspec/changes/stackup-tui-always openspec/changes/archive/2026-09-13-stackup-tui-always` succeeded (exit 0); no `mv` fallback was needed.
- Pre-move recursive snapshot copied with `Copy-Item -Recurse` to temp; post-move `diff -r` readback below is the passing evidence.
- No destination collision (`2026-09-13-stackup-tui-always` did not exist; siblings are `2026-09-12-stackup-cli`, `2026-09-13-stackup-wave2`).
- No file content passed through the model's Read/Write path. Snapshot temp dir removed after readback.

MANDATORY `diff -r` readback (snapshot source vs. archived destination), verbatim:

```text
---DIFF-READBACK-START---
---DIFF-READBACK-END---
```

(empty output, GNU diffutils 3.12, exit 0 — byte-identical; the archive-report file itself is additive-only and excluded because it did not exist in the source snapshot.)

## Archive Contents

- proposal.md ✅
- specs/cli-ux/spec.md (delta) ✅
- design.md ✅
- tasks.md ✅ (15/15 complete, 0 unchecked)
- apply-progress.md ✅
- verify-report.md ✅ (verdict `pass`, clean)
- exploration.md ✅ (carried over from explore phase)
- archive-report.md ✅ (this file, additive post-move)

Active changes directory no longer contains `stackup-tui-always` ✅. Git records staged renames for the previously tracked files (`tasks.md`, `apply-progress.md`) plus the composed canonical spec modification; previously untracked artifacts moved along on disk. Nothing was committed by archive (commits/PRs are the orchestrator's next action).

## Source of Truth Updated

- `openspec/specs/cli-ux/spec.md` now carries the 3 new TUI requirements alongside the 2 pre-existing ones (5 requirements total in the cli-ux domain).

## SDD Cycle Complete

The change was fully planned, implemented, verified, and archived. No code changes occurred after verify.

## Next Action (left obvious for the orchestrator)

Remote + 3 stacked PRs (stacked-to-main, auto-chain), each carrying its recorded `size:exception`: slice 1 (~823), slice 2 (~419, 19 over), slice 3 (~492). Pre-existing workdir strays (`stackup` binary, `package.json`/`package-lock.json`, `go.mod`/`go.sum` tidy, `README.md`) must stay exactly as found — exclude them from PR diffs.
