# Apply progress: stackup-wave2 — Slices 1–3 (Work Units 1–3 / PRs 1–3, COMPLETE)

**Change**: stackup-wave2
**Mode**: Standard (strict_tdd: false per openspec/config.yaml)
**Scope**: Slice 3 = Phase 3 + Phase 4, tasks 3.1–3.2, 4.1–4.5 ONLY. No detect changes except the `rebar3` Short-gate test. Slices 1 (Phase 1, 1.1–1.6) and 2 (Phase 2, 2.1–2.2) preserved below.
**Delivery**: chained PRs, stacked-to-main. This is PR 3 of 3 (FINAL).

## Completed tasks

- [x] 1.1 `Runtime` + `Frameworks` (`omitempty`) in `internal/detect/types.go`, ranking untouched
- [x] 1.2 `internal/detect/csharp.go` (root-only `*.csproj` glob + `*.sln[x]`, pin→High, hints→Low, global.json-first then first-TFM hint)
- [x] 1.3 `internal/detect/java.go` (gradle-wins duel, both manifests in `Signals`, lock/wrapper/pin→High)
- [x] 1.4 `internal/detect/ruby.go` (`Gemfile`/`*.gemspec`, `PackageManager=bundler`, lock/pin→High)
- [x] 1.5 `internal/detect/erlang.go` (`rebar.config`→Medium, lock/pin→High, `mix.exs` ignored, never fires on `.app.src`/`.erl` alone)
- [x] 1.6 Registry append `csharp, java, ruby, erlang` after rust + order comment
- [x] 2.1 Deepen `internal/detect/node.go`: `bun.lock`/`bun.lockb` presence-only (never read/parse), `packageManager`-first else `bun`, `Runtime=bun`, single evidence (bun-coexistence scenario)
- [x] 2.2 Root-only presence-only framework scan in `internal/detect/node.go` (`react`, `vue`, `svelte`, `astro`, `next`, `nuxt`, `sveltekit`; `next`→`react`+`next`, `nuxt`→`vue`+`nuxt`, `sveltekit`→`svelte`+`sveltekit`), no semver, no recursion (framework scenario)
- [x] 3.1 Grew `vscodeExtensionRecommendations` in `internal/generate/generate.go` with exact casings (`Vue.volar`, `Shopify.ruby-lsp`, `svelte.svelte-vscode`, `astro-build.astro-vscode`, `oven.bun-vscode`, `ms-dotnettools.csharp`, `redhat.java`, `pgourlain.erlang`) + `HasCSharp`/`HasJava`/`HasRuby`/`HasErlang`/`HasBun`/`Frameworks` flags + framework→extension lookup, forbidden IDs excluded
- [x] 3.2 Added prose mentions (no array) to `cursor/rules.mdc.tmpl`, `devin/rules.md.tmpl`, `kiro/steering-tech.md.tmpl` (non-vscode scenario)
- [x] 4.1 Fixtures `csharp`, `csharp-pin`, `java-maven`, `java-gradle`, `java-duel`, `ruby`, `ruby-pin`, `erlang`, `erlang-weak`, `node-bun`, `node-next` — all present (created in slices 1–2), verified
- [x] 4.2 `detect_test.go`: ladder + `erlang-weak` silence + `java-duel` gradle-wins + `node-bun` single evidence + `node-next` mapping (slices 1–2) + new `TestDetectErlangWithoutRebar3` with `testing.Short()` gate
- [x] 4.3 `generate_test.go`: React eslint-only + forbidden-IDs-negative (Ruby+Svelte) + union-idempotent re-render + cursor prose C#
- [x] 4.4 Goldens refreshed via `-update` (zero diff — node-only conditionals false), reran without flag green
- [x] 4.5 `gofmt`, `go vet ./...`, `golangci-lint run`, full `go test ./...` green

## Files changed (Slice 3)

