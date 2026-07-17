---
type: Directive
title: Add scoped agent memory
description: Ship the remember and recall vault procedures that implement mem0-style scoped agent memory as explicit Memory records.
status: done
---

# Goal

Implement [Adopt scoped agent memory as explicit vault records](../decisions/adopt-scoped-agent-memory.md): two new vault procedures, `remember` (extract → reconcile → ADD/UPDATE/DELETE/NONE) and `recall` (scoped hybrid retrieval), operating on the Memory concept type shipped by [Add the knowledge concept types](add-knowledge-concept-types.md).

# Scope

- Create `docs/procedures/vault/remember.md` and `docs/procedures/vault/recall.md` with the complete contents below.
- Update `docs/concepts/memory.md` to link the two procedures (exact edit below).
- No Go changes: the procedures compose existing `gnosis search knowledge`, `gnosis apply page`, `gnosis get pages`, and `gnosis validate vault`.
- No memory instances are created except the round-trip verification probe, which is archived by the probe itself.

# Global constraints

- Follow [Adopt scoped agent memory as explicit vault records](../decisions/adopt-scoped-agent-memory.md) and [Use pgvector for semantic knowledge retrieval](../decisions/use-pgvector-semantic-retrieval.md) (retrieval unchanged, lexical default).
- Procedure records must satisfy the Procedure contract (`description`, non-empty `tags`, no `effects`/`relationships`, `## Inputs/Process/Completion` sections) and pass vault validation with zero warnings.
- Memory writes are explicit record operations through `gnosis apply page`; no background or implicit mutation.

# Dependencies

- [Add the knowledge concept types](add-knowledge-concept-types.md) @ sha256:c7a98986dce1284500e4351ad0934b53004e4d4cf5784b4e9eb5a41dd77f0b4f — required contract: the Memory Concept Type is effective at `gnosis://local/concepts/memory.md` with the `scope`/`observed_at`/`hash`/`status` schema; evidence: `gnosis get pages gnosis://local/concepts/memory.md`; prerequisite must be `done`.

# Implementation plan

### Task 1: Ship the remember procedure

**Load:** `docs/procedures/vault/ingest-knowledge.md` (style), `docs/concepts/memory.md`, `docs/decisions/adopt-scoped-agent-memory.md`.
**Files:** create `docs/procedures/vault/remember.md`.
**Interfaces:** consumes an episode or statement from the author; produces Memory records via `gnosis apply page`.

- [x] Write `docs/procedures/vault/remember.md` exactly as below, then apply: `gnosis apply page gnosis://local/procedures/vault/remember.md --filename docs/procedures/vault/remember.md`; expect `changed: true`.

````markdown
---
type: Procedure
title: remember
description: Use when a supplied episode, conversation, or observation should become durable scoped Memory records.
tags: [gnosis, vault, memory]
invocation: model
---

# remember

`remember` turns supplied episodes into self-contained Memory records, reconciling every candidate against existing memories so each durable fact has exactly one active identity.

## Inputs

- The supplied episode, conversation, statement, or observation and the author's scope intent.
- Vault configuration, agent rules, and the effective Memory Concept Type definition.
- Existing Memory records returned by identity and similarity queries.

## Process

1. Resolve the vault and read the Memory Concept Type definition with `gnosis get pages gnosis://_/concepts/memory.md --full`.
2. Extract durable candidates from the episode. Each candidate is one self-contained statement: resolve pronouns, ground relative dates to absolute ones, preserve proper nouns and numbers verbatim, and keep only facts, preferences, and observations that outlive the current task. Never extract working state, plans, or transient context. Mark agent-inferred statements with `^[inferred]`.
3. Assign each candidate its frontmatter: `scope` (`user | agent | session | run`, defaulting to `user` for preferences and `agent` otherwise), `actor`, `source`, `observed_at` (absolute date), and `entities` (named entities mentioned).
4. Compute each candidate's `hash` as the SHA-256 hex of the exact statement text, e.g. `printf '%s' "<statement>" | sha256sum`. If an active Memory with the same `hash` exists, the candidate is NONE; stop processing it.
5. Retrieve the nearest existing memories for each remaining candidate with `gnosis search knowledge --backend lexical --vault <root> "<statement>"`, keeping only Memory records; add `gnosis search knowledge --backend vector --vault <root> "<statement>"` when semantic retrieval is configured.
6. Reconcile each candidate against its nearest neighbors:
   - **ADD** — new information: create a new Memory page at `memories/<kebab-statement-slug>.md` with `status: active`.
   - **UPDATE** — same subject, richer or corrected statement: revise the matching page in place (same URI), updating the statement, `hash`, and `observed_at`, preserving unknown metadata. Git history is the audit trail.
   - **DELETE** — contradicted by newer evidence: set the existing page's `status: archived` and add a body line `Archived: <date> — <reason>`; never silently remove it.
   - **NONE** — already captured or not durable: record the decision and move on.
