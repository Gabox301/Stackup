# Apply Progress: stackup-cli — Slices 1+2+3+4+5 (Detect + Generate/Merge + Diff/Apply + CLI + TUI, PR5)

- Change: `stackup-cli`
- Slice: 5 of 5 (chained delivery, stacked-to-main) — FULL SCOPE IMPLEMENTED
- Scope: tasks 6.1, 6.2, 6.3, 7.1, 7.2 (Phase 6 TUI + Phase 7 tests/goldens)
- Status: complete — ready for `sdd-verify`

## Completed tasks

### Slice 5 (PR5, this batch, commit `67c74a5`)

- [x] 6.1 — `internal/tui/` (`tui.go` `Launcher` iface + `Action{Abort,Apply}`,
  `model.go` Bubble Tea confirm `Model` over the same `generate.Plan`,
  `tea.go` real launcher via `tea.NewProgram` with injected stdio).
  Keys: `y` apply, `n/q/esc` abort, `up/k`+`down/j` move, `enter` apply;
  undecided shutdown (EOF) reads as Abort. View renders plain ASCII
  (explicit `termenv.Ascii` renderer) so piped output and goldens are
  deterministic. TTY-only: `cmd/stackup/apply.go` calls the launcher
  through the `newConfirmLauncher` seam; non-TTY/CI exit-3 path untouched.
- [x] 6.2 — `stub.go` `StubLauncher` (canned Action/Err, records Plans);
  swap proven at both layers: `TestStubSwapKeepsCoreGreen` (tui: full
  16-file plan → stub Apply → `apply.Apply` writes 16, `diff` clean) and
  `TestApplyTUISwapKeepsCoreGreen` (cmd: stub Apply writes w/ backup,
  stub Abort exits 0 with zero writes).
- [x] 6.3 — Direct `Model.Update` tests (decide keys, nav clamps,
  WindowSizeMsg/unbound-key ignored, undecided→Abort) + ONE teatest flow
  (`TestConfirmInteractiveApplies`: pinned `x/exp` teatest
  `v0.0.0-20250227204225-5cbdec3c4e09`, waits for render, sends `y`,
  expects confirmed Apply) + View golden
  (`testdata/TestConfirmViewGolden.golden` via `x/exp/golden`, `-update`
  then clean re-run).
- [x] 7.1 — `gofmt -l .` clean, `go vet ./...` clean,
  `golangci-lint run ./...` **0 issues**. Fixed the 7 baseline errcheck
  findings by propagating `fmt` write errors in `cmd/` (no disables).
- [x] 7.2 — Full `go test ./... -count=1` green (7 packages); goldens
  refreshed with `-update` on golden-owning packages (generate, merge,
  tui) with zero churn to existing goldens, then clean re-run green.
  Open questions resolved: (1) `x/exp` teatest version is
  `v0.0.0-20250227204225-5cbdec3c4e09` (in `go.sum`, exercised by the
  teatest flow); (2) kiro scaffold = **stub files** (6 rendered targets:
  steering product/tech/structure + specs requirements/design/tasks,
  proven by `TestBuildAllIDEsCoversSpecTargets`), not dirs-only.

### Slice 4 (PR4, committed `2f1dc2b`)

- [x] 5.1 — Cobra `detect|generate|diff|apply` in `cmd/stackup/`
  (`main.go` entry + `root.go` shared infra + one file per command).
  Persistent flags: `--path --format --yes --non-interactive
  --allow-unknown`; per-command `--ide` (generate/diff/apply),
  `--dry-run` (generate/apply), `--force` (generate/diff/apply).
  `detect --path --allow-unknown` text output keeps the slice-1 line
  format (compatible).
- [x] 5.2 — Exit mapping 0 ok/no-change, 2 preview-has-changes,
  3 blocked-non-interactive, 4 unknown-stack, 1 error via `exitError`
  (quiet statuses; only exit-1 errors print `error:`).
  `--format json` emits indented parseable JSON on stdout for every
  command (detect payload, preview payload, write payload, blocked
  payload, unknown-stack `{"stacks":[]}`). Non-TTY auto-skip: apply
  with pending overwrites and no `--yes` exits 3 writing nothing under
  `--non-interactive` or a non-terminal stdin; on a TTY it prompts once
  (`[y/N]`, EOF/anything-but-yes declines, never blocks).
