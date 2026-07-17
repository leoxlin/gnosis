---
type: Directive
title: Add the knowledge concept types
description: Bundle seven new Concept Type records so gnosis covers every knowledge use case in the knowledge-research taxonomy.
status: done
---

# Goal

Bundle seven new Concept Type records — Concept, Entity, Resource, Event, Memory, Reflection, and Policy — with exact lifecycles and the shared provenance metadata standard, so gnosis covers every knowledge use case in the knowledge-research taxonomy (knowledge-research.pdf, repository root) as mapped by [Adopt the knowledge-type ontology](../decisions/adopt-knowledge-type-ontology.md).

# Scope

- Create exactly seven files under `docs/concepts/`: `concept.md`, `entity.md`, `resource.md`, `event.md`, `memory.md`, `reflection.md`, `policy.md`, with the complete contents embedded in Task 1.
- No Go code changes: the bundle glob `concepts/*.md` in `docs/embed.go`, the write path, search, graph, and validation already serve new types generically.
- No concept instances are created in this directive; only the type definitions ship.

# Global constraints

- Follow [Adopt the knowledge-type ontology](../decisions/adopt-knowledge-type-ontology.md) and [Adopt scoped agent memory as explicit vault records](../decisions/adopt-scoped-agent-memory.md).
- Records must pass vault validation with zero errors and zero warnings (title TypeName convention, non-empty description, non-empty body).
- Match the structure of the existing bundled definitions in `docs/concepts/decision.md` and `docs/concepts/directive.md` (intro, `## Use this for`, `## Minimum record`, `## Lifecycle`, `## Schema`).

# Dependencies

None.

# Implementation plan

### Task 1: Author and apply the seven Concept Type records

**Load:** `docs/concepts/decision.md`, `docs/concepts/directive.md` (style), `docs/decisions/adopt-knowledge-type-ontology.md` (governing decision).
**Files:** create `docs/concepts/concept.md`, `docs/concepts/entity.md`, `docs/concepts/resource.md`, `docs/concepts/event.md`, `docs/concepts/memory.md`, `docs/concepts/reflection.md`, `docs/concepts/policy.md`.
**Interfaces:** consumes nothing; produces seven Concept Type records applied through `gnosis apply page`.

- [x] Write each file below exactly, then apply it with `gnosis apply page gnosis://local/concepts/<name>.md --filename <file>`; expect `changed: true` on first apply and `changed: false` on repeat.

`docs/concepts/concept.md`:

````markdown
---
type: ConceptType
title: Concept
description: A durable semantic or factual concept.
path: concepts
---

# Concept

A **Concept** preserves what is true: a definition, fact, mechanism, or synthesized understanding that outlives its sources.

By convention, Concept records live at `gnosis://<vault>/concepts/`, alongside Concept Type definitions.

## Use this for

- Technical facts, domain concepts, synthesized explanations, and company or project knowledge that answers "what is true?".

Do not use it for events (Event), lessons (Reflection), rules (Policy), agents or people (Entity), or tools and services (Resource).

## Minimum record

- `# Concept` with the self-contained definition.
- Optional `# Why it matters` and `# Sources`. Synthesized claims carry the inline markers `^[inferred]` or `^[ambiguous]`.

## Lifecycle

- Identity is the concept itself, not its title. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `draft` → `reviewed` → `verified`; `disputed` records a contradiction until resolved; `archived` with a `superseded_by` link replaces deletion.
- Update understanding in place as knowledge grows, preserving unknown metadata; record the change in the nearest `log.md` when `vault_log` is enabled.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Concept
title: <name>
description: <one-line summary>
status: <draft | reviewed | verified | disputed | archived>
confidence: <optional 0.0-1.0>
source: <optional origin of the claim>
valid_from: <optional ISO date>
tier: <optional core | supporting | peripheral>
superseded_by: <optional successor link>
---
```
````

`docs/concepts/entity.md`:

````markdown
---
type: ConceptType
title: Entity
description: A person, team, agent, or organization that knowledge is about.
path: entities
---

# Entity

An **Entity** preserves who is involved: people, teams, agents, and organizations, with their roles and relationships.

By convention, Entity records live at `gnosis://<vault>/entities/`.

## Use this for

- Ownership, authorship, trust relationships, team structure, and agent identity — knowledge that answers "who is involved?".

Do not use it for tools or services (Resource), preferences (Memory with `scope: user`), or events (Event).

## Minimum record

- `kind` frontmatter and `# Entity` describing who this is.
- Optional `# Roles` and typed `relationships` (`owns`, `maintains`, `member_of`, `trusts`).

## Lifecycle

