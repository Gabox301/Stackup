# Proposal: stackup-cli — Stack Detection + IDE Config Generation

## Intent

Greenfield Go CLI detecting MVP-4 stacks and generating configs for `vscode`, `cursor`, `devin`, `kiro`: merge-with-backup default, preview before write, CI-safe.

## Scope

### In Scope
- Ordered Detector registry yielding ranked Evidence list (polyglot-safe) for Node/Go/Python/Rust
- Per-IDE config generation + hujson JSONC merge, backup, `--dry-run` diff preview
- Cobra-core `detect/generate/diff/apply` with `--format json`, `--yes/--non-interactive`, non-TTY auto-detect; TUI behind replaceable interface

### Out of Scope
- Wave 2 detectors (Java/Ruby/PHP/.NET) and plugin tail (Elixir/Dart/Swift/Zig)
- TUI library lock (Bubble Tea-leaning; deferred to design), remote import/Team rules, hooks/agents inventory
- Cascade project-local MCP and `cursor` `.vscode/*` live-compat verdicts (design validations)

## Capabilities

### New Capabilities
- `stack-detection`: registry, Evidence ranking, MVP-4 signals + confidence
- `config-generation`: per-IDE targets, template + hujson merge, union-no-delete, backup
- `cli-ux`: Cobra commands, dry-run/format/CI flags, TUI interface boundary

### Modified Capabilities
- None (greenfield; `openspec/specs/` empty)

## Approach

Cobra-core services in UI-agnostic `internal/` (`detect → generate → merge → diff → apply`); templates render per-IDE files, hujson AST patches JSONC in place (never `encoding/json`); Bubble Tea-leaning TUI as thin layer (tview fallback).

| IDE | Targets |
|-----|---------|
| `vscode` | `.vscode/{settings,extensions,launch,tasks}.json` (JSONC) |
| `cursor` | `.cursor/rules/*.mdc` + `.cursor/mcp.json`; document `.vscode` relationship |
| `devin` | `.devin/rules/*.md` (+`.windsurf` fallback, root `.windsurfrules` read-only, `AGENTS.md`) + `.devin/mcp_config.json` scopes |
| `kiro` | `.kiro/steering/*.md` (+product/tech/structure) + `.kiro/specs/<f>/{requirements,bugfix,design,tasks}.md` + `.kiro/settings/mcp.json` |

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod`, `cmd/stackup/`, `internal/` | New | Scaffold module, CLI layout, detector/merge/diff/apply services |
| `openspec/specs/` | New | Three capability specs land via sdd-spec deltas |
| existing code | None | Greenfield; nothing to migrate |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| JSONC comment loss via naive marshal | High | hujson AST round-trip only; golden-idempotence test |
| Array merge deletes user entries | Med | Union-no-delete rule + backup before write |
| Polyglot misreport (single winner) | Med | Ranked Evidence list + confidence thresholds in spec |
| CI hang on prompts | Med | Non-TTY auto-detect + `--non-interactive` from day one |

## Rollback Plan

Greenfield: delete scaffolded `go.mod`/`cmd/`/`internal/` if init fails. Post-release: restore from timestamped `.bak`; `apply` without backup forbidden; `--dry-run` writes nothing.

## Dependencies

- Go 1.27.1, Cobra, `tailscale/hujson`; TUI deps deferred to design

## Success Criteria

- [ ] MVP-4 detection returns ranked Evidence on fixture repos, polyglot-safe
- [ ] Per-IDE targets generated with merge-with-backup; `--dry-run` diff writes nothing
- [ ] `--format json` + non-TTY path runs without prompts
