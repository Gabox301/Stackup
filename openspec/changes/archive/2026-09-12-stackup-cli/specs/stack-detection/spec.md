# Stack Detection Specification

## Purpose

Detect MVP-4 stacks and return confidence-ranked Evidence that drives generation.

## Requirements

### Requirement: Ordered MVP-4 signal detection

The system MUST run detectors in registry order and MUST recognize Node (`package.json` + lockfiles), Go (`go.mod`/`go.sum`), Python (`pyproject.toml`/`requirements*.txt`/`Pipfile`), Rust (`Cargo.toml`/`Cargo.lock`).

#### Scenario: Polyglot repo returns ranked list

- GIVEN a repo containing `package.json` and `go.mod`
- WHEN detection runs
- THEN both evidences return ranked without single-winner collapse

#### Scenario: Unknown repo with allow-unknown

- GIVEN a repo with no MVP-4 signals
- WHEN detection runs with `--allow-unknown`
- THEN an empty low-confidence list returns with proceed-without-detection guidance

### Requirement: Confidence-graded Evidence ranking

The system MUST grade each evidence high/medium/low and MUST sort descending, with documented grade thresholds.

#### Scenario: Lockfile raises confidence

- GIVEN `package.json` alone grades medium
- WHEN `package-lock.json` is also present
- THEN the Node evidence grades high

#### Scenario: Empty detection stops generation

- GIVEN no signals and no `--allow-unknown`
- WHEN detection runs
- THEN unknown stack is reported and generation stops
