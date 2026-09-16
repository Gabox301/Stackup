# Design: stackup-tui-always

## Technical Approach

Compute-first, present-after: each command builds its result with the current engine call, then branches on `shouldUseTUI()`. Qualifying runs launch one shared `Model` (screen enum) via the extended `Launcher`; every other run and every launch failure uses today's print path byte-identical, with engine exits (0/1/2/3/4) preserved. Only `apply` confirm returns `ActionApply`; presence screens return `ActionAbort` and never gate writes. Maps to proposal §Approach and spec scenarios TTY-shows/confirm/piped/json/yes/fallback.

## Architecture Decisions

| Decision | Options (tradeoff) | Choice + rationale |
|----------|--------------------|--------------------|
| Model shape | Shared enum (one key/quit path, switch-hub risk) vs per-command types (idiomatic, 4× chrome + iface ripple) vs 2nd `Presenter` iface (zero confirm risk, permanent two-seam tax) | Shared `Model{screen,…}` with per-screen constructors. 4 read-only screens do not justify 4 types within review budget; constructors reset cursor/decided, viewport state only on diff screen. |
| Launcher extension | `Run(Input)` (one method, stub asserts payload) vs `Show(tea.Model)` (callers build models, stub asserts weakly) vs keep `Run(p)` + new method (compat, two paths) | `Run(Input) (Action, error)`; `Input` carries screen + payload. Single construction site in `tea.go`; `StubLauncher` records `Inputs`, core stays green. `apply.go` migrates once. |
| Gate | stdin-only (today; breaks `diff \| head`) vs stdout+stdin (correct both directions) | `stdoutIsTTY() && stdinIsTTY() && !yes && !nonInteractive && format==text`. Both probes are stub-vars. `--format json` never presents. |
| Diff content | Plain scroll (simple, unbounded unified text overflows) vs `bubbles/viewport` (already pinned, needs size msgs) | Selector + `viewport.Model`, footer states exit-2 implication. Size msgs update viewport only; other screens ignore them as today. |
| After-TUI output | Re-print plain text (double output on TTY) vs TUI replaces print (single output, exit preserved) | TUI replaces the print; quit exits with engine code; only fallback prints. |

## Data Flow

    cmd ── compute (Detect/Build/Compute) ──┐
         │ shouldUseTUI()? ── no ──▶ fallback print (today's fns) ──▶ exit   │
         └─ yes ──▶ Launcher.Run(Input) ── ok ──▶ quit ──▶ exit              │
                            └── err ──▶ stderr warning + fallback print ──▶ exit ─────┘

Sequence (apply; others omit confirm/write):

    apply ──▶ Compute ──▶ shouldUseTUI? ──▶ Run(Confirm) ──▶ Abort? exit 0
                                                        ──▶ Apply? write ──▶ Run(Result) ──▶ exit 0

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/tui.go` | Modify | Add `Screen` enum + `Input` struct; `Launcher` becomes `Run(Input) (Action,error)` |
| `internal/tui/model.go` | Modify | `Model` gains `screen` + payload fields, constructors, `View`/`Update` switch |
| `internal/tui/diffview.go` | Create | Diff selector + viewport helpers (keeps `model.go` small) |
| `internal/tui/tea.go` | Modify | `NewTeaLauncher(in,out,opts…)`; `Run` builds model from `Input` switch |
| `internal/tui/stub.go` | Modify | Record `Inputs []Input`; canned `Action`/`Err` unchanged |
| `cmd/stackup/root.go` | Modify | Add `stdoutIsTTY` stub-var + `shouldUseTUI(shared)`; prints stay as fallback |
| `cmd/stackup/detect.go`, `generate.go`, `diff.go` | Modify | Compute-then-branch: TUI or fallback; exits unchanged |
| `cmd/stackup/apply.go` | Modify | `Run(Confirm)` + `Run(Result)`; exit-3/abort kept |
| `internal/tui/testdata/Test*View.golden` | Create | Per-screen goldens (keep Confirm) |

## Interfaces / Contracts

```go
type Screen int
const (ScreenConfirm Screen = iota; ScreenEvidence; ScreenPlan; ScreenDiff; ScreenResult)

type Input struct {
    Screen; Evidence []detect.Evidence; Plan generate.Plan
    Pending []string; Preview diff.Result; Written apply.Result
}
type Launcher interface{ Run(Input) (Action, error) }

func NewEvidenceModel(ev []detect.Evidence) Model
func NewPlanModel(p generate.Plan, pending []string) Model
func NewDiffModel(r diff.Result) Model
func NewResultModel(res apply.Result) Model // + existing NewConfirmModel
```

```go
var stdoutIsTTY = func() bool { /* os.Stdout stat */ }
func shouldUseTUI(s *sharedOpts) bool {
    return stdoutIsTTY() && stdinIsTTY() && !s.yes && !s.nonInteractive && s.format == "text"
}
```

Content: Evidence reuses `formatEvidence` (empty mirrors exit-4); Plan reuses `pendingLine`/`fileLine`, quit-then-write; Diff is selector + viewport over `Unified` with exit-2 footer; Result reuses `printWritten` plus `no changes`. Confirm verbatim; undecided stays Abort.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `shouldUseTUI` gate matrix (stdout×stdin×yes×nonInteractive×format); per-screen `Update` (keys, clamps, viewport-only-on-diff) | Table-driven, direct `Update()`, stubbed probes |
| Golden | `View()` per screen, fixed 80×30, ASCII | `teatest.RequireEqualOutput`; `-update` then clean rerun |
| Integration | Launch-failure fallback (stub `Err` → identical stdout + warning + exit); writes never gated | Stub `Launcher`; `testing.Short()`-safe, `t.TempDir()` |
| Interactive | One happy path per new screen (quit→Abort); keep confirm y/n | `teatest.NewTestModel`, minimal; existing confirm test stays |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. No flags/config/data; fallback preserves bytes/exits behind the TTY gate, reversible per proposal rollback.

## Open Questions

- None blocking. Viewport golden pins 80×30; adjust only if `bubbles/viewport` wrap differs.
