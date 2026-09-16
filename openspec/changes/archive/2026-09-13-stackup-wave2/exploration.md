# Exploration: stackup-wave2 — New Detectors (C#, Java, Ruby, Erlang) + JS Deepening (Bun, Frameworks) + Extension Recommendations

## Current State

`internal/detect` runs a `Detector` registry in fixed order (`node, go, python, rust`) and returns
`[]Evidence` sorted High → Low (stable sort preserves registry order on ties):

```go
type Evidence struct {
    Ecosystem string; Confidence Confidence; Signals []string
    VersionHint, PackageManager string
}
type Detector interface { Name() string; Detect(root string) []Evidence }
```

Grade contract (documented in `types.go`, enforced in `detect_test.go`):
manifest alone = Medium; manifest + lockfile/toolchain pin = High; version-manager hint alone = Low.
All detectors are root-only (`exists(root, name)` — no recursion), never error (missing/unreadable
files yield no evidence), and record firing filenames in `Signals` in detection order.

`internal/generate.Build` maps `Ecosystem` → template flags (`HasNode/HasGo/HasPython/HasRust`) and a
sorted-unique `Extensions` list via `vscodeExtensionRecommendations`
(`node→dbaeumer.vscode-eslint`, `go→golang.go`, `python→ms-python.python`, `rust→rust-lang.rust-analyzer`),
rendered into `.vscode/extensions.json` (`{"recommendations": [...]}` template). `internal/merge`
(`hujson` AST `deepMerge`/`UnionArray`) unions arrays without deleting user entries — the
union-no-delete invariant the new extension dimension must respect. Re-render is byte-idempotent.

## Affected Areas

- `internal/detect/types.go` — likely `Evidence` enrichment (`Runtime`, `Frameworks` optional fields, zero-value safe).
- `internal/detect/registry.go` — append `CSharpDetector{}`, `JavaDetector{}`, `RubyDetector{}`, `ErlangDetector{}` to `DefaultRegistry()`; order decision needed (recommend append after rust to keep existing tie-breaks stable).
- `internal/detect/csharp.go` (new), `java.go` (new), `ruby.go` (new), `erlang.go` (new) — one file per detector, following `go.go`/`rust.go` pattern.
- `internal/detect/node.go` — Bun runtime corroboration + `package.json` dependency framework scan (no new ecosystem).
- `internal/detect/detect_test.go` + `internal/detect/testdata/{csharp,java-maven,java-gradle,ruby,erlang,node-bun,node-react,node-vue,node-svelte,node-astro}/` — table-driven fixtures per go-testing skill (`t.TempDir()`, `t.Run`, scenario-named cases).
- `internal/generate/generate.go` — `vscodeExtensionRecommendations` growth + `HasX`/`HasFramework` template flags; possibly `Runtime`-aware task/launch hints.
- `internal/generate/templates/vscode/extensions.json.tmpl` — unchanged shape (array union covers new IDs); cursor/devin/kiro templates gain prose mentions only (no new extension-file targets).
- `openspec/specs/stack-detection/spec.md`, `openspec/specs/config-generation/spec.md` — delta specs in propose/spec phases (new signals, grades, extension table).

## Approaches

### A. New detectors: extend registry (one struct per ecosystem)

DTO pattern already proven. Each new file implements `Name()` + `Detect(root)` with the manifest→Medium / +lock-or-pin→High / version-hint-alone→Low ladder.

| Ecosystem | Manifest → Medium | Corroborating → High (lockfile / toolchain pin) | Weak hint alone → Low | PackageManager | VersionHint priority |
|---|---|---|---|---|---|
| C# (.NET) | `*.csproj` (root glob) or `*.sln` / `*.slnx` | `packages.lock.json`; `global.json` (`sdk.version`) | `.config/dotnet-tools.json` alone; `NuGet.config` alone; `Directory.Build.props` alone | `dotnet` | `global.json:sdk.version` → `TargetFramework(s)` best-effort from csproj XML |
| Java | `pom.xml` (maven) or `build.gradle` / `build.gradle.kts` + `settings.gradle(.kts)` (gradle) | gradle: `gradle.lockfile`, `gradle/verification-metadata.xml`, `gradlew` wrapper; maven: `mvnw` / `.mvn/` wrapper; either: `.java-version` / `.sdkmanrc` / `.tool-versions:java` pin | `.java-version` / `.sdkmanrc` / `.tool-versions` alone | `maven` vs `gradle` (must record; see risk) | `.java-version` → `.sdkmanrc` → manifest best-effort (`maven.compiler.release`, gradle toolchain `languageVersion`) |
| Ruby | `Gemfile`, `*.gemspec` | `Gemfile.lock`; `.ruby-version` / `.rbenv-version` / `.tool-versions:ruby` pin | `.ruby-version` / `.tool-versions` alone | `bundler` | `.ruby-version` first |
| Erlang | `rebar.config` | `rebar.lock`; `.tool-versions:erlang` pin | `.tool-versions` alone; `*.app.src` alone MUST NOT fire (build artifact, too weak); `*.erl` alone MUST NOT fire | `rebar3` | `.tool-versions:erlang` entry |
| Adjacent (out of scope, noted) | `mix.exs` (Elixir) | `mix.lock` | — | `mix` | — |

