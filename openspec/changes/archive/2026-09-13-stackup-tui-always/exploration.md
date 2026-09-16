## Exploration: stackup-tui-always (TUI-first presence for every command)

### Current State
Today only `apply` touches the TUI, and only on one narrow path: pending overwrites
(`Changed && !IsNew`) with no `--yes`, under `stdinIsTTY()` and not
`--non-interactive`. The gate lives in `cmd/stackup/apply.go` (~lines 49-76):
non-TTY/`--non-interactive` prints the blocked message (exit 3) and writes
nothing; TTY opens `newConfirmLauncher(cmd.InOrStdin(), cmd.ErrOrStderr(),
overwrites).Run(plan)` and aborts cleanly (exit 0) on anything-but-apply.

The TUI seam (`internal/tui/`) is deliberately thin per the canonical design
(Engram `sdd/stackup-cli/design`): `Launcher` interface with
`Run(p generate.Plan) (Action, error)` (`tui.go`); a single `Model` confirm
screen over one `generate.Plan` (`model.go` — file list with `(new)` /
`(overwrite)` tags, `y/Y/enter` apply, `n/N/q/esc/ctrl+c` abort, up/k-down/j
navigation, undecided-shutdown-is-Abort, ASCII renderer for byte-identical
goldens); `teaLauncher` wiring stdio (`tea.go`); `StubLauncher` canned answer
(`stub.go`). `detect`/`generate`/`diff` never reach this package — they print
directly via `printPreview` / `printWritten` / `formatEvidence` in
`cmd/stackup/root.go` to `cmd.OutOrStdout()`, with exit codes 0 ok/no-change,
2 preview-has-changes, 3 blocked-non-interactive, 4 unknown-stack, 1 error
(`main.go`). `--format json` emits parseable payloads (`detectPayload`,
`previewPayload`, `writtenPayload`, `blockedPayload`) — the CI contract covered
by `cli_test.go` (`TestDetectJSONIsParseable`,
`TestGenerateDryRunJSONParseableWritesNothing`, `TestDiffPreviewExit2WritesNothing`,
`TestApplyBlockedWithoutPromptInCI`) plus TUI-side `tui_test.go`
(`Model.Update` direct asserts), `teatest_test.go` (one real-program happy
path), and a `View()` golden (`testdata/TestConfirmViewGolden.golden`).
Probes are stub-vars for tests: `stdinIsTTY` (`root.go`) and
`newConfirmLauncher` (`apply.go`).

### Affected Areas
- `cmd/stackup/root.go` — TTY probe (`stdinIsTTY` → stdout-based gate), shared
  `shouldUseTUI()` helper, `printPreview`/`printWritten`/`formatEvidence` stay as
  the plain-text fallback path (byte-identical).
- `cmd/stackup/detect.go`, `generate.go`, `diff.go` — add TUI branch: compute
  result first, then either launch presence screen (TTY, no `--yes`/
  `--non-interactive`, text format) or print as today.
- `cmd/stackup/apply.go` — extend existing gate from confirm-only to
  always-present (result/summary screen even for fresh-only writes and no-change);
  keep exit-3 and abort semantics untouched.
- `internal/tui/tui.go` — `Launcher` interface extension (or new screen
  constructors behind it); decision point of this exploration.
- `internal/tui/model.go` (+ new files) — per-command screens or one shared
  Model with a screen enum; new goldens per view.
- `internal/tui/tea.go`, `stub.go` — launcher wiring for new screens; stub must
  cover new methods so core stays green.
- `cmd/stackup/cli_test.go`, `internal/tui/tui_test.go`, `teatest_test.go`,
  `internal/tui/testdata/` — gate-matrix tests, per-screen `Update` tests,
  minimal teatest, new goldens.
- `openspec/specs/cli-ux/spec.md` — future delta spec will add TUI-presence
  requirement; CI-contract scenarios (exits 2/3/4, JSON parseability) must remain
  verbatim.

