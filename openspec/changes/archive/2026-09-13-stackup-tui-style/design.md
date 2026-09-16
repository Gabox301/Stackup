# Design: stackup-tui-style — Charm vibrante TUI styling

## Technical Approach

A new `Theme` owns all lipgloss styles bound to an injected renderer. `Model` defaults to Ascii (deterministic `View()`), `tea.go` injects an env-detected renderer at runtime, and goldens compare ANSI-stripped output so styled bytes never churn goldens. Styling wraps whole tokens only, so stripped text stays verbatim.

## Architecture Decisions

| Option | Tradeoff | Decision |
|---|---|---|
| Theme struct vs inline styles in `model.go` | Inline is fewer files but scatters palette/caps | **Theme struct** (`internal/tui/theme.go`): single owner for caps, palette, spacing 0/1/2 |
| Stripped-golden compare (option-2) vs regenerating styled goldens | Styled goldens pin escapes and churn on palette tweaks | **Strip-compare**: goldens stay byte-identical; escapes pinned separately in `theme_test.go` |
| Adaptive Light/Dark pairs vs fixed hex | Fixed hex washes out on light terminals | **Adaptive pairs** for green/amber/grey triad (`lipgloss.AdaptiveColor`); exact pairs picked in tasks |
| Runtime injection in `tea.go` vs constructor param | Param threads renderer through every caller/test | **Zero-arg constructors stay Ascii**; only `modelForInput`/`Run` resolves env renderer and sets `Model.theme` |
| No viewport `+`/`-` tint in v1 vs tinted diff lines | Tint is pretty but risks readability and golden churn | **No tint v1** per spec cap; viewport gets border only |

## Data Flow

```text
out+env ──termenv default detect──▶ Renderer ──▶ Theme ──▶ Model.View()
                                                              │
 goldens: strip(View()) ──byte-compare── testdata/*.golden ◀──┘
 theme_test: fixed ANSI256 Renderer ──exact-escape asserts──▶ Theme styles
```

Renderer resolution (runtime only, never forced): `NO_COLOR`/`CLICOLOR=0`/non-TTY → no color; `TERM=dumb` (Unix) → degraded; legacy Windows → Ascii fallback; full-color TTY → full style. Constructors never read env.

## File Changes

| File | Action | Description |
|---|---|---|
| `internal/tui/theme.go` | Create | `Theme` + `NewAsciiTheme`/`NewTheme(renderer)`; `Title`, `RoundedBorder`, `BadgeHigh/Medium/Low`, `PillNew/PillOverwrite`, `Footer`/`FooterEmph`; green/amber/grey adaptive triad |
| `internal/tui/model.go` | Modify | Add `theme` field (zero value = Ascii); wrap whole tokens in `evidenceLine`, `fileLine`, `resultLines`, titles, `pendingLine` |
| `internal/tui/diffview.go` | Modify | Wrap viewport in `RoundedBorder` (width −2 cols); style selector `diffTag` tokens; emphasize `exit 2` in `diffFooter` |
| `internal/tui/tea.go` | Modify | Resolve renderer from `out`+env via default detection; inject into `modelForInput`; never force a profile |
| `internal/tui/theme_test.go` | Create | Exact-escape pins under fixed ANSI256 for badges, pills, title, border corners |
| `internal/tui/tui_test.go`, `screens_test.go` | Modify | Strip ANSI before golden compare via shared helper |
| `internal/tui/testdata/*.golden` | Modify | Harness-only; bytes stay identical |

## Interfaces / Contracts

```go
type Theme struct {
    Title, RoundedBorder, BadgeHigh, BadgeMedium, BadgeLow,
    PillNew, PillOverwrite, Footer, FooterEmph lipgloss.Style
}
func NewAsciiTheme() Theme
func NewTheme(r *lipgloss.Renderer) Theme
```

Contract: styling wraps whole tokens (`high`, `(new)`, `(overwrite)`, `(backup: …)`, `exit 2`); markers (`"> "`) and paths stay outside styles; `no changes` and empty-evidence notice stay unstyled.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | Badge/pill/title/border escapes | `theme_test.go` with fixed ANSI256 renderer, exact-escape asserts |
| Integration | Stripped goldens byte-identical (5 screens); tokens verbatim | Strip helper + existing `teatest.RequireEqualOutput`; `go test ./internal/tui` |
| E2E | Degradation matrix; interactive flows unchanged | Env-matrix cases (`NO_COLOR`, `dumb`, non-TTY); existing `teatest` interactive tests unmodified |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. Land `theme.go` + wiring + strip harness together; `go test ./...` green with identical golden bytes. Rollback: revert `theme.go` + wiring to Ascii `View()`; keep strip harness.

## Open Questions

None.