- Identity is the real-world entity. Query for an existing page before creating one; reject duplicate identity.
- `status` is `active` while the entity is relevant and `archived` when it leaves scope; archived pages are retained, not deleted.
- Update roles and relationships in place as they change, preserving unknown metadata.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Entity
title: <name>
description: <who this is>
kind: <person | team | agent | organization>
status: <active | archived>
source: <optional origin of the claim>
---
```
````

`docs/concepts/resource.md`:

````markdown
---
type: ConceptType
title: Resource
description: A tool, repository, API, service, schema, or dashboard an agent can use or cite.
path: resources
---

# Resource

A **Resource** preserves where something is and how to reach it: repositories, APIs, services, MCP tools, schemas, and dashboards.

By convention, Resource records live at `gnosis://<vault>/resources/`.

## Use this for

- Locations, endpoints, capabilities, and usage constraints of external or internal systems — knowledge that answers "where is something?" or "what can the agent use?".

Do not use it for people or teams (Entity), policies about usage (Policy), or observations of behavior (Event).

## Minimum record

- `kind` frontmatter, `resource` locator, and `# Resource` describing what it provides.
- Optional `# Usage` with exact commands, endpoints, or schemas, and typed `relationships` (`depends_on`, `owned_by`, `deployed_to`).

## Lifecycle

- Identity is the addressed system. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `active` → `deprecated` → `archived`; archived pages are retained, not deleted.
- Update locators and usage in place when they drift, preserving unknown metadata; stale locators are validation-relevant facts, fix them when observed.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Resource
title: <name>
description: <what it provides>
kind: <repository | api | service | tool | dashboard | schema>
resource: <URL or address>
status: <active | deprecated | archived>
---
```
````

`docs/concepts/event.md`:

````markdown
---
type: ConceptType
title: Event
description: A dated episode, action, or observation worth remembering.
path: events
---

# Event

An **Event** preserves what happened: incidents, actions, tool executions, conversations of record, and direct observations, each anchored to a time.

By convention, Event records live at `gnosis://<vault>/events/`.

## Use this for

- Episodic and perceptual knowledge — incidents, deployments, decisions enacted, experiments, and observations that answer "what happened?" or "what was observed?".

Do not use it for durable facts (Concept), distilled lessons (Reflection), or agent working state, which never enters the vault.

## Minimum record

- `occurred_at` frontmatter and `# Event` describing what happened.
- Optional `# Context` and `# Outcome`; link causes and effects with typed `relationships` (`causes`, `caused_by`, `resolved_by`).

## Lifecycle

- Identity is the episode at its time. Events are append-only: correct a record by creating a new Event and setting `superseded_by` on the prior one, never by rewriting history.
- `status` is `recorded` on creation, `verified` once independently confirmed, and `disputed` while accounts conflict.
- Reflect on clusters of Events into Reflection records through the owning procedure instead of editing events into lessons.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Event
title: <what happened>
description: <one-line summary>
occurred_at: <ISO 8601 timestamp>
actor: <optional who acted>
source: <optional where observed>
status: <recorded | verified | disputed>
superseded_by: <optional corrective event link>
---
```
````

`docs/concepts/memory.md`:

````markdown
---
type: ConceptType
title: Memory
description: A scoped, self-contained agent memory of a durable fact, preference, or observation.
path: memories
---

# Memory

A **Memory** preserves one self-contained durable fact, preference, or observation under an explicit scope, written only through the remember procedure and read through the recall procedure.

By convention, Memory records live at `gnosis://<vault>/memories/`.

## Use this for

- User preferences and persona facts (`scope: user`), agent capabilities and learned limitations (`scope: agent`), and session- or run-scoped durable observations (`scope: session | run`).

Do not use it for conversation transcripts, working state, or knowledge with its own type: facts (Concept), lessons (Reflection), rules (Policy), episodes (Event).

## Minimum record

- `scope`, `observed_at`, and `hash` frontmatter, plus `# Memory` with one self-contained statement using absolute dates and verbatim proper nouns.
- Optional `actor`, `source`, and `entities` (named entities, used for retrieval boosts).

## Lifecycle

- Creation, update, and archival go through the `remember` vault procedure, which reconciles each candidate against the nearest existing memories as ADD, UPDATE, DELETE, or NONE; retrieval goes through the `recall` vault procedure. (Both ship with the agent-memory directive; link them here when they land.)
- `status` is `active` while current and `archived` when superseded or deleted; archived memories are retained for audit and negative knowledge, never silently removed.
- `hash` is the SHA-256 hex of the `# Memory` statement text; exact duplicates are never written.
- Delete only through the remember procedure's DELETE operation, which archives; physical removal requires explicit author approval after tracing inbound links.