| File | Action | What was done |
|------|--------|---------------|
| `internal/generate/generate.go` | Modified | Extended `vscodeExtensionRecommendations` (csharp/java/ruby/erlang exact casings) + `frameworkExtensionRecommendations` (vue/nuxt→Vue.volar, svelte/sveltekit→svelte.svelte-vscode, astro→astro-build.astro-vscode; react/next NONE) + `HasCSharp`/`HasJava`/`HasRuby`/`HasErlang`/`HasBun`/`Frameworks` on templateData + bun/framework union with set-dedupe + `sort.Strings` + `contains` template func |
| `internal/generate/templates/cursor/rules.mdc.tmpl` | Modified | Prose mentions for all 8 new IDs via `Has*`/`contains` conditionals; no recommendations array |
| `internal/generate/templates/devin/rules.md.tmpl` | Modified | Same prose mentions as cursor, Devin scope-notes style |
| `internal/generate/templates/kiro/steering-tech.md.tmpl` | Modified | Same prose mentions as cursor, Kiro bullet style after HasRust |
| `internal/detect/detect_test.go` | Modified | Added `TestDetectErlangWithoutRebar3` (`testing.Short()` gate, `exec.LookPath`, file-signals-decide fallback); ladder/duel/silence/coexistence/mapping matrix from slices 1–2 untouched |
| `internal/generate/generate_test.go` | Modified | Added `TestReactEmitsEslintOnly`, `TestForbiddenIDsNeverEmitted`, `TestUnionIdempotentRerender` (9-ID union exactly-once + bytes-equal), `TestCursorProseMentionsCSharp` |
| `openspec/changes/stackup-wave2/tasks.md` | Modified | Marked 3.1–3.2, 4.1–4.5 `[x]` |

## Files changed (Slice 2, preserved)

| File | Action | What was done |
|------|--------|---------------|
| `internal/detect/node.go` | Modified | Bun presence-only (`bun.lock`/`bun.lockb` existence only, High + `Runtime=bun`, field-first else `bun`); `nodeFrameworks` root-only presence-only scan (`dependencies`+`devDependencies`, meta→base mapping, no semver/recursion); single evidence preserved |
| `internal/detect/detect_test.go` | Modified | `TestDetectBun` (3 cases: coexistence field-first, bun-alone else-bun, bun.lockb binary presence-only) + `TestDetectFrameworks` (7 cases: next→react+next, react, nuxt→vue+nuxt, sveltekit→svelte+sveltekit, devDeps, astro+svelte, nested root-only) |
| `internal/detect/testdata/node-bun/*` | Created | `package.json` (packageManager npm field) + `package-lock.json` + `bun.lock` coexistence fixture |
| `internal/detect/testdata/node-next/*` | Created | `package.json` with `next@^14` proving semver-agnostic mapping |

## Files changed (Slice 1, preserved)

| File | Action | What was done |
|------|--------|---------------|
| `internal/detect/types.go` | Modified | Added `Runtime string` + `Frameworks []string`, both `json:",omitempty"`; comparator untouched |
| `internal/detect/csharp.go` | Created | Root-only Glob, `*.sln`/`*.slnx`, High via `packages.lock.json`/`global.json`, Low via dotnet hints, global.json→TFM hint |
| `internal/detect/java.go` | Created | Duel (gradle-wins, both in Signals), High via lock/wrapper/`.mvn`/pins, `.tool-versions:java` gating, `toolVersionsValue` helper |
| `internal/detect/ruby.go` | Created | `Gemfile` + root `*.gemspec`, `PM=bundler`, High via lock/pins incl `.tool-versions:ruby` |
| `internal/detect/erlang.go` | Created | `rebar.config`, `PM=rebar3`, High via `rebar.lock`/`.tool-versions:erlang`, Low via erlang pin alone, weak-signal silence |
| `internal/detect/registry.go` | Modified | Appended 4 detectors after `RustDetector{}`, updated order comment |
| `internal/detect/detect_test.go` | Modified | `TestDetectWave2` ladder (13 cases) + `TestDetectWave2Contracts` (duel signals, csharp pin hint) |
| `internal/detect/testdata/{csharp,csharp-pin,java-maven,java-gradle,java-duel,ruby,ruby-pin,erlang,erlang-weak}/*` | Created | Minimal fixtures proving slice-1 tasks |

