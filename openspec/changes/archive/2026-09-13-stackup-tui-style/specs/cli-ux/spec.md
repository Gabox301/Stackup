# Delta for cli-ux

## ADDED Requirements

### Requirement: TUI styling caps and token fidelity

The system MUST style TUI screens within fixed caps, MUST wrap whole tokens only, MUST keep text tokens verbatim, and MUST never signal by color alone.

| Cap | Limit |
|-----|-------|
| Border sets | 1 rounded set |
| Confidence badges | 3 (high / medium / low) |
| Pills | 2 (`(new)` / `(overwrite)` family) |
| Spacing scale | 0 / 1 / 2 only |
| Viewport `+`/`-` tint | Prohibited in v1 |
| New styling-library major (lipgloss v2 scope) | Prohibited in v1 |

#### Scenario: styled-tty-shows

- GIVEN full-color TTY with TUI gate passed
- WHEN any screen renders
- THEN badges, pills, borders, and footer emphasis present
- AND exits and write gating stay unchanged

#### Scenario: tokens-verbatim-when-stripped

- GIVEN styled output containing `node: high`, `exit 2`, `(new)` / `(overwrite)`
- WHEN ANSI sequences are stripped
- THEN each token appears verbatim and contiguous
- AND meaning survives without color

### Requirement: Style degradation and golden stability

The system MUST degrade by environment without forcing a color profile, and ANSI-stripped output MUST equal current plain text.

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
