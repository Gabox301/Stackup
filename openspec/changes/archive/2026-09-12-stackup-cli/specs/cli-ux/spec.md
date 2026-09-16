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
