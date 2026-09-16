# Archive Report: stackup-tui-style — Charm vibrante TUI styling

- Change: `stackup-tui-style`
- Project: `stackup`
- Archived to: `openspec/changes/archive/2026-09-13-stackup-tui-style/`
- Archive date: 2026-09-13 (ISO)
- Artifact store mode: `hybrid` (filesystem + Engram)
- Engram topic: `sdd/stackup-tui-style/archive-report`

## Final State (authoritative at close)

- Verify: `pass` clean (corrected rerun, NOT pass_with_warnings), 0 blockers, 0 criticals, envelope 2/2 requirements + 5/5 scenarios delta-only admitted by `sdd-verify-validate`; 12 canonical scenarios re-run green as supporting no-regression evidence in body (excluded from envelope). NO code changes after either verify.
- Tasks: 9/9 `[x]` (see Task Completion Gate reconciliation below).
- Delivery: LOCAL commits only, NO remote, NO PRs opened.
- Size exceptions recorded and PENDING at PR time: slice 1 within budget, slice 2 (~474) over, slice 3 (~252) within.
- Pre-existing workdir hygiene (NOT this change, left exactly as found, do NOT touch): untracked `stackup.exe`, untracked `package.json`/`package-lock.json` strays, uncommitted `go.mod`/`go.sum` tidy, untracked `README.md`.

## Artifacts Read (traceability)

| Artifact | Filesystem | Engram observation |
|----------|-----------|-------------------|
| proposal | `openspec/changes/stackup-tui-style/proposal.md` (pre-move path) | #1799 `sdd/stackup-tui-style/proposal` |
| spec delta | `openspec/changes/stackup-tui-style/specs/cli-ux/spec.md` (pre-move path) | #1800 `sdd/stackup-tui-style/spec` |
| design | `openspec/changes/stackup-tui-style/design.md` (pre-move path) | #1801 `sdd/stackup-tui-style/design` |
| tasks | `openspec/changes/stackup-tui-style/tasks.md` (pre-move path, 9/9 `[x]`) | #1802 `sdd/stackup-tui-style/tasks` (stale, see below) |
| apply-progress | `openspec/changes/stackup-tui-style/apply-progress.md` (pre-move path) | #1803 `sdd/stackup-tui-style/apply-progress` (title: `stackup-tui-style all 3 slices done (theme + glow-up + pins/matrix)`) |
| verify-report | `openspec/changes/stackup-tui-style/verify-report.md` (pre-move path) | #1804 `SDD verify report stackup-tui-style (delta-only 2x5 rerun)` |
| canonical spec | `openspec/specs/cli-ux/spec.md` (pre-composition) | n/a (filesystem source of truth) |

All filesystem paths above refer to pre-move locations; post-move they live under `openspec/changes/archive/2026-09-13-stackup-tui-style/`.
Native status at archive time: `dependencies.archive: ready`, `nextRecommended: archive`, `blockedReasons: []`, `taskProgress: total 9 / completed 9 / pending 0 / allComplete true`, `reviewOffer` invitation only (never archive state).

## Task Completion Gate Reconciliation (exceptional, with proof)

The filesystem `tasks.md` shows 9/9 `[x]` with 0 `[ ]`; Engram observation #1802 still shows all boxes unchecked (`- [ ]`). #1802 is a stale pre-apply planning snapshot (created 2026-09-13 04:14:58, before `sdd-apply` marked completion in the file). Reconciliation reason: the orchestrator's launch prompt declares the explicit final-state fact "Authoritative counts: 9/9 tasks [x] (file + verify proof; Engram tasks snapshot stale, reconcile with reason as in tui-always precedent)", and completion is proven by (a) the filesystem `tasks.md` with 9 `[x]` and 0 `[ ]` (confirmed post-move: archived `tasks.md` has 9 `[x]`, 0 unchecked), (b) `apply-progress` (file + #1803) documenting slices 1-3 complete with all tasks 1.1-4.1 `[x]`, and (c) `verify-report` (file + #1804) recording "Tasks total 9, complete 9, incomplete 0" plus native `sdd-status`/`sdd-continue` confirming `taskProgress 9/9` and `apply: all_done`. No task content was altered by archive; the stale Engram planning copy is left untouched as history. The archived audit trail contains zero unchecked implementation tasks.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| cli-ux | Updated | 2 ADDED, 0 modified, 0 removed, 0 renamed |

