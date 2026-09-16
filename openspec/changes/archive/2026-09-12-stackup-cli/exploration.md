## Exploration: stackup-cli

### Current State

Greenfield. Workdir `C:\Users\gortega\Downloads\Gabo\Stackup` contains only
`openspec/` (config, empty `specs/`, `changes/archive/`) and `.atl/` from
`sdd-init` (Engram id 1752). No `go.mod`, no source, not a git repo.
Toolchain verified in init: Go 1.27.1 + golangci-lint available. There is no
existing behavior to preserve and no coupling to untangle — every constraint
below comes from the problem domain, not from legacy code.

Intent under investigation: a Go CLI that (1) detects a project's stack
across many ecosystems, (2) generates/edits IDE configs for `vscode`,
`cursor`, `devin`, `kiro` (names preserved verbatim), (3) with a TUI-first UX
plus dry-run/preview and non-interactive/CI operation.

### Affected Areas

- `go.mod`, `cmd/stackup/`, `internal/` — do not exist yet; first change must
  scaffold module, standard Go CLI layout, and CI before strict TDD can apply
  (strict_tdd currently false, no-runner fallback).
- `openspec/specs/` — empty; future deltas for detection, config generation,
  TUI/CLI UX will land here via propose/spec.
- `openspec/changes/stackup-cli/` — this file (`exploration.md`) is its first
  artifact.
- No existing code files are affected (nothing to modify or migrate).

### Ecosystem Coverage

Detection signals per ecosystem (manifest = high confidence, lockfile/toolchain
pin = corroborating, version-manager files = weak hints):

| Ecosystem | Primary signals (high) | Corroborating signals | Tier |
|---|---|---|---|
| Node.js | `package.json` | `package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`, `bun.lock*`, `.nvmrc`, `.node-version` | MVP |
| Go | `go.mod` | `go.sum`, `go.work`, `.go-version` | MVP |
| Python | `pyproject.toml`, `requirements*.txt`, `Pipfile`, `setup.py/setup.cfg` | `poetry.lock`, `uv.lock`, `Pipfile.lock`, `.python-version`, `environment.yml` | MVP |
| Rust | `Cargo.toml` | `Cargo.lock`, `rust-toolchain.toml` | MVP |
| Java | `pom.xml`, `build.gradle(.kts)`, `settings.gradle(.kts)` | `gradle.lockfile`, `.java-version`, `.sdkmanrc`, `mvnw`/`gradlew` wrappers | Wave 2 |
| Ruby | `Gemfile` | `Gemfile.lock`, `.ruby-version`, `.tool-versions` | Wave 2 |
| PHP | `composer.json` | `composer.lock`, `.php-version` | Wave 2 |
| .NET | `*.csproj`, `*.fsproj`, `*.sln(x)` | `global.json`, `Directory.Packages.props`, `nuget.config` | Wave 2 |
| Extensible tail | `mix.exs` (Elixir), `pubspec.yaml` (Dart), `Package.swift` (Swift), `build.zig` (Zig), `go.work` hints | — | Plugin |

Recommended MVP set: **Node, Go, Python, Rust**. Rationale: four manifest
formats cover the large majority of target repos, each has a lockfile +
toolchain-pin story to design confidence scoring against, and all four are
verifiable on the maintainer's machine. Java/Ruby/PHP/.NET follow as Wave 2
behind a detector registry so no architecture change is needed to add them.

Extensibility shape (observation for design, not a decision): a `Detector`
interface (`Name()`, `Detect(root) -> Evidence`) behind an ordered registry;
`Evidence` carries ecosystem, confidence (high/medium/low), matched signals,
version hints, and package-manager identification. Polyglot repos (e.g.
Go backend + Node frontend) MUST be representable — output is a ranked list
of detected stacks, not a single winner.

### IDE Config Targets

| IDE | Config paths stackup would write | Notes |
|---|---|---|
| vscode | `.vscode/settings.json`, `.vscode/extensions.json`, `.vscode/launch.json`, `.vscode/tasks.json` | Well-documented, stable schema. |
| cursor | `.vscode/*` (compatible, Cursor is a VS Code fork) + `.cursorrules` (legacy) + `.cursor/rules/*.mdc` (current) + `.cursor/mcp.json` | Dual surface: keep `.vscode` output shared, add Cursor-native rules dir. |
| kiro | `.kiro/steering/*.md`, `.kiro/specs/<name>/`, `.kiro/settings/mcp.json` | **Assumption**: Kiro = AWS Kiro IDE, whose project context lives in `.kiro/`. Needs user confirmation. |
| devin | No standard local config file exists | **Assumption**: Devin = Cognition Devin AI agent, which consumes repo docs rather than a config file. Tentative mapping: repo-level agent guidance (`AGENTS.md`) and/or a `.devin/` knowledge stub defined in propose. **Blocking clarification** — if the user meant a different "devin", this row is wrong. |

Key file-format gotcha: `.vscode/settings.json` is **JSONC** (comments +
trailing commas allowed). Go's `encoding/json` round-trip strips comments and
fails on them. Config generation MUST use a JSONC-tolerant strategy (parse
leniently, preserve verbatim where untouched) — naive marshal/unmarshal will
destroy user comments. Same caution applies to `.cursor/mcp.json`-style files.

Merge-vs-overwrite and safety behaviors to lock in propose/spec:

- Merge strategy (recommended default): deep-merge JSON objects, preserve
  unknown user keys verbatim, union `recommendations`-style arrays
  (dedupe, never delete user entries), timestamped backup before any write.
- Overwrite: only under explicit flag, always with backup.
- Dry-run/preview: `--dry-run` prints a per-file unified diff and writes
  nothing; exit code distinguishes "changes previewed" from errors.
- Non-interactive/CI: `--yes`/`--non-interactive` + `--format json` for
  machine-readable output; auto-detect non-TTY stdin and skip all prompts;
  nonzero exit on undetected stack unless `--allow-unknown`.

### Approaches

1. **Manifest-first rule engine + template/merge writer + Bubble Tea TUI** —
   ordered detectors yield ranked `Evidence`; Go templates render per-IDE
   configs; JSONC-aware merge applies them; Bubble Tea drives selection,
   diff preview, and apply.
   - Pros: deterministic and testable (pure detection fns → table-driven
     tests; `Model.Update()` directly for TUI state; golden files for
     rendered `settings.json`); best TUI test story via `teatest`; Charm
     stack (Bubbles components, Lip Gloss styling, `huh` for forms) is the
     most active Go TUI ecosystem.
   - Cons: largest upfront surface (detector registry + merge semantics +
     TUI); Bubble Tea Elm-architecture learning curve.
   - Effort: Medium/High

2. **Heuristic-glob detector + overwrite-with-backup + tview forms** —
   fast `doublestar`-style globbing for manifest presence; write full files
   with `.bak` safety net; tview/tcell widget forms for the TUI.
   - Pros: fastest path to a visible MVP; tview gives batteries-included
     forms/tables with little boilerplate.
   - Cons: overwrites fight the "preserve verbatim" requirement; tview is
     far harder to unit-test (no `teatest` equivalent, rendering coupled to
     tcell screen); weaker styling/accessibility story; glob-only detection
     misfires on polyglot repos.
   - Effort: Low/Medium

3. **Cobra/Viper CLI-first with optional TUI layer** — ship `detect`,
   `generate`, `diff`, `apply` subcommands (Cobra) + config (Viper) first;
   TUI (either library) as a thin front-end over the same
   `internal/` services; `--dry-run`/`--format json` from day one.
   - Pros: CI mode falls out naturally; forces the testable
     detect/generate/merge core to exist independent of any UI; TUI choice
     stays reversible; matches standard Go CLI layout expectations.
   - Cons: two UX surfaces to keep consistent; slightly slower to the
     "wow, a TUI" demo.
   - Effort: Medium

### Recommendation

Combine 3 as the structural spine with 1 as the TUI direction: build the
deterministic core first (detector registry → ranked evidence → template +
JSONC-aware merge writer → diff/preview/apply services, all UI-agnostic in
`internal/`), expose it through Cobra subcommands with `--dry-run` and
`--format json` from day one, then layer a Bubble Tea (+ Bubbles/Lip Gloss,
`huh` for forms) TUI on top. This keeps the TUI decision reversible, makes CI
mode trivial, and concentrates testability where the go-testing skill demands
it: table-driven detector tests on `t.TempDir()` fixtures, direct
`Model.Update()` tests for TUI state, `teatest` only for interactive flows,
golden files for generated configs.

No final TUI library lock in this phase — evidence favors Bubble Tea but the
decision belongs to propose/design. tview stays a documented fallback if the
team values form-building speed over testability. `devin`/`kiro` mappings
above are explicit assumptions awaiting user confirmation (see Risks).

### Risks

- `devin` mapping may be wrong — no standard Devin local-config format is
  publicly established; if the user meant another tool, the Devin target
  needs re-scoping. Clarify before spec.
- `kiro` mapping assumes AWS Kiro IDE (`.kiro/` layout); confirm, and pin
  which sub-paths (steering vs specs vs MCP settings) are in scope.
- JSONC round-trip destroying user comments in `settings.json` (and
  Cursor/Kiro JSON files) — requires JSONC-aware handling, flagged for design.
- Array-merge semantics (`extensions.json` recommendations, lists in
  settings) need an explicit rule (union-no-delete recommended) or the tool
  will silently drop user entries.
- Polyglot repos: single-winner detection would misreport; ranked-evidence
  output is required, with confidence thresholds defined in spec.
- Scope creep: "broad coverage" across 8+ ecosystems in one change —
  mitigated by MVP-4 + registry, enforce in proposal scope.
- TUI/CI duality: interactive prompts left in a CI path hang pipelines —
  non-TTY auto-detection and `--non-interactive` MUST be spec-level
  requirements, not afterthoughts.
- Windows-first dev machine: path handling and golden files MUST stay
  portable (`filepath`, `t.TempDir()`, no hardcoded separators).

### Ready for Proposal

Yes — with two blocking clarifications for the orchestrator to put to the
user: (1) confirm `devin` = Cognition Devin AI (and accept the
`AGENTS.md`/`.devin/`-stub mapping, or correct it); (2) confirm `kiro` = AWS
Kiro IDE and which `.kiro/` sub-paths are in scope. Everything else
(MVP-4 ecosystems, merge-with-backup + dry-run defaults, Cobra-core +
Bubble Tea-leaning TUI, ranked-evidence output) is sufficiently evidenced to
draft the proposal.