- [x] 5.3 — Tests in `cmd/stackup/cli_test.go` (7 tests): `--help`
  lists all commands, detect/generate-dry-run JSON parseable,
  unknown-stack exit 4 (text + parseable JSON, `--allow-unknown`
  proceeds), CI no-prompt apply exit 3 with zero writes vs `--yes`
  exit 0 with one backup, TTY prompt decline-abort vs confirm-write,
  diff exit 2 with zero writes then exit 0 when clean.

### Slice 1 (PR1, committed `4d034ec`)

- [x] 1.1 — `go.mod` (`stackup`, Go 1.27.1); pins: cobra v1.8.1, bubbletea v1.3.4,
  bubbles v0.20.0, lipgloss v1.1.0, huh v2.0.0 (via `charm.land/huh/v2`),
  hujson v0.0.0-20260727124030-b80ff77dac4f, teatest
  v0.0.0-20250227204225-5cbdec3c4e09; `go mod tidy` clean.
- [x] 1.2 — `cmd/stackup/main.go` stub (Cobra root + minimal `detect`
  subcommand as slice harness) + `internal/` layout; `go build ./...` passes.
- [x] 2.1 — `internal/detect/types.go` (`Evidence`, `Confidence`) + `Detector`
  iface + ordered registry (`registry.go`: `NewRegistry`, `DefaultRegistry`,
  `DetectAll`, `Detect`, `ErrUnknownStack`).
- [x] 2.2 — Node/Go/Python/Rust detectors + `internal/detect/testdata/`
  fixtures (node, go, python, rust, polyglot) + root `testdata/node` CLI demo.
- [x] 2.3 — Table-driven `internal/detect/detect_test.go` on `t.TempDir()`
  (fixtures copied via `os.CopyFS`): polyglot ranking, allow-unknown,
  lockfile-high, empty-stops, plus registry tie-break and version hints.

### Slice 2 (PR2)

- [x] 3.1 — `internal/generate/generate.go`: `embed` templates
  (`templates/{vscode,cursor,devin,windsurf,kiro}/*.tmpl`) + `Plan` builder
  (`Build`, `SupportedIDEs`, `FileOp{Path,Content,JSON}`). Full plan is 16
  files: vscode 4 JSON targets, cursor `.mdc` + `mcp.json`, devin
  rules + `mcp_config.json` (project scope) + `.windsurf` fallback,
  kiro steering (product/tech/structure) + specs scaffold
  (requirements/design/tasks) + `settings/mcp.json`. Empty evidence renders
  generic content for `--allow-unknown` flows; unknown IDE names error.
- [x] 3.2 — `internal/merge/merge.go`: hujson AST `EditOp` applier
  (`SetKey`/`UnionArray`/`Noop` over RFC 6901 pointers) + `Merge`
  (deep-merge objects, union arrays without deleting user entries, existing
  scalars win). Imports are `bytes/fmt/strings/hujson` only — no
  `encoding/json` on user files; output is `Value.Pack`, `Format()` never
  called on the write path.
- [x] 3.3 — Golden tests: 7 generate goldens
  (`internal/generate/testdata/*.golden`: 4 vscode JSON + cursor rules +
  devin rules + kiro tech, node evidence) proving fresh render; re-render
  byte-equality asserted in code. 2 merge goldens
  (`internal/merge/testdata/merge-{comment,union}.golden`) proving
  comment-preserving merge. Created via `-update`, then clean re-run.

### Slice 3 (PR3, this batch)

- [x] 4.1 — `internal/diff/diff.go`: `Compute(root, plan, opts)` previews a
  `generate.Plan` against disk as `Result{Files []FileDiff}` with
  `HasChanges()` as the typed pending-changes signal (CLI exit-2 mapping
  stays in slice 4; no Cobra wired here). Resolution: existing JSON targets
  via `merge.Merge`, fresh files verbatim, non-JSON verbatim; `Force`
  previews the overwrite path (`merge.Apply` + one `SetKey` per generated
  top-level key). Single-hunk unified diffs (`--- a/`, `+++ b/`, `@@`)
  with 3 context lines. Filesystem reads only — writes nothing.