### Approaches
1. **Shared Model with Screen enum behind the existing `Launcher` iface (extended payload)** — one `Model{screen, evidence, plan, preview, result}`
   with `ScreenDetect/ScreenGenerate/ScreenDiff/ScreenApply` states; `Launcher`
   gains a screen-aware entry (e.g. `RunScreen(ScreenInput) (ScreenResult, error)`)
   or keeps `Run(p)` with an options struct. Each screen is a `View()` branch
   plus a small `Update()` branch; navigation keys shared.
   - Pros: single key-handling/quit/error path; one stub to maintain; smallest API
     churn; goldens share the ASCII renderer.
   - Cons: `model.go` grows into a switch hub; per-screen state can leak across
     screens without discipline (must reset cursor/decided per screen).
   - Effort: Medium.
2. **Per-command models behind the existing `Launcher` iface (one constructor per screen, iface unchanged)** — `NewDetectModel(ev)`,
   `NewPlanModel(plan)`, `NewDiffModel(preview)`, keep `NewConfirmModel`; each
   command builds its model and calls a generic `Show(model) error` launcher
   (iface gains one `Show(tea.Model)` method instead of `Run(p)`), presence
   screens return no decision (only apply's confirm returns `Action`).
   - Pros: cleanest separation — each screen owns its state/keys; no cross-screen
     leakage; closest to Bubble Tea idioms; easy to add a screen later.
   - Cons: iface change ripples to `apply.go` + stub + all call sites at once;
     shared chrome (title/help line) must be factored into a helper or it drifts.
   - Effort: Medium-High.
3. **Keep confirm-only `Launcher` untouched; presence via a second `Presenter` iface** — `Launcher` stays `Run(p) (Action,error)`
   for apply's gate; new `Presenter` iface (`ShowDetect/ShowPlan/ShowDiff/ShowResult`)
   handles read-only presence. Commands take both; tests stub each independently.
   - Pros: zero risk to the proven confirm gate and its tests; read-only screens
     can never accidentally gate a write; smallest blast radius per command.
   - Cons: two ifaces/wiring paths to maintain; call sites juggle two seams;
     shared styling/input plumbing duplicated unless factored.
   - Effort: Low-Medium (but permanent two-seam tax).

Per-command screen content (same under any approach):
- `detect`: ranked evidence table — one row per finding
  (`ecosystem: confidence (signals: …) pm=… version=…`, same data as
  `formatEvidence`), highlight + `q` quit; empty state mirrors
  `unknown allowed` / exit-4 notice, never fabricates findings.
- `generate` (write path): plan preview — file list with `(new)`/`(overwrite)`
  tags (reuse `fileLine`/`pendingLine` wording), then post-write summary reusing
  `printWritten` lines (`wrote N file(s): … (backup: …)/(new)`); read-only until
  user quits, then the write proceeds (or already proceeded — ordering is a
  proposal decision; recommend show-plan → quit → write → show-summary for apply,
  show-plan → quit → write for generate).
- `diff` (+ `--dry-run`): scrollable unified-diff view — file selector
  (up/down over `preview.Files`) plus viewport over the selected `Unified` text
  (needs `bubbles/viewport`, already in `go.mod` via `bubbles v0.20.0`); footer
  shows exit implication (`changes pending — exit 2` vs `no changes`); quit key
  returns and the command still prints the plain diff to stdout after TUI exit
  OR the TUI replaces stdout display — proposal must pick one (recommend: TUI
  displays, and on quit the command still emits today's stdout bytes so pipes
  after TUI remain consistent… actually if TTY, no pipe; simplest: TUI replaces
  the print, exit code preserved).
- `apply`: keep the existing confirm dialog verbatim (keys, wording, Abort
  default), add a post-write result screen (same `printWritten` content) and a
  no-change screen (`no changes`) so every apply shows presence.

TTY detection (recommendation inside Recommendation below): today `stdinIsTTY`.
Must switch to **stdout-gated** (`stdoutIsTTY`, plus requiring stdin TTY for
screens that read keys). Rationale: `stackup diff | head` has TTY stdin but
piped stdout — opening a fullscreen TUI would swallow the pipe. Conversely
`echo y | stackup apply` has TTY stdout but piped stdin — a key-driven TUI
could never receive keys. Gate rule: presence TUI iff `stdoutIsTTY() &&
stdinIsTTY() && !yes && !nonInteractive && format==text`. `--format json` never
opens TUI (machine output must stay parseable byte-identical).

Fallback when the tea program fails to start: degrade to today's plain-text
path (call `printPreview`/`printWritten`/evidence loop), print
`warning: interactive display unavailable, showing plain output` to stderr,
preserve the engine exit code (0/2/4) — never convert a display failure into
exit 1. Only a confirm-gate failure that leaves the decision unknown should
surface `tui: run confirm screen` as exit 1 (today's `tea.go` behavior, keep for
apply's gate; presence screens must be infallible-by-fallback).

### Recommendation
**Approach 1 (shared Model + Screen enum, iface extended with a screen-aware
entry) with stdout+stdin TTY gating and infallible-presence fallback.** Why: it
is the smallest change that satisfies "every command shows something" while
keeping one stub, one key/quit/error path, and the ASCII golden pipeline. Keep
apply's confirm semantics (keys, wording, Abort default, exit 0 on decline,
exit 3 only for overwrites under non-interactive) byte-identical; add read-only
screens for detect/generate/diff and a result screen for apply. Gate every
presence screen on `stdoutIsTTY() && stdinIsTTY() && !--yes &&
!--non-interactive && --format text`; everything else runs today's print path
untouched. `Launcher` extension sketch for proposal (not final API):
`type ScreenInput struct { Screen ScreenKind; Evidences []Evidence; Plan Plan;
Preview diff.Result; Written apply.Result }` with `RunScreen(ScreenInput)
(ScreenResult, error)`. Testing per go-testing skill: direct `Model.Update()`
table tests per screen (keys, cursor clamp, quit, WindowSizeMsg ignore),
exactly one teatest happy path (diff screen renders → `q` quits), goldens per
`View()` via `-update` then clean rerun, gate-matrix tests in `cli_test.go`
(TTY/non-TTY × `--yes`/`--non-interactive`/json × piped-stdout stub) asserting
byte-identical stdout on the fallback legs and preserved exits 0/2/3/4.
No proposal yet — this exploration only.

### Risks
- Breaking piped-text consumers: any TUI branch that triggers on stdin-TTY
  alone steals piped stdout; must gate on stdout TTY (primary) — gate-matrix
  tests must cover `diff | head` shape via stubbed probes.
- Windows console quirks: `coninput`/`x/windows` + lipgloss/termenv on legacy
  `cmd.exe`/PowerShell 5.1 (this repo's shell) can misrender or fail program
  start; ASCII renderer + graceful fallback mitigate, but needs a manual smoke
  note in verify.
- Golden drift for new views: `View()` strings must stay deterministic (no
  terminal-width sniffing, no timestamps); reuse `pendingLine`/`fileLine`
  wording; update only via `-update` then clean rerun.
- Scope creep into a full diff pager: viewport scrolling, search, huge unified
  diffs (large plans) can balloon the change past the 400-line chained-PR
  budget; cap diff screen at selector + viewport, no search/syntax highlight.
- JSON-format temptation: opening TUI on `--format json` breaks CI parsers;
  hard rule — json never presents, even on TTY.
- Confirm-gate dilution: presence screens must never imply consent; only the
  existing apply dialog returns `ActionApply`, all other screens return on quit
  with zero write semantics.

### Ready for Proposal
Yes — scope is bounded (4 presence screens + gate + fallback + tests), contract
preserved (exits 0/2/3/4, JSON parseability, stub-swap), recommendation concrete.
Orchestrator should tell the user: exploration favors a shared Screen-enum Model
with stdout+stdin gating and text-fallback; next step is `sdd-propose` for
`stackup-tui-always`, then spec deltas on `cli-ux` (presence requirement +
  TTY/json gate scenarios). No implementation was done.