## Work Unit Evidence (Slice 3)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/detect/ ./internal/generate/ -count=1`: `ok stackup/internal/detect` + `ok stackup/internal/generate` (all new tests PASS: React-eslint-only, Forbidden-negative, Union-idempotent, Cursor-prose-C#, ErlangWithoutRebar3) |
| Runtime harness command/scenario and exact result | Re-render twice bytes-equal: `go test ./internal/generate/ -run 'TestUnionIdempotentRerender\|TestRerenderIsIdempotent' -count=1 -v` → both PASS; 9-ID union appears exactly once, second bytes equal first |
| Rollback boundary | Revert `internal/generate/generate.go` + 3 prose templates + new test blocks in `detect_test.go`/`generate_test.go`; fixtures/goldens untouched by this slice (golden `-update` produced zero diff) |

## Work Unit Evidence (Slice 2, preserved)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/detect/ -run 'Bun\|Framework' -count=1`: `ok stackup/internal/detect 0.435s` (rerun after gofmt: `ok stackup/internal/detect 0.191s`; all 10 subtests PASS) |
| Runtime harness command/scenario and exact result | N/A — file-signal logic, no service boundary; fixtures + table tests prove it (Unit 2 row) |
| Rollback boundary | Revert `internal/detect/node.go` + `detect_test.go` Bun/Framework blocks + delete `testdata/node-bun/`, `testdata/node-next/`; no other files touched |

## Work Unit Evidence (Slice 1, preserved)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test ./internal/detect/ -count=1`: `ok stackup/internal/detect 0.544s` (rerun in full suite: `ok ... 0.211s`) |
| Runtime harness command/scenario and exact result | N/A — library-only file-signal logic, no service boundary; fixtures + table tests prove it (Unit 1 row) |
| Rollback boundary | Revert `internal/detect/types.go`, `registry.go`, `detect_test.go` + delete `csharp.go`, `java.go`, `ruby.go`, `erlang.go` + 9 fixture dirs; no other files touched |

## Verification (foreground, Slice 3)

- `go build ./...`: success, no output
- `go test ./internal/detect/ ./internal/generate/ -count=1`: both `ok`
- `go test ./... -count=1`: all 7 packages `ok` (cmd/stackup, apply, detect, diff, generate, merge, tui)
- `go test ./internal/generate/ -update -count=1`: `ok`, zero golden diff (node-only conditionals false by design)
- `gofmt -l .`: empty (clean, after one alignment fix in generate_test.go)
- `go vet ./...`: empty (clean)
- `golangci-lint run ./...`: `0 issues`
- Harness re-render twice: `TestUnionIdempotentRerender` + `TestRerenderIsIdempotent` PASS, 9-ID union exactly-once

## Deviations from design

None — implementation matches design. Interpretations locked: (Slice 3) React/Next map to no framework extension (React=NONE, eslint covers); `nuxt` implies `vue` and `sveltekit` implies `svelte` so prose may mention `Vue.volar`/`svelte.svelte-vscode` twice with different lead-ins (informative, not a bug); array dedupes via set so `extensions.json` holds each ID exactly once; `contains` template func added for framework prose (no extra templateData flags beyond spec). (Slices 1–2 interpretations preserved.)

## Issues found

None. Full suite green, no regressions.

## Remaining (NOT this slice)

None — all wave-2 tasks 1.1–4.5 complete. Ready for `sdd-verify`.

## Workload / PR boundary

- Mode: chained PR slice (stacked-to-main), PR 3 of 3 (FINAL)
- Current work unit: Unit 3 — Extension map + prose + goldens + lint
- Boundary: starts after slice-2 node deepening, ends after generate map + 3 prose templates + matrix tests + golden refresh + lint; excludes detect detectors/node logic (prior slices)
- Size note: Unit 3 authored diff within the 400-line budget (generate.go map+flags+lookup, 3 templates prose-only conditionals, 2 test files with 5 new tests; golden `-update` produced zero diff so no golden churn counted). Exact count reported at commit.
