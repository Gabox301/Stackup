# Tasks: stackup-cli — Detection + IDE Config Generation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1800–2300 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR1→PR5 per work unit below |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Scaffold+detect | PR 1 | `go test ./internal/detect/` | `go run ./cmd/stackup detect --path testdata/node` | drop `go.mod`+`internal/detect/` |
| 2 | Generate+merge | PR 2 | `go test ./internal/generate/ ./internal/merge/` | `go run ./cmd/stackup generate --dry-run` | drop `internal/generate/`+`internal/merge/` |
| 3 | Diff+apply | PR 3 | `go test ./internal/diff/ ./internal/apply/` | `go run ./cmd/stackup diff --path testdata/e2e` | drop `internal/diff/`+`internal/apply/` |
| 4 | CLI wiring | PR 4 | `go test ./cmd/...` | `go run ./cmd/stackup --help` | drop `cmd/stackup/` |
| 5 | TUI+goldens | PR 5 | `go test ./...` | `go run ./cmd/stackup apply --dry-run` | drop `internal/tui/`+`testdata/*.golden` |

## Phase 1: Scaffold

- [x] 1.1 Init `go.mod` (`stackup`, Go 1.27.1); pin cobra/Bubble Tea/hujson; record `x/exp` version in `go.sum`.
- [x] 1.2 Create `cmd/stackup/main.go` stub + `internal/` layout; `go build ./...` passes.

## Phase 2: Detect

- [x] 2.1 Add `internal/detect/types.go` (`Evidence`, `Confidence`) + `Detector` iface + ordered registry.
- [x] 2.2 Implement Node/Go/Python/Rust detectors in `internal/detect/` + `testdata/` fixtures.
- [x] 2.3 Add table-driven `internal/detect/detect_test.go` (`t.TempDir()`); cover polyglot, allow-unknown, lockfile-high, empty-stops.

## Phase 3: Generate/Merge

- [x] 3.1 Add `embed` templates + `Plan` builder in `internal/generate/` per IDE (vscode, cursor, devin+windsurf, kiro).
- [x] 3.2 Implement hujson `EditOp` applier in `internal/merge/` (SetKey/UnionArray/Noop); never `encoding/json` on user files.
- [x] 3.3 Add `testdata/*.golden` tests: fresh render + idempotent re-render.

## Phase 4: Diff/Apply

- [x] 4.1 Implement `internal/diff/` unified diff; write nothing, exit 2 on pending changes.
- [x] 4.2 Implement `internal/apply/` timestamped `.bak.<ts>` backup + gated write; overwrite only with `--force`.
- [x] 4.3 Test comment-preserving merge, backup-before-write, dry-run writes nothing.

## Phase 5: CLI

- [x] 5.1 Wire Cobra `detect|generate|diff|apply` in `cmd/stackup/` with all flags.
- [x] 5.2 Map exit codes 0/1/2/3/4 + `--format json` + non-TTY gate in `cmd/stackup/main.go`.
- [x] 5.3 Test JSON parseable, CI no-prompt exit 3, unknown-stack exit 4.

## Phase 6: TUI

- [x] 6.1 Add `internal/tui/` `Launcher` iface + Bubble Tea `Model` over same `Plan`.
- [x] 6.2 Add stub `Launcher` in `internal/tui/`; TUI swap keeps core green.
- [x] 6.3 Add `Model.Update` tests + `teatest` flow (pinned `x/exp`, interactive-only).

## Phase 7: Tests/Goldens

- [x] 7.1 `gofmt -l .`, `go vet ./...`, `golangci-lint run` clean; fix findings.
- [x] 7.2 Full `go test ./...` green incl. goldens (`-update` then clean re-run).
