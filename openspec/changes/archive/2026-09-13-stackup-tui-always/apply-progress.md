# Apply progress: stackup-tui-always — Work Units 1–3 (TUI core + gate/branches + goldens/matrix/hygiene, COMPLETE)

## Scope (cumulative)

Slice 1 (committed): Phase 1, tasks 1.1–1.5, TUI core. Slice 2 (committed):
Phase 2, tasks 2.1–2.5, gate + 4 branches + fallback. Slice 3 (this unit,
FINAL): Phase 3, tasks 3.1–3.4, goldens + matrix + per-screen Update +
quit→Abort, plus Phase 4, task 4.1, hygiene. Chained delivery,
stacked-to-main, PR 3 slice.

## Completed

- [x] 1.1 `internal/tui/tui.go`: `Screen` enum
      (Confirm/Evidence/Plan/Diff/Result) + `String()`, `Input` struct
      (Screen, Evidence, Plan, Pending, Preview, Written),
      `Launcher` is now `Run(Input) (Action, error)`.
- [x] 1.2 `internal/tui/model.go`: `Model` gains `screen` + payload
      fields (`evidence`, `plan`, `preview`, `written`, `diffIndex`,
      `viewport`); 4 constructors (`NewEvidenceModel`, `NewPlanModel`,
      `NewDiffModel`, `NewResultModel`) + verbatim `NewConfirmModel`;
      `View`/`Update` switch per screen, ASCII deterministic. Confirm
      keys/wording byte-identical; presence screens quit as Abort and
      never confirm (y/n/enter also quit as Abort).
- [x] 1.3 `internal/tui/diffview.go` (new): selector over
      `preview.Files` with `(new)`/`(changed)`/`(unchanged)` tags,
      `viewport.Model` with deterministic defaults resized on
      `WindowSizeMsg`, footer states the exit-2 implication.
- [x] 1.4 `internal/tui/tea.go`: `NewTeaLauncher(in, out, opts...)`
      (pending moved into `Input`); `Run` builds the model via
      `modelForInput` switch, unknown screens fall back to confirm.
- [x] 1.5 `internal/tui/stub.go`: records `Inputs []Input`; canned
      `Action`/`Err` unchanged; `Plans` mirror kept so command tests
      keep asserting the carried plan.
- [x] Minimal break-fix outside `internal/tui`: `cmd/stackup/apply.go`
      confirm call migrated to `Run(Input{ScreenConfirm, ...})`;
      constructor seam left intact for slice 2. `internal/tui/tui_test.go`
      stub calls migrated to the new shape. No other `cmd/` change.
- [x] New `internal/tui/screens_test.go`: constructor screen report,
      evidence rows + empty exit-4 notice, presence nav clamps,
      quit-is-Abort matrix (incl. y/enter never confirming),
      size/unbound-key ignore, diff selector + exit-2 footer + clean
      notice, result written/no-change lines, plan confirm wording,
      stub `Inputs` recording.

## Verification (foreground)

- `go build ./...`: clean.
- `go test ./internal/tui/ -count=1`: ok (includes the pre-existing
  teatest confirm happy path).
- `go test ./... -count=1`: ok, all 7 packages.
- `gofmt -l .`: clean (after `-w` on apply.go, diffview.go, model.go).
- `go vet ./...`: clean.

## Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/tui/ -count=1` → ok (0.6–0.7s) |
| Runtime harness command/scenario and exact result | N/A — no CLI wiring yet; unit proof only (gate + branches are slice 2) |
| Rollback boundary | Revert `internal/tui/` core (`tui.go`, `model.go`, `diffview.go`, `tea.go`, `stub.go`, `screens_test.go`, `tui_test.go` stub calls) + the `apply.go` confirm-call hunk; confirm path intact |

## Deviations from design

- `StubLauncher` keeps the `Plans` mirror alongside new `Inputs`
  (design names only `Inputs`): avoids breaking `cmd/stackup`
  assertions before slice 2 migrates them to screen-aware checks.
- `newConfirmLauncher(in, out, pending)` keeps its signature with a
  slice-2 TODO instead of collapsing to `(in, out)` now: keeps
  `cli_test.go` compiling untouched for this slice.
- Confirm launch error widened from `tui: run confirm screen` to
  `tui: run <screen> screen`: required by the multi-screen `Run`;
  slice 2 owns fallback wording.

## Issues found