- [x] 4.2 — `internal/apply/apply.go`: `Apply(root, plan, opts)` resolves
  via `diff.Compute`, then backs up every existing target to a timestamped
  sibling (`<f>.bak.YYYYMMDD-HHMMSS`, counter-deduped, no colons for
  Windows) BEFORE writing; fresh files need no backup. No exported write
  path skips the backup — apply without backup is structurally forbidden;
  a backup failure aborts before the write. `Force` selects the
  `Apply+SetKey` overwrite path (default stays `Merge`); `DryRun` reports
  the would-write set with zero disk writes.
- [x] 4.3 — Tests: 6 diff tests (fresh-all-new, clean-tree, pending-change,
  merge-keeps-scalars vs force-overwrites, dry-run-writes-nothing via
  tree snapshot, invalid-JSONC errors) + 7 apply tests (fresh create,
  backup-before-write with pre-image check, never-writes-without-backup,
  dry-run-writes-nothing, comment-preserving end-to-end with
  apply→apply-idempotent→diff-clean, force-overwrites-scalars, fail-safe
  on invalid JSONC with file untouched and no stray backup).

## Work Unit Evidence

### Slice 5 (PR5, this batch)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/tui/ ./cmd/... -count=1` — PASS (tui 5/5 incl. teatest flow + golden; cmd 8/8 incl. stub-swap and dialog tests) |
| Runtime harness command/scenario and exact result | Built binary `apply --dry-run --path .` — exit 2 with unified preview of the fresh plan, zero writes; empty dir — exit 4 (unknown-stack). (`go run` wrapper reports its own exit 1 per slice-4 note; exact codes verified on the built binary.) |
| Rollback boundary | Drop `internal/tui/` + revert `cmd/stackup/{apply,cli_test,detect,root}.go` + `go.mod`/`go.sum` cellbuf line — engines (`internal/detect|generate|merge|diff|apply`) untouched |

### Slice 4 (PR4, committed `2f1dc2b`)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./cmd/... -count=1` — PASS, 7/7 |
| Runtime harness command/scenario and exact result | `go run ./cmd/stackup --help` — exit 0, lists detect/generate/diff/apply; `detect --path testdata/node` — exit 0, `node: high (signals: package.json, package-lock.json) pm=npm version=>=20`; built binary `diff --path testdata/e2e` — exit 2 with unified preview of the 16-file fresh plan; `generate --dry-run` at repo root — exit 2 previewing the go plan (16 files); binary `apply --non-interactive` on seeded-overwrite project — exit 3, zero writes; `--format json` blocked body parseable (`pending`, `blocked`) |
| Rollback boundary | Drop `cmd/stackup/{root,detect,generate,diff,apply}.go` + `cli_test.go` and restore the slice-1 `main.go` stub; drop `testdata/e2e/` — engines (`internal/`, `go.mod`) untouched |

### Slice 3 (PR3)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/diff/ ./internal/apply/ -count=1` — PASS, 13/13 (diff 6/6 + apply 7/7) |
| Runtime harness command/scenario and exact result | SUPERSEDED by slice 4: canonical `go run ./cmd/stackup diff --path testdata/e2e` now runs for real — exit 2 with the 16-file fresh-plan preview (was NOT AVAILABLE in slice 3: no `diff` subcommand and no `testdata/e2e`). Slice-3 nearest equivalent (retired): temp in-module harness ran detect→generate→diff→apply(dry-run) over a copy of `testdata/node`: evidence=1, plan_files=16, pending_changes=true, would_write=16, zero writes |
| Rollback boundary | Drop `internal/diff/` + `internal/apply/` — no other code touched (`internal/detect/`, `internal/generate/`, `internal/merge/`, `go.mod` unchanged) |

## Additional verification (slice 5, final)

- `go build ./...`: clean
- `go test ./... -count=1`: all green (7 packages, no regressions)
- `gofmt -l .`: clean
- `go vet ./...`: clean
- `golangci-lint run ./...`: 0 issues (7 baseline errcheck findings fixed by error propagation, no disables)
- Goldens: `-update` on generate/merge/tui green with zero churn to existing goldens; clean re-run green
- Slice-5 constraints: engines untouched; non-TTY/CI apply path byte-identical behavior (exit 3, zero writes); harness runs wrote nothing to the repo

