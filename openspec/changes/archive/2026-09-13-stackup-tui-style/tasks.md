# Tasks: stackup-tui-style — Charm vibrante TUI styling

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~380-440 |
| 400-line budget risk | Medium |
| Chained PRs recommended | Yes |
| Suggested split | PR1 → PR2 → PR3 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Theme core + Ascii default | PR1 | `go test ./internal/tui -run TestTheme -count=1` | N/A — no runtime path; Ascii default keeps View() deterministic | Remove `internal/tui/theme.go`; theme field reverts to Ascii |
| 2 | Screen glow-up + runtime injection | PR2 | `go test ./internal/tui -run TestEvidenceView -count=1` | `go test ./internal/tui -run TestConfirmInteractive -count=1` (live Bubble Tea) | Revert `internal/tui/model.go`, `internal/tui/diffview.go`, `internal/tui/tea.go` to plain View() |
| 3 | Golden migration + pins + matrix + hygiene | PR3 | `go test ./... -count=1` | `go test ./internal/tui -run TestPresenceInteractive -count=1` | Revert `internal/tui/*_test.go` strip helper; goldens bytes unchanged so safe |

## Phase 1: Theme core

- [x] 1.1 Create `internal/tui/theme.go` with `Theme`, `NewAsciiTheme`/`NewTheme(renderer)`, `Title`/`RoundedBorder`/`BadgeHigh`/`BadgeMedium`/`BadgeLow`/`PillNew`/`PillOverwrite`/`Footer`/`FooterEmph`; lock triad green `#1A7F37`/`#3FB950`, amber `#9A6700`/`#D29922`, grey `#59636E`/`#9198A1` via `lipgloss.AdaptiveColor`; spacing 0/1/2 only (→ styled-tty-shows)
- [x] 1.2 Add `theme` field to `Model` in `internal/tui/model.go`; zero value = Ascii, zero-arg constructors unchanged (→ palette-change-keeps-goldens)

## Phase 2: Screen glow-up + injection

- [x] 2.1 Style `internal/tui/model.go` `evidenceLine`/`fileLine`/`resultLines`/titles/`pendingLine` wrapping whole tokens only (`high`, `(new)`, `(overwrite)`, `(backup: …)`); markers/paths outside styles; `no changes` unstyled (→ tokens-verbatim-when-stripped)
- [x] 2.2 Style `internal/tui/diffview.go`: wrap viewport in `RoundedBorder` (width −2 cols), style selector `diffTag` whole tokens, emphasize `exit 2` in `diffFooter`; no `+`/`-` tint (→ styled-tty-shows)
- [x] 2.3 Inject runtime renderer in `internal/tui/tea.go` from `out`+env via termenv default detection in `modelForInput`/`Run`; never force profile (→ no-color-degrades)

## Phase 3: Golden migration + pins + matrix

- [x] 3.1 Add strip helper via `x/ansi.Strip` and migrate `internal/tui/tui_test.go`, `internal/tui/screens_test.go` to strip-before-compare; verify 5 `internal/tui/testdata/*.golden` bytes identical (→ palette-change-keeps-goldens)
- [x] 3.2 Create `internal/tui/theme_test.go` with fixed ANSI256 exact-escape pins for badges, pills, title, border corners (→ styled-tty-shows)
- [x] 3.3 Add degradation tests for `NO_COLOR`/`CLICOLOR=0`/non-TTY → no color, `TERM=dumb` → degraded, legacy Windows → Ascii with no raw escapes, tokens intact (→ no-color-degrades, legacy-windows-ascii)

## Phase 4: Hygiene

- [x] 4.1 Run `go test ./...`, `gofmt -l`, `go vet ./...`, `golangci-lint run`; keep lipgloss v1.1.0/bubbletea v1.3.4/termenv v0.16.0 pinned (→ all scenarios green)
