# Design: stackup-wave2 — Four detectors, JS deepening, extensions

## Technical Approach

Extend the existing `Detector` pattern, don't reinvent it. Four new structs (`csharp`, `java`, `ruby`, `erlang`) copy `rust.go` shape (manifest→Medium, +lock/pin→High, hint→Low). `Evidence` gains additive `Runtime`/`Frameworks`; `node.go` deepens (Bun + presence-only frameworks); `generate.go` grows the recommendations map plus `HasBun`/framework flags. Ranking (`DetectAll` stable sort) is untouched.

## Architecture Decisions

| Decision | Options (tradeoff) | Choice + rationale |
|---|---|---|
| Append-after-rust order | Insert alphabetically (churns ties) vs append after rust (stable) | Append `csharp, java, ruby, erlang` after `rust`. Existing tie order never shifts; proposal locked call. |
| Gradle-wins duel | Two evidences (breaks single-evidence invariant) vs gradle-wins single evidence | Single Java evidence, `PackageManager=gradle`, both manifests in `Signals`. Preserves one-evidence-per-ecosystem rule used by `buildData`. |
| Presence-only frameworks | Parse semver/ranges (false precision, aggregator-root false positives) vs presence-only root scan | Presence-only over root `package.json` `dependencies`+`devDependencies`; record meta-framework plus base (`next`→`react`+`next`). No version logic, no recursion. |
| vscode-array-only extensions | Recommendations array per IDE (breaks cursor/devin/kiro prose contract) vs vscode-only array + prose elsewhere | Only `vscode/extensions.json.tmpl` renders the array. Cursor/devin/kiro mention IDs in prose. Keeps merge surface (union-no-delete) to one JSON target. |

## Data Flow

    FS root ──→ Registry.DetectAll ──→ []Evidence ──→ generate.buildData ──→ templates ──→ Plan
                      │ (stable sort High→Low)          │ (map+flags)          │ (array vs prose)

Sequence (detect→generate): `Detect(root)` runs registry order, returns enriched `Evidence` (with `Runtime`/`Frameworks`); `Build(evidences, ides)` maps ecosystems+frameworks to extension set, sets `Has*` flags, renders vscode array + prose targets deterministically (same input → same bytes).

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/detect/types.go` | Modify | Add `Runtime string` + `Frameworks []string` with `omitempty`; ranking untouched |
| `internal/detect/csharp.go` | Create | Root-only `*.csproj` glob + `*.sln[x]`; `packages.lock.json`/`global.json`→High; hint files→Low; TFM first-string after `global.json` |
| `internal/detect/java.go` | Create | `pom.xml` vs `build.gradle(.kts)`+`settings.gradle(.kts)` duel (gradle-wins, both in `Signals`); lock/wrapper/pin→High |
| `internal/detect/ruby.go` | Create | `Gemfile`/`*.gemspec`; `Gemfile.lock`/version-pin→High (`PackageManager=bundler`); version files alone→Low |
| `internal/detect/erlang.go` | Create | `rebar.config`→Medium; `rebar.lock`/`.tool-versions:erlang`→High (`PackageManager=rebar3`); ignore `mix.exs`, never fire on `.app.src`/`.erl` alone |
| `internal/detect/registry.go` | Modify | Append 4 detectors after `RustDetector{}`; update order comment |
| `internal/detect/node.go` | Modify | `bun.lock` presence-only (never read/parse); `packageManager` field first else `bun` + `Runtime=bun`; framework scan; single evidence |
| `internal/generate/generate.go` | Modify | Grow `vscodeExtensionRecommendations`; add `HasBun`/`HasCSharp`/`HasJava`/`HasRuby`/`HasErlang` + `Frameworks` to `templateData`; framework→extension lookup |
| `internal/generate/templates/cursor/rules.mdc.tmpl`, `devin/rules.md.tmpl`, `kiro/steering-tech.md.tmpl` | Modify | Prose mentions for new IDs/flags; no recommendations array |
| `internal/detect/detect_test.go`, `testdata/*` | Modify | Table-driven cases + fixtures (see Testing Strategy) |
| `internal/generate/generate_test.go`, `testdata/*.golden` | Modify | Framework/Bun cases; golden update via `-update` then rerun |

Non-obvious patterns (only ones needing snippets):

```go
// C#: root-only, never recursive — filepath.Glob(root/*.csproj), no WalkDir.
matches, _ := filepath.Glob(filepath.Join(root, "*.csproj"))
```

```go
// Frameworks: presence-only; meta maps to base. No semver.
if has(deps, "next") { frameworks = append(frameworks, "react", "next") }
```

## Interfaces / Contracts

`Evidence` stays backward compatible: new fields zero-value safe, `json:",omitempty"`, excluded from `sort.SliceStable` comparator. `buildData` unions `vscodeExtensionRecommendations[ecosystem]` plus framework/Bun IDs, dedupes via set, `sort.Strings` for determinism. Forbidden IDs (`rebornix.Ruby`, `octref.vetur`, `rust-lang.rust`, `JamesBirtles.svelte-vscode`) never enter the map. Exact verified casings: `Vue.volar`, `Shopify.ruby-lsp`, `svelte.svelte-vscode`, `astro-build.astro-vscode`, `oven.bun-vscode`, `ms-dotnettools.csharp`, `redhat.java`, `pgourlain.erlang`.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit (detect) | Each detector Medium/High/Low; Java duel; Bun coexistence; `next`→`react`; Erlang silence on weak signals | Table-driven `detect_test.go`, `t.TempDir()` + `writeFile`/`copyFixture`; never real home dir |
| Unit (generate) | React emits eslint-only; forbidden IDs absent; prose targets lack array; idempotent re-render | Table-driven + golden files (`-update`, inspect diff, rerun without) |
| Integration | Erlang without local `rebar3` | `testing.Short()` gate: skip external-command path in `-short`; fixture-only unit always runs. Fallback note: missing `rebar3` never fails detection — file signals alone decide |

Fixtures per setup: `csharp` (csproj alone), `csharp-pin` (+`global.json`), `java-maven`, `java-gradle`, `java-duel`, `ruby`, `ruby-pin`, `erlang`, `erlang-weak` (app.src+erl+mix.exs → silent), `node-bun` (bun.lock + package-lock coexistence), `node-next` (framework mapping).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Detection uses `os.Stat`/`ReadFile`/`Glob` on the scan root only; `bun.lock`/`bun.lockb` never executed or parsed.

## Migration / Rollout

No migration required. `Runtime`/`Frameworks` are additive JSON (old binaries ignore them); recommendations merge union-no-delete so re-render only adds IDs. Rollback: revert append + 4 files + fields, re-render; merged IDs linger by design (manual removal).

## Open Questions

None. Erlang low-install extension risk accepted per proposal (fixtures + skippable integration).