Composition (native, mandatory path — no model Read/Edit merge):

```bash
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/cli-ux/spec.md" \
  --delta "openspec/changes/stackup-tui-style/specs/cli-ux/spec.md" \
  --output "openspec/specs/cli-ux/spec.md.compose-tmp" \
&& mv "openspec/specs/cli-ux/spec.md.compose-tmp" "openspec/specs/cli-ux/spec.md"
```

- Exit: 0 (zero exit is the only composition evidence; the `.compose-tmp` + `mv` kept the write atomic).
- Appended requirements: `TUI styling caps and token fidelity` (styled-tty-shows, tokens-verbatim-when-stripped), `Style degradation and golden stability` (no-color-degrades, legacy-windows-ascii, palette-change-keeps-goldens).
- Canonical now: 7 requirements / 17 scenarios (5 pre-existing + 2 ADDED; 12 pre-existing + 5 delta). All 5 pre-existing canonical requirements (`Core commands with preview and machine output`, `Non-interactive safety with replaceable TUI`, `TUI presence per command`, `TUI gate rule`, `Fallback and write gating`) preserved with all 12 canonical scenarios.
- Delta is ADDED-only: no destructive content, so the `openspec/config.yaml` archive rule ("Warn before merging destructive deltas") required no warning and no confirmation.

## Archive Move (mechanical, shell-only)

- `git mv openspec/changes/stackup-tui-style openspec/changes/archive/2026-09-13-stackup-tui-style` succeeded (exit 0); no `mv` fallback was needed.
- Pre-move recursive snapshot copied with `Copy-Item -Recurse` to temp (`sdd-archive-6e9f9149-2689-4343-94e6-641ed2d26c92`); post-move `diff -r` readback below is the passing evidence.
- No destination collision (`2026-09-13-stackup-tui-style` did not exist; siblings are `2026-09-12-stackup-cli`, `2026-09-13-stackup-tui-always`, `2026-09-13-stackup-wave2`).
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
- tasks.md ✅ (9/9 complete, 0 unchecked)
- apply-progress.md ✅
- verify-report.md ✅ (verdict `pass`, clean, envelope 2/2 + 5/5)
- exploration.md ✅ (carried over from explore phase)
- archive-report.md ✅ (this file, additive post-move)

Active changes directory no longer contains `stackup-tui-style` ✅ (only `archive/` remains). Git records staged renames for the previously tracked files plus the composed canonical spec modification; previously untracked artifacts moved along on disk. Nothing was committed by archive (commits/PRs are the orchestrator's next action).

## Source of Truth Updated

- `openspec/specs/cli-ux/spec.md` now carries the 2 new TUI styling requirements alongside the 5 pre-existing ones (7 requirements / 17 scenarios total in the cli-ux domain).

## SDD Cycle Complete

The change was fully planned, implemented, verified, and archived. No code changes occurred after verify.

## Next Action (left obvious for the orchestrator)

No further SDD phase remains for this change (`next_recommended: none`). Delivery is ordinary repository policy: LOCAL commits only so far — remote + stacked PRs (stacked-to-main, auto-chain) remain pending with recorded `size:exception` (slice 1 within budget, slice 2 ~474 over, slice 3 ~252 within). Pre-existing workdir strays (`stackup.exe`, `package.json`/`package-lock.json`, `go.mod`/`go.sum` tidy, `README.md`) must stay exactly as found — exclude them from PR diffs.
