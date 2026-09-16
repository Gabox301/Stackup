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
### Requirement: Extended verified recommendations table

The system MUST emit these exact IDs into vscode `recommendations`:

| Setup | Recommendations |
|---|---|
| Node, React, Next | `dbaeumer.vscode-eslint` only (React=NONE) |
| Bun | + `oven.bun-vscode` |
| Vue / Nuxt | + `Vue.volar` |
| Svelte / SvelteKit | + `svelte.svelte-vscode` |
| Astro | + `astro-build.astro-vscode` |
| C# | + `ms-dotnettools.csharp` |
| Java | + `redhat.java` |
| Ruby | + `Shopify.ruby-lsp` |
| Erlang | + `pgourlain.erlang` |
| Go / Python / Rust | `golang.go`, `ms-python.python`, `rust-lang.rust-analyzer` unchanged |

The system MUST NOT emit `rebornix.Ruby`, `octref.vetur`, `rust-lang.rust`, or `JamesBirtles.svelte-vscode`.

#### Scenario: React emits eslint only

- GIVEN React evidence without Bun
- WHEN vscode extensions render
- THEN recommendations contain `dbaeumer.vscode-eslint` and no React ID

#### Scenario: Forbidden IDs never emitted

- GIVEN Ruby plus Svelte evidence
- WHEN generation runs
- THEN output contains `Shopify.ruby-lsp` and `svelte.svelte-vscode`, never the forbidden variants

### Requirement: Union merge and prose targets for extended table

The system MUST union new IDs without deleting user entries, MUST stay byte-idempotent on re-render, and MUST render cursor/devin/kiro as prose (no recommendations array).

#### Scenario: Bigger table merges safely

- GIVEN existing recommendations with custom IDs
- WHEN generation re-runs twice
- THEN custom IDs survive, new IDs appear once, and second bytes equal first

#### Scenario: Non-vscode targets use prose

- GIVEN C# evidence targeting cursor
- WHEN generation runs
- THEN cursor rules mention `ms-dotnettools.csharp` in prose, no extensions array
