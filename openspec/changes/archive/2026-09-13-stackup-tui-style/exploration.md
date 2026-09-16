# Exploration: stackup-tui-style ("Charm vibrante")

## Current State

- `internal/tui/model.go` renders all five screens through one package-level
  `ascii` renderer: `lipgloss.NewRenderer(io.Discard, termenv.WithProfile(termenv.Ascii))`.
  Titles use `ascii.NewStyle().Bold(true)` (no-op escapes under Ascii); every
  other line is a hand-joined plain string ending in `"\n"`. `View()` is fully
  deterministic by construction.
- `internal/tui/diffview.go` owns the diff screen: selector rows
  (`(new)`/`(changed)`/`(unchanged)`), a `bubbles/viewport` with fixed defaults
  (78x12, resized by `WindowSizeMsg` to `height-8`), and `diffFooter` stating the
  exit-2 implication. Viewport content is the raw unified diff, unstyled.
- `internal/tui/tea.go` wires stdio into `tea.NewProgram` with no color/profile
  options; Bubble Tea v1.3.4 exposes no `WithColorProfile`, so color can only
  flow through a Lip Gloss renderer bound to the program output + env.
- `cmd/stackup/` (`root.go`, `detect.go`, `generate.go`, `diff.go`, `apply.go`)
  owns the gate (`shouldUseTUI`: both TTYs, text format, no `--yes`/
  `--non-interactive`), the exits (0/1/2/3/4), and the plain-text fallback
  (`printPreview`, `printWritten`, `formatEvidence`) which MUST stay
  byte-identical. JSON output never presents.
- Pins: `lipgloss v1.1.0` (renderer-bound `Renderer.NewStyle()` API),
  `bubbletea v1.3.4`, `termenv v0.16.0`. Five goldens under
  `internal/tui/testdata/` via `teatest.RequireEqualOutput` (`-update` path from
  `x/exp/golden`); substring tests (`strings.Contains`) assert tokens like
  `"node: high"`, `"exit 2"`, `"(overwrite)"`, `"wrote 2 file(s):"`.

## Affected Areas

- `internal/tui/model.go` — `ascii` renderer, `viewConfirm/viewEvidence/viewPlan/viewResult`,
  `evidenceLine/fileLine/pendingLine/resultLines` helpers (style whole tokens, never split them).
- `internal/tui/diffview.go` — `renderDiff`, selector tags, viewport border wrapper
  (border adds 2 cols: shrink viewport width accordingly), `diffFooter` emphasis.
- `internal/tui/tea.go` — build runtime renderer from output + env, inject into Model
  (new `internal/tui/theme.go` suggested; Model gains theme/renderer field, default keeps Ascii).
- `internal/tui/*_test.go` + `testdata/*.golden` — golden strategy change; substring
  assertions move to ANSI-stripped output (see Recommendation).
- NOT touched: `cmd/stackup/*` gate/exits/fallback/JSON, `Launcher`/`StubLauncher`
  interface, `Decision`/`Confirmed` semantics, `detect.Confidence` type.

## Approaches

### 1. Explicit-profile goldens (goldens contain ANSI escapes)

Pin goldens to a fixed renderer, e.g. `termenv.ANSI256`, so styled output is
captured byte-exact including escapes.

- Pros: pixel-pins colors; catches any style regression; single assertion path.
- Cons: goldens become escape soup (unreadable PR diffs); ANY palette tweak churns
  all 5 goldens; brittle across lipgloss/termenv upgrades; width/profile coupling.
- Effort: Medium.

### 2. ANSI-stripped goldens + focused style unit tests (RECOMMENDED)

Keep the 5 view goldens asserting text semantics only (strip ANSI before
`RequireEqualOutput`, so current golden files stay byte-identical = zero migration
churn). Pin colors separately in a small `theme_test.go` with exact-escape
assertions under a fixed `ANSI256` renderer (badges, pills, title, border).

- Pros: palette iteration touches only theme tests, never view goldens; PR diffs stay
  human-readable; existing substring assertions keep working if run on stripped output;
  determinism preserved (strip is deterministic; style tests pin escapes exactly).
- Cons: a color regression outside the pinned snippets could slip through (mitigated:
  route ALL styling through the Theme so snippets cover every style constructor).
- Effort: Medium.

### 3. Dual goldens (plain + styled per screen)

Keep current goldens and add 5 styled companions.

