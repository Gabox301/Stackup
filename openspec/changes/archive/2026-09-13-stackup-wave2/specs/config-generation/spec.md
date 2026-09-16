# Delta for config-generation

## ADDED Requirements

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
