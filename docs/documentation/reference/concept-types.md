# Concept Types

A Concept Type is a Markdown schema for one category of knowledge. Its
definition declares the instance path, required metadata, and body contract.
Run `gnosis get concepts` to list the effective definitions and `gnosis get
concepts <Type>` to list records of one exact type.

gnosis includes twelve definitions:

| Type | Instance path | Purpose | Characteristic fields |
|---|---|---|---|
| `Concept` | `concepts/` | Factual or semantic knowledge | `status`, `confidence`, `source`, `tier`, `superseded_by` |
| `Entity` | `entities/` | People, organizations, systems, and other actors | `kind`, `status` |
| `Event` | `events/` | Something observed or occurring at a time | `occurred_at`, `actor`, `source`, `status` |
| `Memory` | `memories/` | Durable, scoped facts, preferences, or observations | `scope`, `observed_at`, `hash`, `status` |
| `OpenSpecDesign` | `openspec/` | Technical decisions and trade-offs for a change | required design sections |
| `OpenSpecProposal` | `openspec/` | Motivation, scope, capabilities, and impact | required proposal sections |
| `OpenSpecSpec` | `openspec/` | Normative requirements and scenarios | `SHALL`/`MUST`, `WHEN`, `THEN` |
| `OpenSpecTasks` | `openspec/` | Ordered implementation checklists | numbered task checkboxes |
| `Policy` | `policies/` | Conditional or normative rules | `status`, `applies_to`, `superseded_by` |
| `Procedure` | `procedures/` | Repeatable execution contracts | `description`, `tags`, `invocation` |
| `Reflection` | `reflections/` | Lessons derived from experience | `status`, `confidence`, `superseded_by` |
| `Resource` | `resources/` | A usable or addressable resource | `kind`, `resource`, `status` |

Memory pages created by the memory service also include `user_id`, `agent_id`,
`created_at`, `updated_at`, and optional `metadata`.

## Shared metadata

Content types can use `status`, `confidence`, `source`, `observed_at`,
`valid_from`, `superseded_by`, `tier`, and `entities` when their definitions
allow them. Confidence ranges from 0.0 to 1.0. Tier values are `core`,
`supporting`, and `peripheral`.

Unknown frontmatter fields are preserved when a page is read and written.

## Body and relationships

Unmarked claims represent extracted or directly recorded knowledge.
`^[inferred]` marks a generalization; `^[ambiguous]` marks unresolved source
disagreement.

The `relationships` frontmatter field creates typed graph edges. Supported
relationship vocabulary includes `extends`, `implements`, `uses`,
`contradicts`, `derived_from`, `causes`, `depends_on`, `owns`, and
`related_to`.

The four OpenSpec types describe artifacts projected from the registered
OpenSpec store. OpenSpec owns their authoring lifecycle. Vaults can refine
included definitions locally or create additional types.
