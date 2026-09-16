# Proposal: stackup-tui-style

## Intent

Add "Charm vibrante" styling; keep goldens, accessibility, CI behavior.

## Scope

### In Scope
- New `internal/tui/theme.go`: Theme, RoundedBorder, badges, pills, footer, spacing 0/1/2.
- Style all five screens via whole-token wrapping; text tokens verbatim.
- Runtime renderer from output + env; constructors default Ascii.
- Stripped-golden migration + `theme_test.go` exact-escape pins.

### Out of Scope
- Gate, exits, fallback, JSON, Launcher, Decision semantics.
- Viewport `+`/`-` tinting and per-screen structural redesign.
- lipgloss v2; palette beyond green/amber/grey.
- Template, detector, or config-generation changes.

## Capabilities

### New Capabilities
- None — purely visual layer.

### Modified Capabilities
- `cli-ux`: add TUI styling requirement (caps, tokens, degradation). Needs delta.

## Approach

- Theme owns styling; Model defaults Ascii; runtime injects env renderer. Never force profile.
- Goldens: strip ANSI before compare (byte-identical); pin escapes in `theme_test.go` (ANSI256).
- Whole tokens only; bordered cards pad 0/1/2; viewport border costs 2 cols.
- Degradation via termenv default detection:

| Condition | Result |
|-----------|--------|
| `NO_COLOR` / `CLICOLOR=0` / non-TTY | No color |
| `TERM=dumb` (Unix) | Degraded |
| Legacy Windows | Ascii fallback |
| Full color TTY | Charm vibrante |

| Screen | Glow-up |
|--------|---------|
| Evidence | Header card; confidence via badges |
| Plan/Confirm | `(new)`/`(overwrite)` via pills |
| Diff | Selector rows styled; viewport framed; footer emphasizes `exit 2` |
| Result | Success title tone; `(new)`/`(backup: …)` pills; `no changes` kept |

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/theme.go` | New | Theme struct + style constructors |
| `internal/tui/model.go` | Modified | Theme field; whole-token styling |
| `internal/tui/diffview.go` | Modified | Border wrapper; footer emphasis |
| `internal/tui/tea.go` | Modified | Runtime renderer injection |
| `internal/tui/*_test.go`, `testdata/` | Modified | Strip-compare; `theme_test.go` |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Golden churn on palette tweaks | Low | Stripped goldens; theme-only updates |
| Over-styling hurts readability | Med | Cap: 1 border, 3 badges, 2 pills; no tint v1 |
| Legacy Windows raw escapes | Low | Ascii fallback; verify on Win10+ |

## Rollback Plan

Revert `theme.go` + renderer wiring to Ascii default; restore plain `View()`; keep strip harness.

## Dependencies

- lipgloss v1.1.0, bubbletea v1.3.4, termenv v0.16.0 pinned.

## Success Criteria

- [ ] Stripped goldens byte-identical; `go test ./...` green.
- [ ] `theme_test.go` pins badges, pills, title, border escapes.
- [ ] Tokens (`node: high`, `exit 2`, `(new)`/`(overwrite)`) verbatim in stripped output.
- [ ] NO_COLOR/dumb/non-TTY degrade with no forced profile.
