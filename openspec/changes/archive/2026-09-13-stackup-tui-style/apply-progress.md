# Apply progress: stackup-tui-style

## Slice 1 — Theme core + Ascii default (PR1, stacked-to-main)

### Completed tasks
- [x] 1.1 `internal/tui/theme.go`: `Theme` + `NewAsciiTheme`/`NewTheme(renderer)` with `Title`, `RoundedBorder`, `BadgeHigh`/`BadgeMedium`/`BadgeLow`, `PillNew`/`PillOverwrite`, `Footer`/`FooterEmph`; adaptive triad green `#1A7F37`/`#3FB950`, amber `#9A6700`/`#D29922`, grey `#59636E`/`#9198A1`; spacing within 0/1/2.
- [x] 1.2 `theme` field on `Model` (`internal/tui/model.go`); zero value renders Ascii via `currentTheme()` fallback; constructor signatures and `View()` output unchanged (byte-identical).

### Work Unit Evidence
| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/tui/ -run TestTheme -count=1`: ok (4 tests, 15 subtests, PASS) |
| Runtime harness command/scenario and exact result | N/A — no runtime path; Ascii default keeps `View()` deterministic (per Unit 1 row) |
| Rollback boundary | Remove `internal/tui/theme.go` + `internal/tui/theme_test.go`; revert `theme` field in `internal/tui/model.go` |

### Verification (foreground)
- `go build ./...`: clean, no output
- `go test ./internal/tui/ -run TestTheme -count=1`: ok
- `go test ./... -count=1`: all 7 packages ok, no regressions
- `gofmt -l .`: clean (one alignment fix applied to `theme.go`)
- `go vet ./...`: clean

### Deviations from design
None — implementation matches design. `theme.go` reuses the existing package-level `ascii` renderer instead of creating a second one; `NewTheme(nil)` is nil-safe (falls back to Ascii).

### Remaining (next slices)
- Slice 2 (Unit 2): tasks 2.1–2.3 — screen glow-up + runtime injection in `tea.go`.
- Slice 3 (Unit 3) + hygiene: tasks 3.1–3.3, 4.1 — strip helper, exact-escape pins (extends `theme_test.go`), degradation matrix.

## Slice 2 — Screen glow-up + runtime injection (PR2, stacked-to-main)

### Completed tasks
- [x] 2.1 `internal/tui/model.go`: `evidenceLine`/`fileLine`/`resultLines`/`pendingLine` are now `Model` methods wrapping whole tokens only — confidence word via `BadgeHigh`/`BadgeMedium`/`BadgeLow`, `(new)` via `PillNew`, `(overwrite)` and `(backup: …)` via `PillOverwrite`; titles via `Title`, hint footers and `pendingLine` via `Footer`; markers/paths outside styles; `no changes` and the empty-evidence notice unstyled.
- [x] 2.2 `internal/tui/diffview.go`: viewport wrapped in `RoundedBorder` when the theme draws it; `Model.diffWidth` shrinks content by 2 cols at 80 (78 content + border = 80); `diffTag` wraps `(new)`/`(changed)`/`(unchanged)` whole (`PillNew`/`PillOverwrite`/`Footer`); `diffFooter` emphasizes whole token `exit 2` via `FooterEmph`; clean-tree notice unstyled; no `+`/`-` tint.
- [x] 2.3 `internal/tui/tea.go`: `modelForInput(inp, out)` resolves the renderer from `out`+env via `lipgloss.NewRenderer(out)` (termenv default detection, never a forced profile) through new `runtimeRenderer` seam; nil `out` falls back to Ascii; constructors stay zero-arg Ascii.
- [x] Unit tests: new `internal/tui/style_test.go` (behavioral, no golden migration) — whole-token wrapping per screen, markers/notices plain, all three `diffTag` states, border width-2 math + Ascii golden guard, `modelForInput` injection + non-TTY degradation + nil-out fallback.
- [x] `internal/tui/theme.go`: `Theme` gains unexported `bordered`; `NewTheme` draws the border, `NewAsciiTheme` is borderless (plain layout for goldens/pipes/legacy), `NewTheme(nil)` returns the Ascii theme; new `drawsBorder()` capability query.

### Work Unit Evidence
| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/tui/ -run TestEvidenceView -count=1`: PASS (shows + golden); `go test ./internal/tui/ -run TestStyle -count=1`: PASS (5 new tests); `go test ./internal/tui/ -run TestModelForInput -count=1` + `TestRuntimeRenderer` + `TestDiffBorder`: PASS |
| Runtime harness command/scenario and exact result | `go test ./internal/tui/ -run TestConfirmInteractive -count=1`: PASS; `TestPresenceInteractiveQuitAborts` (live Bubble Tea, all 4 presence screens): PASS |
| Rollback boundary | Revert `internal/tui/model.go`, `internal/tui/diffview.go`, `internal/tui/tea.go` to plain `View()`; remove `bordered`/`drawsBorder` from `internal/tui/theme.go`; delete `internal/tui/style_test.go` |

### Verification (foreground)
- `go build ./...`: clean, no output
- `go test ./internal/tui/ -count=1`: ok (full package incl. 5 goldens + live teatest)
- `go test ./... -count=1`: all 7 packages ok, no regressions
- Existing 5 goldens (`TestConfirm/Evidence/Plan/Diff/ResultViewGolden`): PASS byte-identical, files untouched
- `gofmt -l .`: clean (no output)
- `go vet ./...`: clean (no output)