7. Persist every write with `gnosis apply page '<memory URI>' --filename <draft-file>` and read it back. When `vault_log` is enabled, add one concise newest-first entry to the nearest `log.md` per write.
8. Run `gnosis validate vault --vault <root>`. Report the per-candidate operations and evidence.

## Completion

Every durable candidate has exactly one active Memory identity or an explicit NONE; duplicates and contradictions are reconciled as UPDATE or DELETE with retained audit; enabled navigation reflects the writes; and vault validation passes.
````

- [x] Run `gnosis get procedures gnosis://local/procedures/vault/remember.md --full`; expect the contract loads with no invalid-procedure error.
- [x] Commit: `feat: add remember procedure`.

### Task 2: Ship the recall procedure

**Load:** `docs/procedures/vault/query-vault.md` (style), `docs/concepts/memory.md`.
**Files:** create `docs/procedures/vault/recall.md`.
**Interfaces:** consumes a question and optional scope/entity filters; produces ranked memories with provenance; read-only.

- [x] Write `docs/procedures/vault/recall.md` exactly as below, then apply: `gnosis apply page gnosis://local/procedures/vault/recall.md --filename docs/procedures/vault/recall.md`; expect `changed: true`.

````markdown
---
type: Procedure
title: recall
description: Use when answering from scoped Memory records, such as user preferences or agent lessons.
tags: [gnosis, vault, memory]
invocation: model
---

# recall

`recall` answers from Memory records only, ranking scoped candidates by relevance, entity match, and recency, and returning each memory's provenance. It never writes.

## Inputs

- The author's question and any requested `scope`, `entities`, or recency bounds.
- Vault configuration and the effective Memory Concept Type definition.
- Memory candidates returned by lexical and, when configured, vector search.

## Process

1. Resolve the vault and the requested filters. Default to all active scopes when none are given.
2. Run `gnosis search knowledge --backend lexical --vault <root> "<question>"` and keep only Memory records; add `gnosis search knowledge --backend vector --vault <root> "<question>"` when semantic retrieval is configured, merging candidates by URI.
3. Exclude `status: archived` memories unless the author asks for history or the question is about a change of mind.
4. Rank the remaining candidates: scope match first, then shared `entities` with the question, then `observed_at` recency. Prefer the newest memory when several cover one subject; archived predecessors explain the change.
5. Read only the top candidates with `gnosis get pages '<URI>' --full`; never scan the whole `memories/` directory.
6. Answer with each memory's statement and provenance (`scope`, `actor`, `source`, `observed_at`). Label `^[inferred]` memories as inference. When no memory applies, report the gap and suggest `remember` if the author supplies the fact.

## Completion

The answer is grounded in ranked Memory records with visible provenance; archived memories surface only as history; no files changed.
````

- [x] Run `gnosis get procedures gnosis://local/procedures/vault/recall.md --full`; expect the contract loads with no invalid-procedure error.
- [x] Commit: `feat: add recall procedure`.

### Task 3: Link the procedures from the Memory concept type and round-trip verify

**Load:** `docs/concepts/memory.md`.
**Files:** modify `docs/concepts/memory.md`.
**Interfaces:** produces the corrected type record and round-trip evidence.

- [x] In `docs/concepts/memory.md`, replace this line:

```text
- Creation, update, and archival go through the `remember` vault procedure, which reconciles each candidate against the nearest existing memories as ADD, UPDATE, DELETE, or NONE; retrieval goes through the `recall` vault procedure. (Both ship with the agent-memory directive; link them here when they land.)
```

  with this line:

```text
- Creation, update, and archival go through [remember](../procedures/vault/remember.md), which reconciles each candidate against the nearest existing memories as ADD, UPDATE, DELETE, or NONE; retrieval goes through [recall](../procedures/vault/recall.md).
```
- [x] Apply: `gnosis apply page gnosis://local/concepts/memory.md --filename docs/concepts/memory.md`; run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- [x] Round-trip probe following `remember` exactly: extract one memory from the statement "The gnosis author prefers concise, lower-case project naming" (`scope: user`, `observed_at: <today>`, hash via `printf '%s' ... | sha256sum`); ADD it to `docs/memories/author-prefers-concise-lowercase-naming.md`; run `recall` for "naming preferences"; expect the probe ranked; then DELETE it (`status: archived`) per the procedure. Expect: exactly one archived Memory page remains, validation passes. This probe is real evidence, and the archived page is the negative-knowledge trail.
- [x] Commit: `feat: link memory procedures and verify round-trip`.

# Acceptance criteria

- Both procedures are discoverable — run `gnosis get procedures --tags gnosis,vault`; expect `remember` and `recall` listed; run `gnosis get procedures gnosis://local/procedures/vault/remember.md --full`; expect a complete contract.
- The Memory type links them — run `gnosis get pages gnosis://local/concepts/memory.md --full`; expect links to both procedure records and no placeholder text.
- The round-trip works — inspect `docs/memories/`; expect exactly one Memory page with `status: archived` whose body records the archive reason, proving ADD and DELETE.
- Vault integrity holds — run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- No regressions — run `mise run checks`; expect all green.