- Pros: minimal diff, consistent with 4 existing detectors, trivially table-testable, no ranking-model change.
- Cons: `*.csproj` needs a root-only `Glob` (not `exists`); Java duality lives inside one detector (branching).
- Effort: Low.

Alternative **B: sub-detectors** (`MavenDetector` + `GradleDetector`, per-framework detectors, `BunDetector` as separate ecosystem): rejected for explore — explodes registry order/tie-break surface, confuses ranking (is `react` High? it depends on `node`), risks single-winner-collapse regressions. Report only.

### B. JS deepening inside Node detector (no new ecosystems)

**Bun runtime.** Manifest stays `package.json` (Medium unchanged). Corroborating Bun signals upgrade to High and set `Runtime=bun` (with `PackageManager=bun` when `packageManager: bun@*` or bun lockfile is the resolving lock):
presence signals: `bun.lock`, `bun.lockb` (legacy binary — existence check only, never parse), `bunfig.toml`, `.bun-version`, `packageManager: ^bun@`, `engines.bun`.
Edge: `bunfig.toml` without `package.json` → Low hint (mirrors `.nvmrc`-alone precedent), not Medium.
Coexistence: repo with `package-lock.json` AND `bun.lock` keeps single Node evidence; `PackageManager` prefers explicit `packageManager` field, then first lockfile in deterministic order (document priority; bun last-wins vs npm-first is a design-phase call).

**Framework signals.** Scan `package.json` `dependencies` + `devDependencies` (ignore `peerDependencies` — too noisy); presence-only, NO semver interpretation (`^`, `~`, `workspace:*`, `catalog:` are all "present"):

| Signal | Trigger deps (any one fires) | Meta-framework note |
|---|---|---|
| React | `react`, `react-dom` | `next` implies React — record both (`Frameworks=[react,next]`) |
| Vue | `vue` | `nuxt` / `@nuxt/kit` implies Vue — record both |
| Svelte | `svelte` | `@sveltejs/kit` implies Svelte — record both |
| Astro | `astro` | standalone; `@astrojs/*` integrations corroborate but `astro` alone suffices |
| Next / Nuxt / SvelteKit | `next`, `nuxt`/`nuxt3`/`@nuxt/kit`, `@sveltejs/kit` | report as frameworks list entries, don't lock taxonomy in explore |