### Deviations from design
- Ascii theme is borderless by construction (`NewAsciiTheme` clears `RoundedBorder`, `bordered=false`): a drawn border is structural bytes, not escapes, so any Ascii border would churn goldens — slice-2 contract requires them byte-identical. Runtime theme draws the border; slice-3 pins its corners under ANSI256.
- `diffWidth` shrinks the viewport only when the theme draws the border (Ascii keeps full width): same golden-stability reason; at 80 cols runtime shows 78 content + 2 border = 80.
- `diffTag` mapping `(new)`→`PillNew`, `(changed)`→`PillOverwrite`, `(unchanged)`→`Footer`: design pins whole-token styling but not the exact style per tag; amber-for-changed and dim-for-unchanged stay within the green/amber/grey triad and caps.
- `pendingLine` takes the `Footer` (dim subtitle) style as one whole line: wording untouched, no count token split.
- Test seam uses `Renderer.SetColorProfile(ANSI256)` (lipgloss's testing API): `termenv.WithProfile` alone still degrades non-TTY writers to Ascii, so it cannot force color in tests. Production path passes no profile option at all. Slice-3 pins can reuse the `ansi256Theme()` helper seam.

### Remaining (next slice)
- Slice 3 (Unit 3) + hygiene: tasks 3.1–3.3, 4.1 — strip helper (`x/ansi`), golden harness migration (bytes stay identical), exact-escape pins in `theme_test.go` (can build on `ansi256Theme()` seam), degradation matrix (`NO_COLOR`/`CLICOLOR=0`/non-TTY/`TERM=dumb`/legacy Windows via `runtimeRenderer` + `modelForInput` seams), lint + pins check.

## Slice 3 — Golden migration + pins + matrix + hygiene (PR3, stacked-to-main, FINAL)

### Completed tasks
- [x] 3.1 Strip helper + harness migration: `stripANSI` via `x/ansi.Strip` in `internal/tui/tui_test.go` (shared by package `tui_test`); all 5 golden compares strip before `teatest.RequireEqualOutput`; `internal/tui/screens_test.go` content assertions strip before compare; all 5 `internal/tui/testdata/*.golden` bytes identical (`git status -- testdata` clean).
- [x] 3.2 Exact-escape pins in `internal/tui/theme_test.go` under the slice-2 `ansi256Theme()` seam: `TestThemeANSI256ExactEscapes` (title `\x1b[1m`, badges `1;38;5;71/172/103`, pills mirror badges, footer `38;5;103`, emph `1m`, each with stripped-token check) + `TestThemeANSI256BorderCorners` (╭╮╰╯ in grey `38;5;103`, stripped box exact).
- [x] 3.3 Degradation matrix in `internal/tui/theme_test.go` (no forced profile, never parallel where env is mutated): `TestRuntimeDegradationMatrix` (non-TTY/`NO_COLOR`/`CLICOLOR=0`/`TERM=dumb` → `termenv.Ascii` profile, no escapes, tokens intact via `runtimeRenderer` + `modelForInput`), `TestRuntimeDegradationKeepsTokens` (evidence `high`, diff `exit 2`, result `(new)` verbatim under `NO_COLOR`), `TestLegacyWindowsAsciiFallback` (Ascii views: no `\x1b`, no `╭` chrome, tokens intact on all 5 screens), `TestStyledStrippedMatchesAscii` (stripped forced-color view == Ascii bytes on borderless screens; diff only adds border chrome).
- [x] 4.1 Hygiene: `go build ./...` clean, `go test ./... -count=1` all 7 packages green, golden clean rerun with zero `testdata` churn, `gofmt -l .` clean, `go vet ./...` clean, `golangci-lint run ./...` 0 issues; lipgloss v1.1.0 / bubbletea v1.3.4 / termenv v0.16.0 confirmed pinned. Pre-existing dirt (`go.mod`/`go.sum` termenv-promotion + cellbuf lines, `package.json`/`package-lock.json`, `README.md`, `stackup.exe`) left untouched and unstaged.

### Work Unit Evidence
| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./... -count=1`: all 7 packages ok (cmd/stackup, apply, detect, diff, generate, merge, tui) |
| Runtime harness command/scenario and exact result | `go test ./internal/tui -run TestPresenceInteractive -count=1`: PASS (4/4 presence screens quit-as-Abort, live Bubble Tea); `TestConfirmInteractiveApplies`: PASS |
| Rollback boundary | Revert `internal/tui/tui_test.go` + `internal/tui/screens_test.go` strip helper/migration; revert `internal/tui/theme_test.go` slice-3 additions; goldens bytes unchanged so safe |

### Verification (foreground)
- `go build ./...`: clean, no output
- `go test ./... -count=1`: all 7 packages ok, no regressions
- `go test ./internal/tui -run TestPresenceInteractive -count=1`: PASS (4/4); `-run TestConfirmInteractive`: PASS
- Existing 5 goldens: PASS stripped-compare, files byte-identical (`git status -- internal/tui/testdata` empty)
- `gofmt -l .`: clean (no output)
- `go vet ./...`: clean (no output)
- `golangci-lint run ./...`: 0 issues

### Deviations from design
- `screens_test.go` has no `RequireEqualOutput` goldens (all 5 live in `tui_test.go`); its migration wraps content assertions with the shared `stripANSI` helper instead — same strip-before-compare contract, no-op on Ascii output.
- Exact pin values assume the dark triad under the forced ANSI256 seam (`71`/`172`/`103`); bold-only styles (`Title`, `FooterEmph` → `\x1b[1m`) are background-independent. The seam reuses `ansi256Theme()` from `style_test.go` as slice-2 foresaw.
- `CLICOLOR_FORCE` was probed but deliberately not pinned: forcing semantics on non-TTY writers are implementation-defined, while the spec matrix only requires the degrade-by-default path.

### Unit size
- `git diff --stat` for the unit: 3 test files, +234/−18 (≈252 changed lines) — inside the 400-line budget, single PR, no `size:exception`.
