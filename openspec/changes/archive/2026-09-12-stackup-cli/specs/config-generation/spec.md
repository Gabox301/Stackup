# Config Generation Specification

## Purpose

Render per-IDE configs from Evidence; merge JSONC safely with pre-write backup.

## Requirements

### Requirement: Per-IDE target rendering

The system MUST render templates to these targets:

| IDE | Targets |
|-----|---------|
| `vscode` | `.vscode/{settings,extensions,launch,tasks}.json` |
| `cursor` | `.cursor/rules/*.mdc` + `.cursor/mcp.json` |
| `devin` | `.devin/rules/*.md` + `.devin/mcp_config.json` (project scope; `.windsurf` fallback) |
| `kiro` | `.kiro/steering/*.md` + `.kiro/specs/**` + `.kiro/settings/mcp.json` |

#### Scenario: Fresh render creates targets

- GIVEN Node evidence and target `vscode`
- WHEN generation runs with no existing files
- THEN all four vscode targets are created from templates

#### Scenario: Re-render is idempotent

- GIVEN identical evidence and existing generated files
- WHEN generation re-runs
- THEN output bytes are unchanged

### Requirement: Safe JSONC merge with backup

The system MUST merge JSONC via AST preserving comments and trailing commas, MUST NOT overwrite user files with naive `encoding/json` marshal, MUST union arrays without deleting user entries, MUST back up with timestamp before any write, and MUST overwrite only under an explicit flag.

#### Scenario: Comment-preserving merge

- GIVEN `settings.json` with comments and custom keys
- WHEN generation merges new keys
- THEN comments and custom keys survive with new keys added

#### Scenario: Backup precedes overwrite

- GIVEN an existing `settings.json` and the overwrite flag set
- WHEN apply writes
- THEN a timestamped `.bak` predates the write
