# Design: stackup-cli — Stack Detection + IDE Config Generation

## Technical Approach

Cobra-core services in UI-agnostic `internal/` (`detect → generate → merge → diff → apply`); Bubble Tea TUI as thin `Launcher` over the same plan. Templates render per-IDE files; `tailscale/hujson` AST patches JSONC in place (never `encoding/json` on user files). Covers all proposal targets and spec scenarios; resolves both validations below.

## Architecture Decisions

| Decision | Options (tradeoff) | Choice + rationale |
|---|---|---|
| TUI library lock | Bubble Tea (testable `Update`, teatest, active Charm stack) vs tview (fast forms, untestable tcell coupling, single-maintainer) | **Bubble Tea `v1.3.4` + `bubbles v0.20.0` + `lipgloss v1.1.0` + `huh/v2 v2.0.0`**; `x/exp` teatest pinned at `go mod init` pseudo-version, interactive-only. Testability (go-testing lens) outweighs form speed; `Launcher` interface keeps tview replaceable. |
| JSONC engine | hujson AST round-trip (comment-preserving) vs `encoding/json` (destroys comments, fails on JSONC) vs `tidwall/jsonc` (one-way, no preservation) | **`tailscale/hujson`**: `Parse → Patch → Pack`; `Standardize`+unmarshal for diff-only bridge, never marshal-back over originals. `Format()` banned from write path (opinionated drift). |
| Merge semantics | union-no-delete vs overwrite vs three-way | **Deep-merge objects, union arrays (dedupe, never delete), preserve unknown keys/comments/trailing commas**; overwrite only with `--force` + timestamped `.bak`. Prevents silent user-data loss (spec: comment-preserving merge, backup-precedes-overwrite). |
| Cursor `.vscode/*` verdict | live-compat vs import-only | **Import-only.** Evidence shows one-click migration only, no live-compat doc. Stackup writes shared `.vscode/*` AND native `.cursor/*`; docs state no live-sync assumption. |
| Devin project-local MCP verdict | project-local works for Cascade vs CLI-only | **CLI-only.** Legacy Cascade reads global `~/.codeium/windsurf/mcp_config.json` only (undocumented project-local). Stackup writes project `.devin/mcp_config.json` (CLI scope) + `.windsurf/rules/` fallback, documents Cascade-global note. |

## Data Flow

```
root ──→ detect.Registry ──→ []Evidence (ranked) ──→ generate.Templates ──→ Plan{FileOps}
                                                                    │           │
                                              merge.hujson ──→ patched bytes ──→ diff.Unified ──→ apply.Backup+Write
```

## File Changes

| File | Action | Description |
|---|---|---|
| `go.mod` (+`go.sum`) | Create | Module `stackup`, Go 1.27.1, pins above + `cobra v1.8.1`, `hujson` |
| `cmd/stackup/main.go` | Create | Cobra root, TTY gate, exit-code mapping |
| `internal/detect/*` | Create | `Detector` iface, registry, 4 detectors, confidence sort |
| `internal/generate/*` | Create | `embed` templates per IDE, `Plan` builder, idempotent render |
| `internal/merge/*` | Create | hujson AST `EditOp` applier (SetKey/UnionArray/Noop) |
| `internal/diff/*`, `internal/apply/*` | Create | Unified diff; backup (`<f>.bak.<ts>`) + gated write |
| `internal/tui/*` | Create | Bubble Tea `Model` behind `Launcher` iface; stub for tests |

## Interfaces / Contracts

```go
type Confidence int // Low < Medium < High
type Evidence struct {
  Ecosystem string; Confidence Confidence; Signals []string
  VersionHint, PackageManager string
}
type Detector interface { Name() string; Detect(root string) []Evidence }
type EditOp struct { Path string; Kind string } // SetKey | UnionArray | Noop
type Launcher interface { Run(p Plan) (Action, error) } // Action: Apply|Abort
```

Confidence: manifest alone = Medium; +lockfile/toolchain pin = High; version-manager hint alone = Low. Sort High→Low, registry order breaks ties. Empty + no `--allow-unknown` = exit 4, generation stops.
Cobra: `detect|generate|diff|apply [--path --ide vscode,cursor,devin,kiro --format text|json --dry-run --force --yes --non-interactive --allow-unknown]`. Exits: `0` ok/no-change, `2` preview-has-changes, `3` blocked-non-interactive, `4` unknown-stack, `1` error.

## Sequence Diagrams

```
detect→generate→diff→apply:
User → Cobra → detect.Registry → []Evidence → generate.Plan → merge.AST
  → diff.Unified (stdout; exit 2 if changes, write nothing)
  → [apply] TUI.Launcher? → apply.Backup → Write (exit 0)
CI (non-TTY, no --yes): skip Launcher → pending writes? exit 3, write nothing
```

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | Detectors × fixtures; merge EditOps; confidence sort | Table-driven, `t.TempDir()`, `t.Run` |
| TUI state | Screen transitions | Direct `Model.Update(tea.Msg)` asserts |
| Interactive | Select→preview→confirm | `teatest` only, pinned `x/exp` |
| Golden | Rendered configs + diffs; idempotence re-render | `testdata/*.golden`, `-update` then re-run clean |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Pure file read/render/write; non-TTY gate is a UI skip, not command composition. No RED tests carried.

## Migration / Rollout

No migration (greenfield). Rollout: `go mod init` + pins → `internal/` core → Cobra → TUI; `--dry-run` writes nothing; `apply` without backup forbidden.

## Open Questions

- [ ] Exact `x/exp` teatest pseudo-version at `go mod init` (pin then record in `go.sum`)?
- [ ] Kiro `.kiro/specs/<feature>` scaffold depth (stub files vs dirs-only)?
