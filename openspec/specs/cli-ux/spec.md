# CLI UX Specification

## Purpose

Expose a CI-safe Cobra core with preview-before-write and a replaceable TUI.

## Requirements

### Requirement: Core commands with preview and machine output

The system MUST provide `detect`, `generate`, `diff`, `apply`. `diff` and any `--dry-run` run MUST print a unified diff, write nothing, and exit with a distinct preview status. `--format json` MUST emit machine-readable output.

#### Scenario: Dry run writes nothing

- GIVEN existing configs that would change
- WHEN `generate --dry-run` runs
- THEN a unified diff prints and the filesystem is unchanged

#### Scenario: JSON output for CI

- GIVEN any core command run with `--format json`
- WHEN it completes
- THEN stdout is parseable JSON carrying the ranked evidence or diff

### Requirement: Non-interactive safety with replaceable TUI

The system MUST skip prompts under `--yes`/`--non-interactive` or non-TTY, MUST never block waiting for input, and MUST keep the TUI behind a replaceable interface with no library lock.

#### Scenario: CI run skips prompts

- GIVEN non-TTY stdin with pending overwrites and no `--yes`
- WHEN `apply` runs
- THEN it skips prompts, writes nothing, and exits with a distinct status

#### Scenario: TUI swap keeps core green

- GIVEN the TUI implementation replaced by a stub
- WHEN core commands run
- THEN all command behaviors remain unchanged

#### Scenario: TTY decline aborts with success status

- GIVEN a TTY session with pending overwrites and no `--yes`
- WHEN the user declines the confirmation (EOF or anything-but-yes)
- THEN the system MUST abort, write nothing, print `apply aborted (no writes performed)`, and exit 0 (an explicit user abort is neither ok-with-writes nor an error nor a non-interactive block)

#### Scenario: Fresh files apply without confirmation

- GIVEN pending changes that are all fresh files (no overwrites)
- WHEN `apply` runs, including under `--non-interactive`
- THEN the system MUST write the fresh files without confirmation; the confirmation gate and the non-interactive block (exit 3) apply only to pending overwrites (`Changed && !IsNew`), so `--non-interactive` is not a global write blocker
### Requirement: TUI presence per command

The system MUST show a TUI screen for each command on qualifying runs. Content MUST equal current text semantics; exits MUST stay 0/1/2/3/4.

| Command | Screen | Content | On quit |
|---------|--------|---------|---------|
| `detect` | Evidence | Same rows as text; empty mirrors exit-4 notice | Keep 0/4 |
| `generate` | Plan preview | File list with `(new)`/`(overwrite)` | Return, then write |
| `diff` | Unified view | Selector + viewport; footer states exit-2 | Keep 2/0 |
| `apply` | Result | Confirm verbatim + written summary; no-change screen | Semantics kept |

#### Scenario: TTY-shows

- GIVEN TTY stdout+stdin, text format, no `--yes`/`--non-interactive`
- WHEN `detect`, `generate`, `diff`, or `apply` runs
- THEN the matching screen presents with equivalent content
- AND the exit matches the non-TUI run

#### Scenario: apply-confirm-unchanged

- GIVEN TTY `apply` with pending overwrites and no `--yes`
- WHEN the user confirms or declines
- THEN wording, writes, and exits match current behavior
- AND only an explicit write answer (apply confirm, generate plan gate) returns ActionApply

### Requirement: TUI gate rule

The system MUST present iff `stdoutIsTTY && stdinIsTTY && !yes && !non-interactive && format==text`. `--format json` MUST NEVER present, even on TTY.

#### Scenario: piped-stays-text

- GIVEN piped stdout (e.g. `diff | head`) with TTY stdin
- WHEN `diff` runs
- THEN output stays plain text, byte-identical
- AND exit is preserved

#### Scenario: json-never

- GIVEN TTY stdout+stdin with `--format json`
- WHEN any command runs
- THEN output is parseable JSON with no presentation
- AND bytes match the non-TTY run

#### Scenario: yes-flag-stays-text

- GIVEN TTY stdout+stdin with `--yes` (or `--non-interactive`)
- WHEN any command runs
- THEN output stays plain text, byte-identical
- AND `apply` overwrite rules stay unchanged

### Requirement: Fallback and write gating

On gate exclusion or launch failure the system MUST print byte-identical text plus a stderr warning, MUST preserve the engine exit (never coerce to 1), and presence MUST NEVER gate writes — only an explicit write answer returns ActionApply.

#### Scenario: launch-failure-fallback

- GIVEN a qualifying TTY run where TUI launch fails
- WHEN the command runs
- THEN stdout text is byte-identical to plain mode
- AND a stderr warning prints with the engine exit preserved
### Requirement: TUI styling caps and token fidelity

The system MUST style TUI screens within fixed caps, MUST wrap whole tokens only, MUST keep text tokens verbatim, and MUST never signal by color alone.

| Cap | Limit |
|-----|-------|
| Border sets | 1 rounded set |
| Panels | Reuse the single rounded set as a screen frame and inner panes |
| Startup banner | 1 ASCII header (`Banner`), 6 lines, runtime-only |
| Confidence badges | 3 (high / medium / low) |
| Pills | 2 (`(new)` / `(overwrite)` family) |
| Section labels | 1 per screen, grey, styled as a whole line |
| Selected-row highlight | 1 subtle adaptive background; the `>` marker keeps meaning |
| Spacing scale | 0 / 1 / 2 only |
| Viewport `+`/`-` tint | Prohibited in v1 |
| New styling-library major (lipgloss v2 scope) | Prohibited in v1 |

#### Scenario: styled-tty-shows

- GIVEN full-color TTY with TUI gate passed
- WHEN any screen renders
- THEN badges, pills, panels, the startup banner, section labels, the selected-row tint, and footer emphasis present
- AND exits and write gating stay unchanged

#### Scenario: tokens-verbatim-when-stripped

- GIVEN styled output containing `node: high`, `exit 2`, `(new)` / `(overwrite)`
- WHEN ANSI sequences are stripped
- THEN each token appears verbatim and contiguous
- AND meaning survives without color

### Requirement: Style degradation and golden stability

The system MUST degrade by environment without forcing a color profile, and ANSI-stripped output MUST equal current plain text once border chrome and panel padding are discounted (chrome is runtime-only decoration that never carries content tokens).

| Condition | Result |
|-----------|--------|
| `NO_COLOR` / `CLICOLOR=0` / non-TTY | No color |
| `TERM=dumb` (Unix) | Degraded style |
| Legacy Windows console | Ascii fallback, no raw escapes |
| Full-color TTY | Full style |

#### Scenario: no-color-degrades

- GIVEN `NO_COLOR` set (or `CLICOLOR=0`, non-TTY, or `TERM=dumb`)
- WHEN any screen renders
- THEN output degrades per matrix with tokens intact
- AND no color profile is forced

#### Scenario: legacy-windows-ascii

- GIVEN legacy Windows console without ANSI support
- WHEN any screen renders
- THEN output falls back to Ascii with no raw escapes
- AND layout and tokens stay intact

#### Scenario: palette-change-keeps-goldens

- GIVEN a palette-only style tweak with no text change
- WHEN ANSI sequences are stripped from the view
- THEN bytes are identical to the current golden
- AND color-only diffs never break goldens
