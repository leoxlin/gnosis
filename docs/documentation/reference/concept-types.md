# Concept types

Eleven bundled types. Each definition lives in `docs/concepts/` (overridable in any vault) and declares a `path` where its instances belong. List them with `gnosis get concepts`.

## Intent types

| Type | Path | Purpose | Key fields |
|---|---|---|---|
| Purpose | `purpose.md` (singleton) | Enduring outcomes and boundaries | — |
| Decision | `decisions/` | Durable non-obvious choices | `supersedes` |
| Directive | `directives/` | Executable implementation handoffs | `status` (draft/open/blocked/done), checkbox tasks, deltas, scenarios |
| Procedure | `procedures/` | Agent-executable contracts | `description`, `tags`, `invocation` |

## Content types

| Type | Path | Answers | Key fields |
|---|---|---|---|
| Concept | `concepts/` | What is true? | `status`, `confidence`, `source`, `tier`, `superseded_by` |
| Entity | `entities/` | Who is involved? | `kind`, `status` |
| Resource | `resources/` | Where is something? What can the agent use? | `kind`, `resource`, `status` |
| Event | `events/` | What happened? What was observed? | `occurred_at`, `actor`, `source`, `status` |
| Memory | `memories/` | Durable scoped facts, preferences, observations | `scope`, `observed_at`, `hash`, `entities`, `status` |
| Reflection | `reflections/` | What lesson was learned? | `status`, `confidence`, `superseded_by` |
| Policy | `policies/` | What should or must be done? When does it apply? | `status`, `applies_to`, `superseded_by` |

## Shared provenance metadata

Optional on content types: `status`, `confidence` (0.0–1.0), `source`, `observed_at`, `valid_from`, `superseded_by`, `tier` (core/supporting/peripheral), `entities`. Unknown frontmatter is always preserved verbatim.

## Body conventions

Unmarked claims are extracted from sources; `^[inferred]` marks agent generalizations; `^[ambiguous]` marks unresolved disagreement. Typed `relationships:` frontmatter adds semantic edges (`extends`, `implements`, `uses`, `contradicts`, `derived_from`, `causes`, `depends_on`, `owns`, `related_to`).
