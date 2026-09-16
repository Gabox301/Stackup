# Archive Report: stackup-wave2 — Four detectors, JS deepening, extensions

**Change**: stackup-wave2
**Archived to**: `openspec/changes/archive/2026-09-13-stackup-wave2/`
**Date**: 2026-09-13
**Mode**: hybrid (Engram `sdd/stackup-wave2/archive-report` + filesystem)
**Verdict at close**: `pass_with_warnings` (verify), 0 blockers, 0 criticals, 15/15 tasks `[x]`

## Final State (authoritative at close)

- **Verify**: `pass_with_warnings`, 0 blockers, 0 criticals; 5/5 requirements, 10/10 scenarios covered by passing tests. Exact extension-ID casings confirmed in emitted output; forbidden IDs appear only in comments/negative asserts, never in emitted output. NO code changes after verify (working tree was clean at archive start except the intended spec sync below).
- **Verify WARNING resolved**: the untracked-artifacts warning in `verify-report` (Engram #1786) is RESOLVED — planning artifacts were committed as `c8cbe44` on top of slice commits (`8d9819c`, `6fae378`, `76ecabc`). Proven by clean `git status` at archive start plus `git show --stat HEAD` listing all six planning files.
- **Tasks**: authoritative count 15/15 `[x]` in the persisted filesystem artifact `tasks.md` (also confirmed in the archived copy — zero `- [ ]` lines). No stale-checkbox reconciliation was needed.
- **Stale-snapshot note (Final-State Authority)**: Engram observation #1784 (`sdd/stackup-wave2/tasks`, revisions: 2) still shows Phase 3–4 boxes unchecked because it was saved mid-chain (after slice 2). It is superseded by the filesystem `tasks.md` (rank 1) and the orchestrator's explicit final-state facts (rank 2). Reported here only as history; it is NOT the state at close.
- **Delivery**: LOCAL commits only, NO remote configured (`git remote -v` empty), NO PRs opened. size:exception pending for wave-2 PR1 (~590 authored lines); PR2 (~314) and PR3 (~340) are within the 400-line budget.
- **Research lineage**: formal `sdd-research` was n/a for this change (unavailable runtime). Extension-ID evidence is direct verification recorded in Engram `sdd/stackup-wave2/ext-ids` (#1780): canonical Marketplace item URLs, Firecrawl 401 fallback noted, no IDs invented.

## Engram Observations Read (traceability)

| Topic | Observation ID | Notes |
|---|---|---|
| `sdd/stackup-wave2/proposal` | #1781 | Full read |
| `sdd/stackup-wave2/spec` | #1782 | Full read |
| `sdd/stackup-wave2/design` | #1783 | Full read |
| `sdd/stackup-wave2/tasks` | #1784 | Full read — STALE (mid-chain snapshot, see note above) |
| `sdd/stackup-wave2/apply-progress` | #1785 | Full read (slices 1–3 FINAL) |
| `sdd/stackup-wave2/verify-report` | #1786 | Full read (`pass_with_warnings`, 0/0) |
| `sdd/stackup-wave2/ext-ids` | #1780 | Full read (Marketplace verification, Firecrawl 401 fallback) |
| `sdd/stackup-wave2/explore` | #1778 | Search preview (context only) |
| Propose handoff | #1779 | Search preview (context only) |

Filesystem sources read in full: `proposal.md`, `specs/stack-detection/spec.md`, `specs/config-generation/spec.md`, `design.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`, `exploration.md`, plus both canonical specs before and after composition.

## Specs Synced (native composition, ADDED-only)

Both deltas were ADDED-only; composition ran through native `sdd-archive-compose` (mandatory path, zero exit = only evidence). Canonical requirements preserved byte-for-byte at the top; delta sections appended.

1. `gentle-ai sdd-archive-compose --canonical "openspec/specs/stack-detection/spec.md" --delta "openspec/changes/stackup-wave2/specs/stack-detection/spec.md" --output "openspec/specs/stack-detection/spec.md.compose-tmp"` → EXIT:0 → moved atomically over canonical. Result: 39 → 95 lines; 3 requirements ADDED (Wave-2 ecosystem detectors, Optional Runtime/Frameworks evidence fields, Node Bun runtime and framework signals), 0 modified/removed/renamed.
2. `gentle-ai sdd-archive-compose --canonical "openspec/specs/config-generation/spec.md" --delta "openspec/changes/stackup-wave2/specs/config-generation/spec.md" --output "openspec/specs/config-generation/spec.md.compose-tmp"` → EXIT:0 → moved atomically over canonical. Result: 46 → 92 lines; 2 requirements ADDED (Extended verified recommendations table, Union merge and prose targets), 0 modified/removed/renamed.

No destructive merge (nothing removed); `rules.archive` ("Warn before merging destructive deltas") did not trigger. No `.compose-tmp` files remain.

## Archive Move (mechanical, shell-only)

- Source: `openspec/changes/stackup-wave2` → Destination: `openspec/changes/archive/2026-09-13-stackup-wave2` via `git mv` (all 8 files tracked). Pre-move recursive snapshot copied to temp dir; post-move `diff -r` snapshot-vs-destination is the readback. No file content passed through the model Read/Write path.
- MANDATORY readback verbatim output (`diff -r`, GNU diff via Git-for-Windows `usr/bin/diff.exe`):

```text
DIFF-EXIT:0
```

(empty diff = byte-identity proven; this `archive-report.md` was written after the move and is additive-only, excluded from the comparison by construction.)

## Archive Verification

- [x] Main specs updated correctly (compose exit 0 × 2; ADDED sections present; unrelated requirements preserved)
- [x] Change folder moved to archive (`2026-09-13-stackup-wave2`)
- [x] Archive contains all artifacts: proposal.md, specs/stack-detection/spec.md, specs/config-generation/spec.md, design.md, tasks.md, apply-progress.md, exploration.md, verify-report.md (+ this archive-report.md)
- [x] Archived `tasks.md` has no unchecked implementation tasks (zero `- [ ]` lines; 15/15 `[x]`)
- [x] Active `openspec/changes/` holds only `archive/` — the change is gone from active
- [x] Verbatim `diff -r` readback included above and empty

CRITICAL gate: none in verify-report — archive not blocked. No user override was needed or used.

## Source of Truth Updated

- `openspec/specs/stack-detection/spec.md` — now includes wave-2 detectors, Runtime/Frameworks fields, Bun/framework rules.
- `openspec/specs/config-generation/spec.md` — now includes the verified extension table (exact casings, React=NONE, forbidden-IDs rule) plus union/prose targets.

## SDD Cycle Complete

The change was fully planned, implemented (3 stacked slices), verified (`pass_with_warnings`, warning since resolved), synced, and archived. Uncommitted at close: the two synced canonical specs + staged archive renames (commit left to the orchestrator/delivery step).

## Next Action (obvious)

Configure the remote, then open the 3 stacked PRs to main (PR1 needs the recorded `size:exception`, ~590 authored lines; PR2 ~314, PR3 ~340 within budget). Suggested mapping: Unit 1 (evidence fields + 4 detectors + registry order) → PR1; Unit 2 (Node Bun + framework scan) → PR2; Unit 3 (extension map + prose + goldens + lint) → PR3. See `tasks.md` Suggested Work Units and `apply-progress.md` rollback boundaries per PR.