- First pass dropped `tea.Quit` on confirm decide keys (caught by
  `TestConfirmInteractiveApplies` timeout); fixed by returning
  `(Model, tea.Cmd)` from `updateConfirm`/`updatePresence`.

## Remaining (after slice 3: none — all tasks complete)

- [x] Phase 3 (3.1–3.4): goldens + matrix — done in slice 3 (this unit).
- [x] Phase 4 (4.1): hygiene — done in slice 3 (this unit).

## Work Unit 2 (this slice): gate + branches + fallback

### Completed

- [x] 2.1 `cmd/stackup/root.go`: `stdoutIsTTY` stub-var, `shouldUseTUI`
      (`stdoutIsTTY && stdinIsTTY && !yes && !nonInteractive &&
      format==text`), `newTUILauncher(in, out)` seam (pending moved into
      `Input`), `warnTUIFallback` (`warning: interactive display
      unavailable, showing plain output: …` on stderr). Prints stay as the
      fallback path.
- [x] 2.2 `cmd/stackup/detect.go`: compute-then-present Evidence; TTY
      success returns exit 0 with no print, launch failure warns + falls
      back byte-identical.
- [x] 2.3 `cmd/stackup/generate.go`: dry-run shows Plan then exit 2/0
      with no print; write path shows Plan, then writes on quit with no
      second print; launch failure warns, then writes + prints fallback.
- [x] 2.4 `cmd/stackup/diff.go`: compute-then-present Diff; TTY success
      maps to exit 2/0 with no print, failure warns + `printPreview`
      fallback.
- [x] 2.5 `cmd/stackup/apply.go`: dry-run shows Diff; overwrites gate on
      `!shouldUseTUI` blocks exit 3 byte-identical (covers piped stdout,
      piped stdin, non-interactive, json-never); qualifying runs
      `Run(Confirm)` (launch failure warns + safe abort exit 0, never 1)
      then write then `Run(Result)` (failure warns + `printWritten`
      fallback); fresh writes always present Result with no confirm.
- [x] Tests: new `cmd/stackup/gate_test.go` (`TestShouldUseTUI`, 8
      gate legs) + `cmd/stackup/branches_test.go` (Evidence/Diff/Plan/
      write-then-Result/fallback legs); `cli_test.go` seam migrated to
      `newTUILauncher(in, out)`, TTY tests stub both probes,
      `TestApplyTUISwapKeepsCoreGreen` now expects confirm+result (2
      calls) and abort-only-confirm (1 call).

### Verification (foreground)