Version ranges: explicitly NOT evaluated in wave2 (brittle, zero generation value today; raw spec string may be kept as hint later).
Monorepo edge (known limitation, report-don't-lock): detectors are root-only; `apps/*/package.json` and `packages/*/package.json` are invisible, so a root `package.json` with only a `workspaces` field and no direct framework dep yields a framework-less Node evidence (false negative), while a root aggregator depending on everything yields over-broad frameworks (false positive). Wave2 stays root-only for determinism; a bounded-depth workspace scan goes to backlog with its own ranking/false-positive analysis.

### C. Evidence enrichment: add `Runtime`/`Frameworks` fields vs new Evidence kinds

**Recommended: enrich `Evidence` (backward-compatible).**

```go
// optional, zero-value empty when undetected
Runtime string    // e.g. "bun"
Frameworks []string // e.g. ["react","next"]
```

- Pros: one Evidence per ecosystem preserved; sorting/ranking untouched; `generate.buildData` gains flags without contract break; JSON output (`--format json`) gains additive fields only.
- Cons: template flag growth (`HasBun`, `HasReact`, …) needs a deterministic mapping table in design.
- Effort: Low.

Alternative **B: new Evidence kinds** (`Ecosystem=bun`, `Ecosystem=react`, …): rejected — breaks one-per-ecosystem mental model, forces grade semantics for derived signals (is React High?), risks polyglot-collapse and spec churn across `detect|generate|diff|apply --format json`. Report only.

### D. Per-setup extension recommendations (union-no-delete)

Shape is fixed: generated `recommendations` array unions into the user's existing array via `merge` (never deletes). Research-grade candidate table — **all IDs below are UNVERIFIED and MUST be verified against Marketplace/Open VSX in the research phase before any spec locks them**:

| Setup | VSCode (`extensions.json` recommendations) | Cursor / Devin / Kiro equivalent |
|---|---|---|
| Node base | `dbaeumer.vscode-eslint` (existing, keep) | prose in `.cursor/rules`, `.kiro/steering` (no extension-file target; Cursor reads shared `.vscode/*` import-only per prior design verdict) |
| Bun runtime | `oven.bun-vscode` (?) + base | same prose-only rule |
| React / Next | base only (no extra ID in wave2; `esbenp.prettier-vscode` and tailwind-gated extras are second-order — deferred) | prose |
| Vue / Nuxt | `Vue.volar` (?) (Vetur→Volar migration — verify) | prose |
| Svelte / SvelteKit | `svelte.svelte-vscode` (?) | prose |
| Astro | `astro-build.astro-vscode` (?) | prose |
| C# | `ms-dotnettools.csharp` (?) (+ `ms-dotnettools.csdevkit` (?) for solution management — verify whether to recommend pack vs single) | prose |
| Java | `vscjava.vscode-java-pack` (?) vs split (`redhat.java` + `vscjava.vscode-maven` + `vscjava.vscode-gradle-for-java`) — verify; pack is simpler for union arrays | prose |
| Ruby | `Shopify.ruby-lsp` (?) (vs deprecated `rebornix.Ruby` — verify current publisher) | prose |
| Erlang | `pgourlain.erlang` (?) — low-install-base, verify; fallback is no recommendation + docs note | prose |
| Go/Python/Rust | unchanged existing IDs | unchanged |

Authoritative sources for verification: `marketplace.visualstudio.com` (publisher.ID) and `open-vsx.org` (Cursor/air-gap parity). Cursor/Devin/Kiro have no `recommendations`-array equivalent — any new JSON target there would be inventing a contract; wave2 adds prose mentions only. Stale recommendations are never removed (union-no-delete is intended behavior, matching `merge` contract — document, don't "fix").

## Recommendation

1. **Registry extension** (approach A) with the signal table above; `DefaultRegistry` appends `csharp, java, ruby, erlang` after `rust` to keep existing tie-breaks stable.
2. **Evidence enrichment** (`Runtime string`, `Frameworks []string`) — no new Evidence kinds; Bun sets `Runtime`, frameworks append to `Frameworks` (meta-frameworks record both base + meta).
3. **Root-only, presence-only** JS deepening in wave2; monorepo depth scan explicitly deferred to backlog.
4. **Extension table stays research-grade** until marketplace verification; vscode-array-only delivery + prose elsewhere.

## Risks

- **Erlang toolchain scarcity on maintainer machine** — no local `rebar3` to e2e-verify beyond fixtures; mitigate with hand-built `rebar.config`/`rebar.lock`/`.tool-versions` fixtures and `testing.Short()`-gated integration (per go-testing skill), never a hard external-command dependency.
- **Gradle-vs-Maven duality** — both manifests can coexist (migrations); single Java evidence needs a documented `PackageManager` resolution priority (design call) and must not emit two competing Java evidences that confuse ranking/generation.
- **.NET monolith manifests** — `*.csproj` requires `Glob`, not `Stat`; constrain to root-only (depth 0) to bound cost, and define multi-`csproj` behavior (single evidence, all hits in `Signals`) so solutions with N projects don't emit N evidences.
- **Framework false-positives/negatives** — `devDependencies` inclusion vs exclusion, aggregator root `package.json`, and `peerDependencies` noise; presence-only + root-only keeps wave2 deterministic but guarantees monorepo edge misses (accepted, documented).
- **`bun.lockb` is binary** — existence-only signal; any content sniffing is out of scope (parse risk, zero value).
- **Extension ID drift/authority** — publisher renames (Vetur→Volar, Ruby→Ruby LSP), Cursor marketplace mirror lag, low-install Erlang extensions; every ID above is research-grade until live-verified. Wrong IDs ship broken recommendations that union-no-delete then preserves — verification is a hard gate before spec.
- **Union-no-delete staleness** — removed stacks' extensions linger in user arrays by design; must be documented as intended (matches `merge` contract), not treated as a bug in verify.

## Ready for Proposal

Yes — scope is bounded (4 detectors + Node enrichment + extension table), patterns are settled (registry + Evidence enrichment + table-driven fixtures), and open questions are design-phase sized (registry order lock, Java PM priority, bun-vs-npm lock priority, extension ID verification, `TargetFramework` parsing depth, `mix.exs` adjacency verdict). No user clarification needed before proposal.