## Schema

```yaml
---
type: Memory
title: <short label>
description: <one-line summary>
scope: <user | agent | session | run>
actor: <optional who stated it>
source: <optional where observed>
observed_at: <ISO 8601 date>
hash: <SHA-256 hex of the statement>
entities: [<optional named entities>]
status: <active | archived>
---
```
````

`docs/concepts/reflection.md`:

````markdown
---
type: ConceptType
title: Reflection
description: A distilled lesson, heuristic, or failure pattern learned from experience.
path: reflections
---

# Reflection

A **Reflection** preserves what was learned: a reusable heuristic, failure pattern, or strategy distilled from events, memories, and outcomes.

By convention, Reflection records live at `gnosis://<vault>/reflections/`.

## Use this for

- Lessons that answer "what lesson was learned?" — reusable guidance grounded in recorded experience.

Do not use it for raw episodes (Event), durable facts (Concept), or binding rules (Policy).

## Minimum record

- `# Reflection` with the lesson as one actionable statement.
- `# Evidence` linking the events, memories, or pages the lesson is distilled from, and `# Application` describing when it applies.

## Lifecycle

- Identity is the lesson. Query for an existing page before creating one; merge new evidence into the existing record instead of duplicating it.
- `status` follows `draft` → `established` as evidence accumulates, then `retired` with a `superseded_by` link when the lesson stops holding; retired pages are retained.
- Strengthen or qualify `# Application` in place as counterevidence arrives; record contradictions explicitly rather than deleting them.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Reflection
title: <the lesson>
description: <one-line summary>
status: <draft | established | retired>
confidence: <optional 0.0-1.0>
superseded_by: <optional successor link>
---
```
````

`docs/concepts/policy.md`:

````markdown
---
type: ConceptType
title: Policy
description: A rule, constraint, or permission that governs what should or must be done.
path: policies
---

# Policy

A **Policy** preserves what should or must be done: rules, constraints, permissions, and conditional guidance with their enforcement.

By convention, Policy records live at `gnosis://<vault>/policies/`.

## Use this for

- Normative and conditional knowledge — security controls, permissions, technology-selection rules, and situational guidance that answers "what should or must be done?" or "when does this apply?".

Do not use it for durable choices already settled (Decision), facts (Concept), or lessons (Reflection).

## Minimum record

- `# Policy` with the rule in one exact statement.
- `# Rationale`, `# Enforcement` describing how compliance is checked, and optional `# Exceptions`.

## Lifecycle

- Identity is the rule. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `draft` → `active` → `retired`; retired policies are retained with their rationale, never deleted.
- Change a rule in place only as a non-semantic correction; a changed rule retires the old record and creates a new one linked by `superseded_by`.
- Delete only a confirmed local duplicate or invalid `draft` after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Policy
title: <the rule>
description: <one-line summary>
status: <draft | active | retired>
applies_to: <optional scope of application>
superseded_by: <optional successor link>
---
```
````

- [x] Run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- [x] Run `gnosis get concepts`; expect 11 rows including Concept, Entity, Resource, Event, Memory, Reflection, and Policy.
- [x] Commit: `feat: add knowledge concept types`.

### Task 2: Verify bundle, scaffold, and suite

**Load:** `docs/embed.go`, `internal/vault/scaffold_test.go`, `internal/vault/search_test.go`.
**Files:** none modified.
**Interfaces:** consumes Task 1 records; produces verification evidence.

- [x] Run `go build ./... && go test ./...`; expect all packages ok (bundled-document tests assert existence, not counts).
- [x] Run `mise run checks`; expect exit 0.
- [x] The bundle is embedded at build time, so the pre-existing `gnosis` on PATH is stale: run `mise run build`, then `./dist/gnosis get pages gnosis://core/concepts/memory.md`; expect the Memory definition served from the `core` bundle origin, proving `docs/embed.go` picked it up.
- [x] Commit: `chore: verify bundled concept types` (only if any file changed; otherwise record the evidence and skip the commit).

# Acceptance criteria

- Seven new concept types are effective — run `gnosis get concepts`; expect 11 rows listing Concept, Entity, Resource, Event, Memory, Reflection, and Policy plus Purpose, Decision, Directive, and Procedure.
- The bundle serves them without a local vault — run `mise run build`, then `./dist/gnosis get pages gnosis://core/concepts/concept.md`; expect `vault: core` origin and the Concept definition body.
- Vault integrity holds — run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- No regressions — run `mise run checks`; expect gofmt, vet, tests with race detector, build, and vault validation all green.
