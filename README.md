# stackup

Detect your project's stack and generate IDE configs for it — safely.

> 🌐 [Leer en español](README.es.md) — English is the canonical version; translations are best-effort.

## English

`stackup` scans a project root, detects its ecosystems with confidence-ranked
evidence, and renders editor configuration for VS Code, Cursor, Devin Desktop,
and Kiro. Writes are previewable (`--dry-run`), backed up before every change,
and CI-safe (machine-readable output, never blocks on prompts).

### Install

Requires Go 1.27.1+.

```sh
go install ./cmd/stackup
# or build locally
go build -o stackup ./cmd/stackup
```

#### Prebuilt binaries

Download the archive matching your OS and architecture from
[GitHub Releases](https://github.com/Gabox301/Stackup/releases):

| OS | Architecture | Example file |
| --- | --- | --- |
| Windows | x86_64 (amd64) | `stackup_0.1.0_Windows_x86_64.zip` |
| Linux | x86_64 (amd64) | `stackup_0.1.0_Linux_x86_64.tar.gz` |
| Linux | arm64 | `stackup_0.1.0_Linux_arm64.tar.gz` |
| macOS | x86_64 (amd64) | `stackup_0.1.0_Darwin_x86_64.tar.gz` |
| macOS | arm64 | `stackup_0.1.0_Darwin_arm64.tar.gz` |

Or install the latest release with Go (requires Go 1.27.1+):

```sh
go install github.com/Gabox301/Stackup/cmd/stackup@latest
```

### Quickstart

```sh
# What stacks are in this repo?
stackup detect --path .

# Preview what would be generated (writes nothing)
stackup generate --dry-run --path .

# Show the unified diff against disk
stackup diff --path .

# Apply (backs up every overwritten file as <f>.bak.<timestamp>)
stackup apply --path .
```

### Try it locally on Windows

Run from your project directory in **Windows Terminal** (or any ANSI-capable
console). Build the binary, then open the interactive TUI:

```powershell
go build -o stackup.exe ./cmd/stackup
.\stackup.exe detect --path .
```

The TUI screens — rounded panels, confidence badges, and the selected-row
highlight — only present on a real TTY:

- Run `.\stackup.exe detect .` directly; do **not** pipe or redirect stdout.
- Use Windows Terminal rather than the legacy console so colors and
  box-drawing characters render (legacy consoles intentionally fall back to
  plain ASCII).
- Add `--yes` / `--non-interactive`, or `--format json`, to force plain or
  machine output (this also happens automatically when stdout is piped).
- Keys: `q` quits any screen, `up/down` / `j/k` move the selection, and in
  `apply` `y` applies / `n` aborts without writing.

Preview everything before writing anything:

```powershell
.\stackup.exe generate --dry-run --path .   # preview only (exit 2 if changes)
.\stackup.exe diff --path .                 # unified diff, writes nothing
.\stackup.exe apply --yes --path .          # safe write; backs up first
```

The repository's own `testdata/node` project is a ready-made trial target:
`.\stackup.exe detect --path testdata/node`.

### Commands

| Command    | What it does                                             |
| ---------- | -------------------------------------------------------- |
| `detect`   | Print ranked stack evidence for a project root           |
| `generate` | Render IDE configs (use `--dry-run` to preview only)     |
| `diff`     | Show unified diff of planned changes, writes nothing     |
| `apply`    | Write configs with timestamped backups before overwrites |

#### Which commands write to disk?

- Read-only (never write): `detect`, `diff`, and any command with `--dry-run`.
- Writes: `generate` (without `--dry-run`) and `apply`.
- `apply` is the command you want: it creates missing files directly and,
  before overwriting anything existing, saves a `<file>.bak.<UTC-timestamp>`
  backup. Overwrites need `--force` (or a TTY confirmation); in CI use
  `--yes` / `--non-interactive` (blocked overwrites exit `3` instead of
  prompting).

```sh
stackup apply --path <project-dir>          # interactive (asks before overwriting)
stackup apply --yes --path <project-dir>    # non-interactive
stackup apply --force --path <project-dir>  # also overwrite existing files
```

#### Flags

```
--path <dir>            project root (default: .)
--ide <list>            vscode,cursor,devin,kiro (default: all)
--format <text|json>    human or machine-readable output
--dry-run               preview only, writes nothing
--force                 allow overwriting existing files (backup still taken)
--yes, --non-interactive  skip all prompts (CI mode)
--allow-unknown         proceed even when no stack is detected
```

#### Exit codes

| Code | Meaning                         |
| ---- | ------------------------------- |
| 0    | ok / no changes / user aborted  |
| 2    | preview has changes             |
| 3    | blocked in non-interactive mode |
| 4    | unknown stack                   |
| 1    | error                           |

### Detected stacks

| Ecosystem          | Signals (manifest → lock/pin → hint)                               | IDE extension recommended  |
| ------------------ | ------------------------------------------------------------------ | -------------------------- |
| Node.js            | `package.json` → lockfiles / `packageManager` / Bun (`bun.lock`)   | `dbaeumer.vscode-eslint`   |
| Bun                | `bun.lock` / `packageManager: bun` (runtime on Node evidence)      | `oven.bun-vscode`          |
| React / Next.js    | `react`, `next` in dependencies                                    | covered by ESLint          |
| Vue / Nuxt         | `vue`, `nuxt`                                                      | `Vue.volar`                |
| Svelte / SvelteKit | `svelte`, `sveltekit` / `@sveltejs/kit`                            | `svelte.svelte-vscode`     |
| Astro              | `astro`                                                            | `astro-build.astro-vscode` |
| Go                 | `go.mod` → `go.sum`                                                | `golang.go`                |
| Python             | `pyproject.toml` / `requirements*.txt` / `Pipfile` → lockfiles     | `ms-python.python`         |
| Rust               | `Cargo.toml` → `Cargo.lock`                                        | `rust-lang.rust-analyzer`  |
| C# (.NET)          | `*.csproj` / `*.sln` → `global.json` / `packages.lock.json`        | `ms-dotnettools.csharp`    |
| Java               | `pom.xml` / `build.gradle` (Gradle wins on duel) → wrappers / pins | `redhat.java`              |
| Ruby               | `Gemfile` → `Gemfile.lock` (Bundler)                               | `Shopify.ruby-lsp`         |
| Erlang             | `rebar.config` → `rebar.lock` (rebar3)                             | `pgourlain.erlang`         |

Detection is root-only and polyglot-safe: a Go backend + Node frontend
returns both evidences ranked by confidence, never a single winner.

### IDE targets

| IDE           | Files written                                                                 |
| ------------- | ----------------------------------------------------------------------------- |
| VS Code       | `.vscode/{settings,extensions,launch,tasks}.json` (JSONC, comments preserved) |
| Cursor        | `.cursor/rules/*.mdc` + `.cursor/mcp.json`                                    |
| Devin Desktop | `.devin/rules/*.md` (+ `.windsurf` fallback) + `.devin/mcp_config.json`       |
| Kiro          | `.kiro/steering/*.md` + `.kiro/specs/<feature>/` + `.kiro/settings/mcp.json`  |

Extension recommendations go to the VS Code `recommendations` array
(union merge — your entries are never deleted); other IDEs get prose
mentions. Re-renders are byte-idempotent.

### Safety model

- `--dry-run` / `diff` never touch disk.
- Every overwrite is preceded by a `<file>.bak.<UTC-timestamp>` backup;
  writing without a backup is impossible by construction.
- Overwrites require `--force` (or an explicit TTY confirmation in `apply`).
- Non-TTY runs skip prompts and exit `3` instead of hanging a pipeline.

### Development

```sh
go build ./...
go test ./... -count=1
gofmt -l . && go vet ./...
golangci-lint run ./...
```

Layout: `cmd/stackup/` (Cobra tree) + `internal/{detect,generate,merge,diff,apply,tui}/`
(UI-agnostic pipeline; the Bubble Tea TUI sits behind a replaceable
`Launcher` interface with a stub for tests).
