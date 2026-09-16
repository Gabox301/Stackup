# Proposal: stackup-tui-always

## Intent

Every interactive command shows TUI presence. Only `apply` confirms today; `detect`/`generate`/`diff` dump plain text, so interactive runs feel inconsistent.

## Scope

### In Scope
- Presence screens for `detect`, `generate`, `diff`, `apply`.
- Shared `shouldUseTUI()` gate on stdout+stdin TTY.
- Fallback to byte-identical plain text on any exclusion or launch failure.
- Gate-matrix tests + per-screen goldens.

### Out of Scope
- No new detectors or signal changes.
- No template or generated-file changes.
- No exit-code changes (0/1/2/3/4 preserved).
- No pager, search, or syntax highlighting.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `cli-ux`: add presence + gate/fallback requirements; CI scenarios stay verbatim.

## Approach

Contract only; Model shape belongs to design.
- Show TUI iff `stdoutIsTTY() && stdinIsTTY() && !--yes && !--non-interactive && --format text`.
- `--format json` NEVER presents, even on TTY.
- Compute first, then present; presence never gates writes. Only `apply` confirm returns `ActionApply`.
- Launch failure prints current text + stderr warning; engine exit preserved, never coerced to 1.

| Command | Screen | Content | On quit |
|---------|--------|---------|---------|
| `detect` | Evidence | Same rows as `formatEvidence`; empty mirrors exit-4 notice | Exit 0/4 kept |
| `generate` | Plan preview | File list with `(new)`/`(overwrite)` wording | Return, then write |
| `diff` | Unified diff | File selector + viewport; footer states exit-2 implication | Exit 2/0 kept |
| `apply` | Result | Confirm verbatim + `printWritten` summary; no-change screen | Semantics untouched |

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/stackup/root.go` | Modified | Stdout-based gate + `shouldUseTUI()`; prints become fallback |
| `cmd/stackup/detect.go`, `generate.go`, `diff.go` | Modified | TUI branch after result computed |
| `cmd/stackup/apply.go` | Modified | Confirm-only → always-present; exits/abort kept |
| `internal/tui/` | Modified | New screens; shape decided in design |
| `cli_test.go`, `tui_*_test.go`, `testdata/` | Modified | Gate matrix + goldens |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Piped breakage (`diff \| head` opens TUI) | High | Gate on stdout TTY; stubbed-probe matrix test |
| Windows console misrender | Med | ASCII renderer + fallback |
| Golden drift | Med | Deterministic `View()`; `-update` then clean rerun |

## Rollback Plan

Restore `stdinIsTTY` gate in `apply.go`; delete TUI branches in `detect`/`generate`/`diff.go`; keep confirm path. Drop unarchived delta.

## Dependencies

- None. `bubbles/viewport` already pinned in `go.mod`.

## Success Criteria

- [ ] Four commands present on TTY text runs; silent otherwise.
- [ ] JSON/`--yes`/`--non-interactive`/non-TTY bytes + exits identical.
- [ ] Launch failure falls back with exits preserved.
- [ ] `apply` confirm/abort wording unchanged.