- Pros: pins both layers fully.
- Cons: 10 goldens to maintain; every wording change needs two `-update` runs and
  double review; worst churn/readability ratio; no extra safety over option 2.
- Effort: High.

## Recommendation

**Option 2.** Rationale, mapped to the confirmed direction:

- *Personality without breaking determinism*: styling enters through one new
  `internal/tui/theme.go` (`Theme` struct: renderer + named styles). `Model` carries
  the theme; constructors default to the current Ascii behavior, tests inject either
  Ascii (goldens, zero diff) or fixed `ANSI256` (style tests, exact escapes).
  Runtime (`teaLauncher.Run`) builds the renderer from the program output + env with
  default termenv detection — never a forced profile.
- *Golden determinism*: stripping before compare keeps the `-update` workflow from the
  go-testing skill intact (`-update`, inspect diff, rerun clean) while making the
  migration itself a no-op for the 5 existing files.
- *NO_COLOR / dumb terminals*: free via termenv — `NO_COLOR!= ""` or `CLICOLOR=0`
  disables color (`termenv.go`), non-TTY output reports `Ascii`, `TERM=dumb`-family
  degrades on Unix. Rule: runtime code MUST NOT pass `WithProfile(force)`,
  `WithUnsafe`, or color cache tricks; explicit profiles live in tests only.
- *Windows*: Bubble Tea owns the console; legacy pre-10/build-10586 consoles degrade
  to Ascii per `termenv_windows.go`. Do NOT call `EnableWindowsANSIConsole` manually.
  No new console-mode code.
- *Shared theme sketch*: palette as fixed ANSI numbers (deterministic under ANSI256)
  wrapped in `AdaptiveColor`/`LightDark` for dark/light backgrounds; `RoundedBorder`
  reserved for header + diff viewport frame; badge styles `badgeHigh` (green),
  `badgeMedium` (amber), `badgeLow` (grey/red-muted); pills `pillNew` (green),
  `pillOverwrite` (amber); footer style with exit-code emphasis; spacing scale
  0/1/2 (padding inside bordered cards only, never on raw rows, to protect the
  80-col assumptions and copy-paste).
- *Per-screen glow-up sketch*:
  - Evidence: confidence word rendered through `badgeHigh/Medium/Low` (same lowercase
    token, so stripped semantics unchanged); header card.
  - Plan/Confirm: `(new)`/`(overwrite)` through `pillNew`/`pillOverwrite`;
    `pendingLine` counts wording untouched.
  - Diff: selector rows keep path + tag tokens (style whole row); viewport wrapped in
    rounded border (width -2); footer keeps `exit 2` wording, emphasized.
  - Result: green success title tone; `(new)`/`(backup: …)` pills; `no changes` kept.
  - Viewport unified content: keep raw (copy-paste safe); `+`/`-` tinting explicitly
    deferred (highest over-styling risk, lowest value).
- *Accessibility*: never color-only — every badge/pill keeps its text label; use
  `LightDark` adaptive pairs; plain-text fallback path already covers screen readers.
- *What NOT to touch*: exits, `shouldUseTUI`, fallback printers, JSON, `Launcher`
  interface, `Decision`/`Confirmed`, `detect.Confidence` strings, lipgloss major
  version (v1.1.0 pinned; v2 removes `Renderer` — upgrading would force a rewrite and
  is out of scope).

## Risks

- Golden churn on palette tweaks — contained by option 2 (theme-only test updates).
- Snapshot readability in PRs — contained by stripped goldens; styled output reviewed
  via `theme_test.go` snippets, not 5 escape dumps.
- Over-styling hurting readability — cap the system (one border set, 3 badges, 2 pills,
  no viewport content tinting in v1); keep 80-col layout math explicit.
- Token-splitting breaking substring/teatest waits — rule: style whole tokens only
  (`teatest.WaitFor` on `[]byte(".vscode/settings.json")` survives wrapping, not splitting).
- Lipgloss v2 temptation — out of scope; v2 deletes the `Renderer` API this plan builds on.
- Windows legacy consoles emitting raw escapes — termenv Ascii fallback + Bubble Tea VT
  handling cover it; verify on Win10+ during `sdd-verify`, no custom code.

## Ready for Proposal

Yes. Proposal input: change `stackup-tui-style`, direction "Charm vibrante" per this
exploration, golden strategy option 2, `theme.go`-shaped design, touch-list above.
Open question for propose (not blocking): exact ANSI palette numbers (suggest
green/amber/grey triad, adaptive pairs) — designer choice, no user clarification needed.
