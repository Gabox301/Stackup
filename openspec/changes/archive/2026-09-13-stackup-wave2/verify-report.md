```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:42cc13d172f567b20ddbae9c534d68ae66ab4307887cc62dfc413fa282794e0d
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 5/5
scenarios: 10/10
test_command: go test ./... -count=1
test_exit_code: 0
test_output_hash: sha256:97f842666867c7f302dfd26b5fe41ee53418aaf5988baa3d1fcaf4c4eef6ebb8
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: stackup-wave2
**Version**: N/A
**Mode**: Standard (strict_tdd: false per openspec/config.yaml)
**Scope**: FULL — delta specs (stack-detection + config-generation), design, all 15 tasks, 3 slices (commits 8d9819c, 6fae378, 76ecabc on top of archived stackup-cli)

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 15 (1.1–1.6, 2.1–2.2, 3.1–3.2, 4.1–4.5) |
| Tasks complete | 15 |
| Tasks incomplete | 0 |

All tasks `[x]` in `openspec/changes/stackup-wave2/tasks.md`; apply-progress confirms slices 1–3 complete.

### Build & Tests Execution

**Build**: ✅ Passed (exit 0, empty output)
```text
go build ./...
(no output)
```

**Tests**: ✅ 7/7 packages passed, 0 failed, 0 skipped
```text
go test ./... -count=1
ok  	stackup/cmd/stackup	1.234s
ok  	stackup/internal/apply	0.547s
ok  	stackup/internal/detect	0.471s
ok  	stackup/internal/diff	0.504s
ok  	stackup/internal/generate	0.202s
ok  	stackup/internal/merge	0.204s
ok  	stackup/internal/tui	0.660s
```

Targeted re-runs (all PASS, exit 0):
- `go test ./internal/detect/ -count=1 -v -run 'TestDetectWave2|TestDetectBun|TestDetectFrameworks|TestDetectErlangWithoutRebar3'` — 13-case ladder + duel-contracts + 3 Bun cases + 7 framework cases + Erlang Short-gate.
- `go test ./internal/generate/ -count=1 -v -run 'TestReactEmitsEslintOnly|TestForbiddenIDsNeverEmitted|TestUnionIdempotentRerender|TestCursorProseMentionsCSharp|TestRerenderIsIdempotent'` — all 5 PASS.

Static checks: `gofmt -l .` empty, `go vet ./...` clean, `golangci-lint run ./...` → `0 issues`.

**Coverage** (`go test ./... -count=1 -cover`, no threshold configured → ➖ informational): cmd 65.7%, apply 77.1%, detect 76.4%, diff 90.9%, generate 85.9%, merge 83.9%, tui 81.2%.

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| Wave-2 ecosystem detectors | C# glob upgrades with pin | `internal/detect/detect_test.go > TestDetectWave2/csharp_manifest_alone_grades_medium + csharp_pin_raises_to_high_with_version_hint` | ✅ COMPLIANT |
| Wave-2 ecosystem detectors | Java duel resolves gradle-wins | `internal/detect/detect_test.go > TestDetectWave2/java_duel_resolves_gradle-wins_single_evidence + TestDetectWave2Contracts/java_duel_keeps_both_manifests_in_signals` | ✅ COMPLIANT |
| Wave-2 ecosystem detectors | Erlang weak signals stay silent | `internal/detect/detect_test.go > TestDetectWave2/erlang_weak_signals_stay_silent` (fixture `erlang-weak`: app.src+erl+mix.exs → silent) | ✅ COMPLIANT |
| Optional Runtime/Frameworks fields | Additive fields preserve ranking | `internal/detect/detect_test.go > TestDetect` (pre-existing ranked/polyglot suite exercises DetectAll sort with zero-value new fields; `omitempty` + Confidence-only comparator verified in types.go/registry.go) | ✅ COMPLIANT |
| Node Bun runtime and frameworks | Bun coexistence keeps single evidence | `internal/detect/detect_test.go > TestDetectBun/bun_coexistence_keeps_single_evidence_with_packageManager-first` (+ bun-alone, bun.lockb presence-only) | ✅ COMPLIANT |
| Node Bun runtime and frameworks | Framework presence without semver | `internal/detect/detect_test.go > TestDetectFrameworks/next_maps_to_react_and_next_regardless_of_range` (+6 further mapping/root-only cases) | ✅ COMPLIANT |
| Extended verified recommendations | React emits eslint only | `internal/generate/generate_test.go > TestReactEmitsEslintOnly` | ✅ COMPLIANT |
| Extended verified recommendations | Forbidden IDs never emitted | `internal/generate/generate_test.go > TestForbiddenIDsNeverEmitted` (Ruby+Svelte; asserts Shopify.ruby-lsp + svelte.svelte-vscode present, rebornix.Ruby/octref.vetur/JamesBirtles/bare rust-lang.rust absent) | ✅ COMPLIANT |
| Union merge and prose targets | Bigger table merges safely | `internal/generate/generate_test.go > TestUnionIdempotentRerender` (9-ID union exactly-once, re-render bytes-equal) + `TestRerenderIsIdempotent` + `TestMergeUnionArraysKeepUserEntries` | ✅ COMPLIANT |
| Union merge and prose targets | Non-vscode targets use prose | `internal/generate/generate_test.go > TestCursorProseMentionsCSharp` (prose mention, no array; devin/kiro templates carry identical prose conditionals) | ✅ COMPLIANT |

**Compliance summary**: 10/10 scenarios compliant, 5/5 requirements met.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|------------|--------|-------|
| Wave-2 ecosystem detectors | ✅ Implemented | csharp.go (root Glob, pin→High, hints→Low), java.go (gradle-wins, both manifests in Signals), ruby.go (PM=bundler), erlang.go (mix.exs ignored, .app.src/.erl alone silent); registry order node,go,python,rust,csharp,java,ruby,erlang with order comment |
| Optional Runtime/Frameworks fields | ✅ Implemented | `Runtime string json:",omitempty"` + `Frameworks []string json:",omitempty"`; sort comparator Confidence-only, unchanged |
| Node Bun runtime and frameworks | ✅ Implemented | bun.lock/bun.lockb existence-only (never read/parsed), packageManager-field-first else bun, Runtime=bun, single `[]Evidence{ev}` return; root-only presence-only framework scan with meta→base mapping, no semver |
| Extended verified recommendations | ✅ Implemented | Exact casings in map: Vue.volar, Shopify.ruby-lsp, svelte.svelte-vscode, astro-build.astro-vscode, oven.bun-vscode, ms-dotnettools.csharp, redhat.java, pgourlain.erlang; React/Next map to no framework extension (eslint covers); forbidden IDs appear only in comments + negative-test assertions, never in emitted output |
| Union merge and prose targets | ✅ Implemented | Set-dedupe + sort.Strings union; cursor/devin/kiro templates mention IDs in prose only (no `recommendations` key in any non-vscode template); goldens refreshed with zero diff (node-only conditionals false by design) |

Fixtures verified present: csharp, csharp-pin, java-maven, java-gradle, java-duel, ruby, ruby-pin, erlang, erlang-weak, node-bun, node-next. No regressions: all 7 pre-existing suites green.

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| Append-after-rust order | ✅ Yes | Registry appends csharp,java,ruby,erlang after rust; tie order stable |
| Gradle-wins single evidence | ✅ Yes | Single Java evidence, PM=gradle, both manifests in Signals |
| Presence-only frameworks | ✅ Yes | Root package.json deps+devDeps scan, meta→base, no semver/recursion |
| vscode-array-only extensions | ✅ Yes | Array only in vscode target; cursor/devin/kiro prose-only |

### Issues Found

**CRITICAL**: None
**WARNING**:
- Untracked change artifacts in working tree (`design.md`, `proposal.md`, `exploration.md`, `specs/` under `openspec/changes/stackup-wave2/`) — implementation commits (8d9819c, 6fae378, 76ecabc) cover code+tests+tasks only. Commit these files before `sdd-archive` so the change is self-contained.
**SUGGESTION**:
- `TestDetectErlangWithoutRebar3` ran with `rebar3` present in this environment (fallback branch logged but file-signals path decided); the no-rebar3 branch is fixture-logic-identical but was not exercised on a rebar3-less host in this run.

### Verdict

PASS WITH WARNINGS
All 15 tasks complete, 10/10 spec scenarios covered by passing tests, full suite + vet + lint green; one hygiene warning (untracked SDD artifacts to commit before archive).