- `go build ./...`: clean.
- `go test ./cmd/stackup -run TestShouldUseTUI -count=1`: PASS (8/8 subtests).
- `go test ./cmd/... -count=1`: ok.
- `go test ./... -count=1`: ok, all 7 packages.
- `gofmt -l .`: clean.
- `go vet ./...`: clean.
- Runtime harness: `go run ./cmd/stackup diff --path testdata/node`
  prints the unified fallback text with zero writes; engine exit 2
  surfaces as `exit status 2` through the `go run` wrapper (Cobra-level
  exit 2 is proven by `TestDiffPreviewExit2WritesNothing` +
  `TestDiffPresentsDiffScreen`).

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./cmd/stackup -run TestShouldUseTUI -count=1` → PASS (8/8 subtests, 0.4s) |
| Runtime harness command/scenario and exact result | `go run ./cmd/stackup diff --path testdata/node` → unified fallback text, zero writes, engine exit 2 (`exit status 2` via go-run wrapper; `testdata/small` from the task row does not exist, `testdata/node` used) |
| Rollback boundary | Revert `cmd/stackup/` branches + gate (`root.go`, `detect.go`, `generate.go`, `diff.go`, `apply.go`, `cli_test.go`, `gate_test.go`, `branches_test.go`); `internal/tui/` core untouched |

### Deviations from design

- Launcher seam renamed `newConfirmLauncher(in, out, pending)` →
  `newTUILauncher(in, out)` and moved to `root.go`: it now serves all
  five screens, so the confirm-only name lied; pending travels in `Run`
  `Input` per design. `cli_test.go` stub updated to the new shape.
- Apply blocked condition widened from `nonInteractive || !stdinIsTTY`
  to `!shouldUseTUI`: piped stdout (`apply | head`) and `--format json`
  on TTY now block exit 3 byte-identical instead of opening confirm.
  Required by piped-stays-text / json-never; wording and exits kept.
- Confirm launch failure warns + safe-aborts exit 0 (no writes) instead
  of returning the tea error exit 1: required by the fallback rule
  (engine exit never coerced to 1); undecided stays Abort.
- Apply dry-run presents the Diff screen (preview content) rather than
  staying text-only: every apply run presents something; exits 2/0 kept.
- Fallback warning wording fixed as `warning: interactive display
  unavailable, showing plain output: <err>` (exploration §Fallback
  wording + tea `tui: run <screen> screen` error detail); slice 3 matrix
  asserts this prefix.
- Generate write path computes one read-only `diff.Compute` for the
  Plan pending tags before presenting; non-TUI runs skip it and keep
  today's write path byte-identical.

### Issues found

- `TestApplyTUISwapKeepsCoreGreen` assumed 1 confirm call per apply;
  always-present makes it confirm+result (2 calls, `[confirm result]`)
  and abort-only-confirm (1 call): updated in place.
- Task-row harness path `testdata/small` does not exist in the repo;
  `testdata/node` used as the real fallback harness instead.
- None blocking; no design gaps beyond the documented deviations.

## Workload / PR boundary (slices 1–2 record)

- Mode: chained PR slice (stacked-to-main), PR 2 = Unit 2.
- Boundary: `cmd/stackup/` gate + 4 branches + fallback + co-located
  unit tests. No `internal/tui/` core changes, no goldens, no matrix.
- Unit 2 authored size: counted at commit time via `git diff --stat`
  (tracked) + new `gate_test.go`/`branches_test.go`; reported in the
  return summary with a `size:exception` note if over the 400-line
  budget. The unit is already the smallest cohesive slice (gate without
  branches would not compile; branches without fallback would break the
  spec rule), so overage goes with `size:exception`, never code-golf.
- Cumulative record: PR 1 (Unit 1) was ~715 additions / ~70 deletions
  across 8 paths (incl. 335 lines of new unit tests) with
  `size:exception`.

## Work Unit 3 (this slice, FINAL): goldens + matrix + hygiene

### Completed

- [x] 3.1 `cmd/stackup/cli_test.go`: `TestGateMatrixBytesIdentical`
      (text piped-stdout/piped-stdin/yes/non-interactive × detect/diff/
      generate-dry-run/apply-dry-run + json-never × detect/diff/
      generate-dry-run + apply-blocked exit-3 legs; all byte-identical
      stdout+stderr+exit, stub Calls==0) and
      `TestLaunchFailureFallbackBytesIdentical` (read-only detect/diff/
      generate-dry-run/apply-dry-run strict stdout-identical + warning +
      engine exit preserved + zero writes; apply-confirm safe-abort exit 0
      no writes; fresh-apply result fallback warns + prints summary +
      writes). Harness uses `testdata/node` via `nodeProject`:
      task-row `testdata/small` does not exist (same as slice 2).
- [x] 3.2 `internal/tui/testdata/`: per-screen 80x30 ASCII goldens
      `TestEvidenceViewGolden`, `TestPlanViewGolden`,
      `TestDiffViewGolden`, `TestResultViewGolden` (existing
      `TestConfirmViewGolden` kept). Each applies
      `WindowSizeMsg{80,30}` before `View()` so the diff viewport pins to
      80x22; other screens ignore size by design. Refreshed with `-update`,
      inspected, then clean rerun green.
- [x] 3.3 `internal/tui/tui_test.go`:
      `TestPresenceScreensUpdatePerScreen` (evidence/plan/result cursor
      clamps, quit→Abort incl. y/n/enter never confirming, size/unbound
      ignore) + `TestDiffScreenUpdateKeysAndViewport` (selector moves/
      clamps, size resizes only diff and never decides, quit→Abort,
      unbound ignore).
- [x] 3.4 `internal/tui/teatest_test.go`:
      `TestPresenceInteractiveQuitAborts` (one live 80x30 program per
      presence screen: wait-for-render, send q, `WaitFinished`, assert
      Decided + Abort + never Confirmed). Existing
      `TestConfirmInteractiveApplies` (y→Apply) kept for
      apply-confirm-unchanged. Minimal deterministic; every quit key
      returns `tea.Quit` so no dropped-quit timeout (slice-1 lesson).
- [x] 4.1 Hygiene: `gofmt -l .` clean, `go vet ./...` clean,
      `golangci-lint run ./...` 0 issues (one staticcheck QF1001 fixed by
      simplifying the bottom-clamp condition), `go build ./...` clean,
      `go test ./... -count=1` green (all 7 packages), goldens `-update`
      then clean rerun green.

### Verification (foreground)

- `go build ./...`: clean.
- `go test ./... -count=1`: ok, all 7 packages.
- Golden `-update` + clean rerun: `go test ./internal/tui -run
  'TestEvidenceViewGolden|TestPlanViewGolden|TestDiffViewGolden|
  TestResultViewGolden|TestConfirmViewGolden' -count=1 -update` → ok,
  new goldens inspected (evidence rows + exit-4 wording, plan confirm
  wording + quit-writes footer, diff selector + viewport + exit-2 footer,
  result written summary); clean rerun without `-update` → PASS (5/5).