- `go build ./...`: clean
- `go test ./... -count=1`: all green (cmd 7/7; detect, generate, merge, diff, apply still pass, no regressions)
- `gofmt -l .`: clean
- `go vet ./...`: clean
- Slice-4 constraints: `internal/` untouched; no new external deps; harness runs wrote nothing (`testdata/e2e` holds only `package.json`, no configs leaked to repo root)
- TTY-prompt-decline path also exercised at binary level (prompt printed, EOF declined, exit 0, zero writes)

## Deviations from design

None — implementation matches design.

### Slice 5 notes

1. The slice-4 `y/N` line prompt is gone by design (replaced by the TUI
   dialog this slice wires in). `TestApplyPromptConfirmsOnTTY` became
   `TestApplyConfirmDialogOnTTY` with identical assertions driven through
   the launcher seam; the real keystroke path is covered by the teatest
   flow. A headless `tea.NewProgram.Run()` with piped stdio hangs on
   Windows, so cmd tests never run the real program — the seam exists
   precisely for that.
2. `x/cellbuf` bumped Mar-2025 pseudo-version → `v0.0.15`: the first real
   compile of the TUI stack exposed that `huh/v2` forces `ansi v0.11.6`
   (bool-arg style API) while lipgloss's pinned cellbuf expected the old
   API. Direct pins (Bubble Tea v1.3.4, bubbles, lipgloss, huh, teatest)
   unchanged; only the incompatible indirect pin moved.
3. `bubbles`/`huh` stay pinned-but-unused: a single-key confirm needs
   neither a forms engine nor widgets. The pins stand for the future
   interactive flows the design anticipates; forcing them into this
   dialog would add code without behavior.
4. TUI View golden uses `x/exp/golden` (`testdata/TestConfirmViewGolden.golden`)
   instead of a local helper: teatest already registers golden's `-update`
   flag, so a second `update` flag panics the test binary. One `-update`
   now refreshes every golden in the repo uniformly.

### Slice 4 notes

1. `generate` (real write) goes through `apply.Apply`, so it inherits
   mandatory backup-before-write; the confirmation gate lives only in
   `apply`, giving `generate` (protected writer) vs `apply` (gated
   writer) roles.
2. The gate blocks only on overwrites (`Changed && !IsNew`); fresh-file
   applies proceed without confirmation, per the spec scenario wording
   ("pending overwrites").
3. TTY-prompt decline aborts with exit 0 and "apply aborted (no writes
   performed)": an explicit user abort is neither ok-with-writes nor an
   error nor a non-interactive block.

### Slice 3 notes (retained)

1. `EditOp.Path` uses RFC 6901 JSON pointers (matching hujson's `Find`
   vocabulary); `""` addresses the document root.
2. `Merge` keeps existing scalars (union-no-delete); explicit overwrite is
   only via `Apply` + `OpSetKey` (the `--force` path, now wired in
   slice 4). Backup-before-write stays in slice 3 (`internal/apply`).
3. Devin/Kiro/Cursor MCP scaffolds all use `{"mcpServers": {}}`; Devin scope
   (project-local CLI) and the Cascade-global note live in the markdown
   rules docs, per the design verdicts.
4. `generate` imports `internal/detect` for `Evidence` (data flow
   `detect → generate`); `merge` depends only on hujson. No cycles.
5. `diff` imports `generate` (Plan) + `merge` (resolution); `apply`
   imports `diff` (single resolution path, no duplicated merge logic).
   Chain `generate → merge → diff → apply` matches the design data flow;
   no cycles. Exit codes 2/3/4 are mapped in slice 4 (`cmd/stackup/`)
   from the `HasChanges()` typed signal — `internal/` carries no exit
   codes.
6. Backup stamps use `YYYYMMDD-HHMMSS` UTC (no colons: Windows-safe);
   same-second reruns dedupe with a `-N` counter suffix.

## Remaining (out of scope for this slice)

None — full scope (Phases 1–7) implemented across PR1–PR5. Next: `sdd-verify`.
