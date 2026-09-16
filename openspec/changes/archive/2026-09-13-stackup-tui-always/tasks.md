# Tasks: stackup-tui-always

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 550-700 authored (goldens excluded) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | TUI core: Screen/Input/Run + constructors + diffview | PR 1 | `go test ./internal/tui -count=1` | N/A — no CLI wiring yet; unit proof only | `internal/tui/` core revert, confirm path intact |
| 2 | Gate + 4 branches + fallback | PR 2 | `go test ./cmd/stackup -run TestShouldUseTUI -count=1` | `go run ./cmd/stackup diff --path testdata/small` | `cmd/stackup/` branches revert to print path |
| 3 | Goldens + matrix + hygiene | PR 3 | `go test ./... -count=1` | `go run ./cmd/stackup detect --path testdata/small` text vs json | `internal/tui/testdata/`, `cmd/stackup/cli_test.go` revert |

## Phase 1: TUI core

- [x] 1.1 Extend `internal/tui/tui.go` with Screen enum, Input struct, Launcher Run(Input) (TTY-shows)
- [x] 1.2 Extend `internal/tui/model.go` with payload fields, 4 constructors, View/Update switch (TTY-shows)
- [x] 1.3 Create `internal/tui/diffview.go` selector + viewport helpers, exit-2 footer (TTY-shows diff)
- [x] 1.4 Switch `internal/tui/tea.go` Run to build Model from Input (apply-confirm-unchanged)
- [x] 1.5 Record Inputs in `internal/tui/stub.go`, keep canned Action/Err (launch-failure-fallback)

## Phase 2: Gate and command branches

- [x] 2.1 Add stdoutIsTTY stub-var + shouldUseTUI to `cmd/stackup/root.go` (piped/json/yes)
- [x] 2.2 Branch `cmd/stackup/detect.go` compute-then-present Evidence, exits 0/4 (TTY-shows)
- [x] 2.3 Branch `cmd/stackup/generate.go` compute-then-present Plan, quit-then-write (TTY-shows)
- [x] 2.4 Branch `cmd/stackup/diff.go` compute-then-present Diff or fallback (piped-stays-text)
- [x] 2.5 Migrate `cmd/stackup/apply.go` to Run(Confirm)+Run(Result) with fallback warning (launch-failure-fallback)

## Phase 3: Tests, goldens, matrix

- [x] 3.1 Cover gate matrix + launch-failure bytes/exit in `cmd/stackup/cli_test.go` (piped/json/yes/fallback)
- [x] 3.2 Add per-screen 80x30 ASCII goldens in `internal/tui/testdata/` (TTY-shows)
- [x] 3.3 Cover per-screen Update keys/clamps in `internal/tui/tui_test.go` (TTY-shows)
- [x] 3.4 Cover quit→Abort per screen in `internal/tui/teatest_test.go`, keep confirm y/n (apply-confirm-unchanged)

## Phase 4: Hygiene

- [x] 4.1 Run gofmt/vet/lint, refresh goldens with -update then clean `go test ./... -count=1` green (all scenarios)
