# Proposal: stackup-wave2 — Four detectors, JS deepening, extensions

## Intent

Stackup detects only Node/Go/Python/Rust; C#, Java, Ruby, Erlang fall to unknown, and Node lacks Bun/framework signals for IDE recommendations. Wave-2 closes both gaps without changing ranking.

## Scope

### In Scope
- 4 detectors after rust; manifest→Medium, +lock/pin→High, hint→Low
- Optional `Runtime` + `Frameworks` on Evidence (zero-value safe, additive JSON)
- Node: Bun (`Runtime=bun`) + presence-only frameworks from root `package.json`
- Verified IDs into vscode `recommendations` (union-no-delete); prose for cursor/devin/kiro

### Out of Scope
- Wave-3, PHP, Elixir (`mix.exs` silent), monorepo deep-scan, semver, `bun.lockb` parsing, extension targets, stale removal

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `stack-detection`: 4 ecosystems + `Runtime`/`Frameworks` + Bun/framework rules
- `config-generation`: bigger recommendations table + flags + prose mentions

## Approach

Registry, Evidence, Node, map-only extensions.

| Locked call | Decision |
|---|---|
| Registry | `node, go, python, rust, csharp, java, ruby, erlang` |
| Java duel | `gradle` wins; both in `Signals` |
| Bun duel | `packageManager` field, else `bun`; `Runtime=bun` |
| TFM | first string only, after `global.json` |
| `mix.exs` | OUT |
| Erlang tests | fixtures + `testing.Short()` gate, no hard `rebar3` |

| Setup | vscode `recommendations` (+ prose for cursor/devin/kiro) |
|---|---|
| Node, React-Next | `dbaeumer.vscode-eslint` only (React=NONE) |
| Bun, Vue-Nuxt, Svelte-Kit, Astro | +`oven.bun-vscode` / +`Vue.volar` / +`svelte.svelte-vscode` / +`astro-build.astro-vscode` |
| C#, Java, Ruby, Erlang | +`ms-dotnettools.csharp` / +`redhat.java` / +`Shopify.ruby-lsp` / +`pgourlain.erlang` (low-usage risk) |
| Go, Python, Rust | `golang.go` / `ms-python.python` / `rust-lang.rust-analyzer` unchanged |

Forbidden: `rebornix.Ruby`, `octref.vetur`, `rust-lang.rust`, `JamesBirtles.svelte-vscode`.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `internal/detect/csharp.go, java.go, ruby.go, erlang.go` | New | one Detector each |
| `internal/detect/registry.go, types.go, node.go` | Modified | order; fields; Bun+frameworks |
| `internal/detect/detect_test.go, testdata/*` | Modified | fixtures per setup |
| `internal/generate/generate.go`, cursor/devin/kiro templates | Modified | map, flags, prose |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Erlang low-install extension, no local `rebar3` | Med | fixtures + skippable integration; fallback note |
| Gradle/Maven duality | Med | single evidence; gradle-wins; both in `Signals` |
| Framework false positives (aggregator roots) | Med | presence-only, root-only; deep-scan to backlog |

## Rollback Plan

Revert append + 4 files + fields (additive JSON, old binaries ignore); re-render. Merged recommendations linger by design — manual removal.

## Dependencies

- None (IDs verified 2026-09-13).

## Success Criteria

- [ ] New detectors: Medium alone, High with lock/pin, Low on hint
- [ ] Bun sets `Runtime=bun`; frameworks presence-only, root-only; single Node evidence
- [ ] Union without deletes; re-render idempotent; `go test ./...` green
