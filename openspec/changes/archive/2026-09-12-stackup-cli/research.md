# SDD Research — stackup-cli (`gentle-ai.sdd-research/v1`)

revision: 2
outcome: blocked
change: stackup-cli
project: stackup

## Retained selected request (pre-write intent, recorded before source access)

Selected research lanes retained verbatim from confirmed handoff
(`sdd/stackup-cli/pending` #1756; exploration `sdd/stackup-cli/explore` #1755):

1. devin lane — Establish Devin Desktop local project config paths (rules /
   memories / MCP equivalent, legacy `.windsurf/*` mapping, global vs project
   scope), file formats (JSON / JSONC / Markdown), merge-safe behaviors.
   Prior Cognition-assumption is SUPERSEDED per handoff.
2. kiro lane — Pin AWS Kiro IDE FULL scope exact sub-paths and formats
   (`.kiro/steering`, `.kiro/specs`, settings / `mcp.json`).
3. cursor lane — Validate `.vscode`-compat plus `.cursor/rules/*.mdc` plus
   `.cursor/mcp.json`.
4. JSONC-in-Go lane — How to parse and merge `.vscode/settings.json` plus
   cursor / kiro JSON files WITHOUT destroying user comments or trailing
   commas (libraries, round-trip strategy, what to forbid).
5. TUI lane — Bubble Tea (+ Bubbles / Lip Gloss / huh) vs tview evidence for
   testability (teatest, direct `Model.Update`), forms, styling; reversible
   recommendation. Evaluated through the go-testing skill lens
   (table-driven tests, `t.TempDir()`, direct `Model.Update`, teatest only
   for interactive flows, golden files via `-update`).

Canonical desired content retained: source-backed auditable evidence for the
five lanes above, with each validated claim mapped to source IDs, plus
contradictions, uncertainty, and freshness. No proposal, spec, design, or
tasks content.

## Requested source classes

- `documentation`
- `open-web`

## Admission (`gentle-ai.sdd-research-capability/v1`)

- Launch-declared grants (RETRY revision 2 launch): `documentation=[context7_resolve-library-id, context7_query-docs]`; `open-web=[webfetch, websearch]`.
- Session effective evidence grants (authoritative): `documentation=[]`; `open-web=[]`.
- Observed exact grants: none for `documentation`; none for `open-web`.
  Expected-but-unobservable grant strings: `webfetch`, `websearch` (neither
  is present in this runtime's tool list, so no `open-web` evidence access
  was possible). Function names resembling `documentation` tooling are not
  counted: persistence tools and inherited unnamed tools are not evidence
  grants, and no exact declared grant for a requested class was satisfied.
- Persistence-tool access is NOT an evidence grant and was not counted.
- Capability `gentle-ai.sdd-research-capability/v1` with exact declared
  grants for a requested class: NOT SATISFIED for any requested class.
- Verdict: ADMISSION DENIED. Source access was not attempted. No fetch,
  no documentation lookup, no open-web access was performed under this phase.

## Sources

None. Admission denial admits no sources.

No `id, class, title, publisher, URL, accessed_at, excerpt` entries exist
for this revision. Prior handoff references (exploration text, pending
handoff) are lineage pointers only and are NOT admitted as research sources.

## Validated claims

None. This `blocked` revision emits no validated claims and no
source-to-claim mappings. Every question in the Retained selected request
section above remains unsupported.

## Contradictions

None recorded. With zero admitted sources there is nothing to contradict
and no contradiction to resolve.

## Uncertainty

Total. All five lanes (devin, kiro, cursor, JSONC-in-Go, TUI) are
unanswered in this revision. No confidence level can be assigned to any
lane outcome.

## Freshness

Not applicable. No source was accessed, so no `accessed_at` timestamp and
no version pin exists for this revision.

## Product choices (non-authoritative, NOT evidence)

Retained for orchestrator recovery only. These are NOT validated claims
and MUST NOT be treated as research findings:

- stackup remains a greenfield Go CLI per exploration #1755
  (MVP-4 detector registry scope; TUI-first plus Cobra-core spine;
  `--dry-run` plus `--format json` plus non-TTY from day one).
- devin identity correction retained from pending handoff #1756
  (Devin Desktop, ex-Windsurf; Cognition-assumption superseded).
- kiro FULL scope retained from pending handoff #1756.
- Research selection retained as mandatory before propose.

## Readiness

`proposal_ready: false`. Selected research is NOT `done`. Per the closed
readiness matrix, `blocked` outcome in any store mode is not ready,
and hybrid mode additionally requires equal revision and bytes on
readback, which cannot confer readiness on a `blocked` outcome.

## Recovery

Requires explicit re-entry with exact positive grants for the requested
classes (`documentation` and/or `open-web` via
`gentle-ai.sdd-research-capability/v1`), issued as a new launch with
runtime-observable tooling for every declared grant string. Retained
intent above restores the request; a matching restart restores evidence
references. Never invent state from either surviving store.
