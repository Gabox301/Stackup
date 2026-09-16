```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:96541268b512deffe0f003b0c07b3259a3ac3139306bb980cb9b1c8118547557
verdict: pass
blockers: 0
critical_findings: 0
requirements: 5/5
scenarios: 12/12
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:07cf69f9a5a40bc0597d9cf3fb8f4322f4b926b5617cebbd29ab9d12f43893e6
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: stackup-tui-always
**Version**: N/A
**Mode**: Standard (strict_tdd: false per openspec/config.yaml; no workspace runner)

### Completeness
| Metric | Value |
|--------|-------|
| Tasks total | 15 |
| Tasks complete | 15 |
| Tasks incomplete | 0 |

All tasks 1.1-1.5, 2.1-2.5, 3.1-3.4, 4.1 are `[x]` in `openspec/changes/stackup-tui-always/tasks.md` (5 + 5 + 4 + 1). No pending task blocks verification.

### Build & Tests Execution
**Build**: ✅ Passed (exit 0, empty output)
```text
$ go build ./...
EXIT:0
(empty output, sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)
```

**Tests**: ✅ Passed — full suite green THREE times (rules out reported transient teatest flake)
```text
$ go test ./... -count=1   # run 1
ok  stackup/cmd/stackup  2.554s
ok  stackup/internal/apply  0.367s
ok  stackup/internal/detect  0.339s
ok  stackup/internal/diff  0.323s
ok  stackup/internal/generate  0.261s
ok  stackup/internal/merge  0.104s
ok  stackup/internal/tui  0.588s
EXIT:0

$ go test ./... -count=1   # run 2
ok  stackup/cmd/stackup  2.943s
ok  stackup/internal/apply  0.437s
ok  stackup/internal/detect  0.411s
ok  stackup/internal/diff  0.401s
ok  stackup/internal/generate  0.098s
ok  stackup/internal/merge  0.271s
ok  stackup/internal/tui  0.673s
EXIT:0

$ go test ./... -count=1   # run 3 (hashed evidence)
ok  stackup/cmd/stackup  3.073s
ok  stackup/internal/apply  0.566s
ok  stackup/internal/detect  0.484s
ok  stackup/internal/diff  0.520s
ok  stackup/internal/generate  0.430s
ok  stackup/internal/merge  0.332s
ok  stackup/internal/tui  0.858s
EXIT:0 (sha256:07cf69f9a5a40bc0597d9cf3fb8f4322f4b926b5617cebbd29ab9d12f43893e6)
```

**Focused gates and fallbacks**: ✅ All passed
```text
$ go test ./cmd/stackup -run TestShouldUseTUI -count=1
PASS (8/8: tty_text_presents, piped_stdout_stays_text, piped_stdin_stays_text, both_piped, yes_flag, non-interactive, json_never, json_piped)

$ go test ./cmd/stackup -run TestGateMatrixBytesIdentical -count=1
PASS (8/8: text detect/diff/generate-dry-run/apply-dry-run + json detect/diff/generate-dry-run + text apply-blocked; stdout+stderr+exit byte-identical, stub Calls==0)

$ go test ./cmd/stackup -run TestLaunchFailureFallbackBytesIdentical -count=1
PASS (6/6: detect/diff/generate-dry-run/apply-dry-run strict stdout-identical + warning + engine exit + zero writes; apply-confirm safe-abort exit 0; fresh-apply result fallback warns + prints + writes)

$ go test ./internal/tui -run 'TestEvidenceViewGolden|TestPlanViewGolden|TestDiffViewGolden|TestResultViewGolden|TestConfirmViewGolden' -count=1
PASS (5/5, deterministic rerun WITHOUT -update)

$ go test ./internal/tui -run 'TestPresenceInteractiveQuitAborts|TestConfirmInteractiveApplies|TestPresenceScreensUpdatePerScreen|TestDiffScreenUpdateKeysAndViewport' -count=1
PASS (confirm y->Apply kept; 4/4 presence quit->Abort; per-screen clamps; diff selector/viewport)

