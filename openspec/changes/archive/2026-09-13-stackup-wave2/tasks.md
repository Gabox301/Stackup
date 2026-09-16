# Tasks: stackup-wave2 — Four detectors, JS deepening, extensions

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~600–700 authored (+ regenerated goldens) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 (stacked to main) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Evidence fields + 4 detectors + registry order | PR 1 | `go test ./internal/detect/ -count=1` | N/A — library-only, no service boundary | `internal/detect/types.go`, `registry.go`, `csharp.go`, `java.go`, `ruby.go`, `erlang.go` |
| 2 | Node Bun runtime + framework scan | PR 2 | `go test ./internal/detect/ -run 'Bun\|Framework' -count=1` | N/A — file-signal logic, fixtures prove it | `internal/detect/node.go` + node fixtures/cases |
| 3 | Extension map + prose + goldens + lint | PR 3 | `go test ./internal/generate/ -count=1` | Re-render twice on fixtures, bytes equal | `internal/generate/generate.go`, 3 prose templates, goldens |

## Phase 1: Evidence + new detectors

- [x] 1.1 Add `Runtime` + `Frameworks` (`omitempty`) to `internal/detect/types.go`, ranking untouched (additive-fields scenario)
- [x] 1.2 Create `internal/detect/csharp.go`: root-only `*.csproj` glob + `*.sln[x]`, pin→High, hints→Low (C# glob scenario)
- [x] 1.3 Create `internal/detect/java.go`: gradle-wins duel, both manifests in `Signals`, lock/wrapper/pin→High (duel scenario)
- [x] 1.4 Create `internal/detect/ruby.go`: `Gemfile`/`*.gemspec`, `PackageManager=bundler`, lock/pin→High (ladder)
- [x] 1.5 Create `internal/detect/erlang.go`: `rebar.config`→Medium, lock/pin→High, ignore `mix.exs` (weak-silence scenario)
- [x] 1.6 Append `csharp, java, ruby, erlang` after rust in `internal/detect/registry.go` + order comment

## Phase 2: Node deepening

- [x] 2.1 Deepen `internal/detect/node.go`: `bun.lock` presence-only, `packageManager`-first else `bun`, `Runtime=bun`, single evidence (bun-coexistence scenario)
- [x] 2.2 Add root-only presence-only framework scan in `internal/detect/node.go` incl `next`→`react`+`next`, no semver (framework scenario)

## Phase 3: Generate map + prose

- [x] 3.1 Grow `vscodeExtensionRecommendations` in `internal/generate/generate.go` with exact casings (`Vue.volar`, `Shopify.ruby-lsp`, `svelte.svelte-vscode`, `astro-build.astro-vscode`, `oven.bun-vscode`, `ms-dotnettools.csharp`, `redhat.java`, `pgourlain.erlang`) + `Has*`/`Frameworks` flags, forbidden IDs excluded
- [x] 3.2 Add prose mentions (no array) to `internal/generate/templates/cursor/rules.mdc.tmpl`, `internal/generate/templates/devin/rules.md.tmpl`, `internal/generate/templates/kiro/steering-tech.md.tmpl` (non-vscode scenario)

## Phase 4: Tests, goldens, lint

- [x] 4.1 Add fixtures under `internal/detect/testdata/`: `csharp`, `csharp-pin`, `java-maven`, `java-gradle`, `java-duel`, `ruby`, `ruby-pin`, `erlang`, `erlang-weak`, `node-bun`, `node-next`
- [x] 4.2 Extend `internal/detect/detect_test.go`: ladder per detector + `erlang-weak` silence + `java-duel` gradle-wins + `node-bun` single evidence + `node-next` mapping; `testing.Short()` gate for `rebar3` path
- [x] 4.3 Extend `internal/generate/generate_test.go`: React eslint-only + forbidden-IDs-negative (Ruby+Svelte) + union-idempotent re-render + cursor prose C#
- [x] 4.4 Refresh goldens in `internal/generate/testdata/*.golden` via `-update`, inspect diff, rerun without flag
- [x] 4.5 Run `gofmt`, `go vet ./...`, `golangci-lint run`, full `go test ./...` green

Order: Phase 1 → 2 → 3 → 4; each phase's tests ship with its unit per work-unit-commits.
