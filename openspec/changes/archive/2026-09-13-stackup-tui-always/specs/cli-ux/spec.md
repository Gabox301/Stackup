# Delta for cli-ux

## ADDED Requirements

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
- AND only confirm returns ActionApply

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

On gate exclusion or launch failure the system MUST print byte-identical text plus a stderr warning, MUST preserve the engine exit (never coerce to 1), and presence MUST NEVER gate writes — only `apply` confirm returns ActionApply.

#### Scenario: launch-failure-fallback

- GIVEN a qualifying TTY run where TUI launch fails
- WHEN the command runs
- THEN stdout text is byte-identical to plain mode
- AND a stderr warning prints with the engine exit preserved