$ go test ./cmd/stackup -run 'TestDiffPreviewExit2WritesNothing|TestApplyConfirm|TestApplyBlocked|TestApplyTUISwapKeepsCoreGreen|TestGenerateDryRun|TestDetect' -count=1
PASS (prior-suite legs incl. TestApplyBlockedWithoutPromptInCI, TestApplyConfirmDialogOnTTY, TestApplyTUISwapKeepsCoreGreen, TestDiffPreviewExit2WritesNothing, TestDetectJSONIsParseable, TestDetectUnknownStackExit4)
```

**Hygiene**: ✅ All clean
```text
$ gofmt -l .        -> empty (exit 0)
$ go vet ./...      -> clean (exit 0)
$ golangci-lint run ./... -> 0 issues (exit 0)
```

**Runtime harness (JSON parseability + text parity)**:
```text
$ go run ./cmd/stackup detect --path testdata/node
node: high (signals: package.json, package-lock.json) pm=npm version=>=20 (exit 0)
$ go run ./cmd/stackup detect --path testdata/node --format json
parseable JSON, stacks[0].ecosystem == node (exit 0, verified via ConvertFrom-Json)
```
Note: task-row path `testdata/small` does not exist in repo; `testdata/node` used (same substitution as apply-progress slices 2-3).

**Coverage**: ➖ Not available / threshold: 0 per openspec/config.yaml verify.coverage_threshold → not enforced, no gate.

### Spec Compliance Matrix
| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Delta: TUI presence per command | TTY-shows | `cmd/stackup/branches_test.go > TestDetectPresentsEvidenceScreen + TestDiffPresentsDiffScreen + TestGenerateDryRunPresentsPlanScreen + TestApplyFreshPresentsResultScreen` + `internal/tui/tui_test.go > TestEvidenceViewGolden/TestPlanViewGolden/TestDiffViewGolden/TestResultViewGolden` + `internal/tui/teatest_test.go > TestPresenceInteractiveQuitAborts` | ✅ COMPLIANT |
| Delta: TUI presence per command | apply-confirm-unchanged | `cmd/stackup/cli_test.go > TestApplyConfirmDialogOnTTY` (decline abort / confirm write) + `internal/tui/teatest_test.go > TestConfirmInteractiveApplies` (y->Apply) | ✅ COMPLIANT |
| Delta: TUI gate rule | piped-stays-text | `cmd/stackup/gate_test.go > TestShouldUseTUI/piped_stdout_stays_text` + `cmd/stackup/cli_test.go > TestGateMatrixBytesIdentical/text/*` (byte-identical stdout+stderr+exit) | ✅ COMPLIANT |
| Delta: TUI gate rule | json-never | `cmd/stackup/gate_test.go > TestShouldUseTUI/json_never_presents` + `cmd/stackup/cli_test.go > TestGateMatrixBytesIdentical/json/*` + `TestDetectJSONIsParseable` | ✅ COMPLIANT |
| Delta: TUI gate rule | yes-flag-stays-text | `cmd/stackup/gate_test.go > TestShouldUseTUI/yes_flag_stays_text + non-interactive_stays_text` + `cmd/stackup/cli_test.go > TestGateMatrixBytesIdentical` | ✅ COMPLIANT |
| Delta: Fallback and write gating | launch-failure-fallback | `cmd/stackup/cli_test.go > TestLaunchFailureFallbackBytesIdentical` (6/6) + `cmd/stackup/branches_test.go > TestDiffLaunchFailureFallsBack` | ✅ COMPLIANT |
| Canonical: Core commands with preview and machine output | Dry run writes nothing | `cmd/stackup/cli_test.go > TestDiffPreviewExit2WritesNothing` + `TestGenerateDryRunJSONParseableWritesNothing` | ✅ COMPLIANT |
| Canonical: Core commands with preview and machine output | JSON output for CI | `cmd/stackup/cli_test.go > TestDetectJSONIsParseable` + `TestGenerateDryRunJSONParseableWritesNothing` + live `detect --format json` parse check | ✅ COMPLIANT |
| Canonical: Non-interactive safety with replaceable TUI | CI run skips prompts | `cmd/stackup/cli_test.go > TestApplyBlockedWithoutPromptInCI` (exit 3, zero writes, --yes writes with backup) | ✅ COMPLIANT |
| Canonical: Non-interactive safety with replaceable TUI | TUI swap keeps core green | `cmd/stackup/cli_test.go > TestApplyTUISwapKeepsCoreGreen` (confirm+result 2 calls, abort 1 call) + `internal/tui/tui_test.go > TestStubSwapKeepsCoreGreen` | ✅ COMPLIANT |
| Canonical: Non-interactive safety with replaceable TUI | TTY decline aborts with success status | `cmd/stackup/cli_test.go > TestApplyConfirmDialogOnTTY` (decline exit 0, zero writes) — wording `apply aborted (no writes performed)` verified in `cmd/stackup/apply.go:88,94` | ✅ COMPLIANT |
| Canonical: Non-interactive safety with replaceable TUI | Fresh files apply without confirmation | `cmd/stackup/branches_test.go > TestApplyFreshPresentsResultScreen` (1 result screen, writes, empty stdout) | ✅ COMPLIANT |

**Compliance summary**: 12/12 scenarios compliant (delta 6/6 + canonical 6/6). Requirements 5/5.

### Correctness (Static Evidence)
| Requirement | Status | Notes |
|------------|--------|-------|
| TUI presence per command | ✅ Implemented | `detect.go:36-44` Evidence, `generate.go:40-52,60-78` Plan (dry-run + write-then-write), `diff.go:37-48` Diff, `apply.go:41-52,78-98,107-116` Diff-preview/Confirm/Result; TUI replaces print, quit maps to engine exits 0/2/4 |
| TUI gate rule | ✅ Implemented | `root.go:75-77` is `stdoutIsTTY() && stdinIsTTY() && !yes && !nonInteractive && format==text`; both probes are real `os.Stdout/Stdin.Stat` ModeCharDevice checks (`root.go:44-62`), stub-vars only for tests — piped-stdout exclusion is real, not stubbed |
| Fallback and write gating | ✅ Implemented | Every TUI leg catches `Run` error → `warnTUIFallback` (`warning: interactive display unavailable, showing plain output: …`) + byte-identical fallback print; engine exits preserved (detect 0/4, diff/generate 2/0, apply blocked 3, confirm-failure safe-abort 0 never coerced to 1); presence never gates writes, only Confirm Apply writes |
| Core commands with preview and machine output | ✅ Implemented | `printPreview`/`writeJSON` paths untouched; `diff --dry-run`/generate dry-run still exit 2 with zero writes; `--format json` short-circuits before any gate (`detect.go:29-31`, apply blocked JSON leg) |
| Non-interactive safety with replaceable TUI | ✅ Implemented | `apply.go:62` blocks on `!shouldUseTUI` (covers piped-stdout, piped-stdin, non-interactive, json) exit 3 byte-identical; confirm/abort wording `apply aborted (no writes performed)` unchanged at both abort sites; `newTUILauncher` seam keeps TUI replaceable |

### Coherence (Design)
| Decision | Followed? | Notes |
|----------|-----------|-------|
| Shared Model{screen,…} with per-screen constructors | ✅ Yes | `Screen` enum + `Input` + 4 constructors per design contract; presence quit→Abort, only Confirm Apply writes |
| Launcher Run(Input) single construction site | ✅ Yes | `Run(Input) (Action, error)` in `tea.go modelForInput`; `StubLauncher.Inputs` recorded; deviations (Plans mirror, seam rename to `newTUILauncher`) are documented in apply-progress and preserve assertions |
| Gate stdout+stdin | ✅ Yes | Exact predicate implemented; json never presents |
| Diff selector + viewport with exit-2 footer | ✅ Yes | `diffview.go` selector + viewport, 80x30 goldens (diff pins 80x22 viewport via WindowSizeMsg) |
| TUI replaces print, fallback prints | ✅ Yes | All four commands follow compute-then-present; fallback is the only second print |

### Issues Found
**CRITICAL**: None
**WARNING**: None
**SUGGESTION**:
- Remove stray `stackup` binary at repo root (`?? stackup` in git status) before archive/PR so the tree stays source-only.
- Commit or explain uncommitted `go.mod`/`go.sum` tidy (`termenv v0.16.0` direct vs indirect, one `cellbuf` sum dropped): build-clean `go mod tidy` side effect, no behavior change, but archive should record the final pinned set.

### Verdict
PASS — all 15 tasks complete; 12/12 scenarios have passing covering tests across three full-suite runs plus focused gate/fallback/golden legs; hygiene (gofmt/vet/lint/build) clean; gate is a real dual-TTY check; exits and confirm/abort wording preserved.