- `gofmt -l .`: clean (empty).
- `go vet ./...`: clean.
- `golangci-lint run ./...`: 0 issues.
- Focused: `go test ./cmd/stackup -run TestGateMatrixBytesIdentical
  -count=1` → PASS (8 subtests); `go test ./cmd/stackup -run
  TestLaunchFailureFallbackBytesIdentical -count=1` → PASS (6 subtests);
  `go test ./internal/tui -run TestPresenceInteractiveQuitAborts
  -count=1` → PASS (4/4).
- Runtime harness: `go run ./cmd/stackup detect --path testdata/node`
  → `node: high (signals: package.json, package-lock.json) pm=npm
  version=>=20`, exit 0; `go run ./cmd/stackup detect --path
  testdata/node --format json` → parseable JSON stacks[0].ecosystem node,
  exit 0 (`testdata/small` absent, `testdata/node` used).

### Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./cmd/stackup -run TestGateMatrixBytesIdentical -count=1` → PASS (8/8 subtests); `go test ./cmd/stackup -run TestLaunchFailureFallbackBytesIdentical -count=1` → PASS (6/6); `go test ./internal/tui -run TestPresenceInteractiveQuitAborts -count=1` → PASS (4/4) |
| Runtime harness command/scenario and exact result | `go run ./cmd/stackup detect --path testdata/node` → text row `node: high ...`, exit 0; `go run ./cmd/stackup detect --path testdata/node --format json` → parseable JSON, exit 0 (`testdata/small` does not exist, `testdata/node` used) |
| Rollback boundary | Revert test/golden additions only: `cmd/stackup/cli_test.go` matrix legs, `internal/tui/tui_test.go` goldens + Update legs, `internal/tui/teatest_test.go` quit→Abort legs, `internal/tui/testdata/Test*ViewGolden.golden` (4 new); no `cmd/` branch or `internal/tui/` core change |

### Deviations from design

- Task-row harness `testdata/small` does not exist; `testdata/node`
  via `nodeProject` used for every matrix/fallback/harness leg (same as
  slice 2).
- Golden harness applies `WindowSizeMsg{80,30}` before `View()` to pin
  the diff viewport (80x22); design says fixed 80x30 ASCII — satisfied,
  other screens ignore size by design.
- Write-path fallbacks (apply confirm safe-abort, fresh-apply result
  fallback) assert behavior (exit/warning/writes) without strict
  cross-run byte equality because backup timestamps vary per run;
  read-only legs carry the strict byte-identical proof.
- None — implementation matches design otherwise.

### Issues found

- `golangci-lint` staticcheck QF1001 on the bottom-clamp condition
  (`&& !(result && ...)`); fixed by simplifying to `bottom.Cursor() !=
  sc.top` (result top is 0, cursor stays 0, extra clause dead).
- None blocking; no design gaps.

## Workload / PR boundary (this slice)

- Mode: chained PR slice (stacked-to-main), PR 3 = Unit 3 (FINAL).
- Boundary: test/golden additions only. No `cmd/` branch, gate, or
  `internal/tui/` core change. Starts after slice-2 commit, ends with all
  tasks 1.1–4.1 `[x]` and full suite green.
- Authored size: `git diff --stat HEAD` on the unit paths gives 487
  insertions / 5 deletions across 4 tracked paths
  (`cmd/stackup/cli_test.go` +268, `internal/tui/tui_test.go` +168,
  `internal/tui/teatest_test.go` +46, `tasks.md` 10 changed) = 492
  changed lines; 4 new goldens add 41 lines excluded from the authored
  count per the work-unit skill. Over the 400-line budget, so this slice
  goes with `size:exception`. The unit is already the smallest cohesive
  slice (matrix without goldens leaves TTY-shows unproven; goldens
  without matrix leave piped/json/yes/fallback unproven), so no further
  split — never code-golf.
