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
7. Persist every write with `gnosis apply page '<memory URI>' --filename <draft-file>` and read it back. When `vault_log` is enabled, add one concise newest-first entry to the nearest `log.md`.
8. Run `gnosis validate vault --vault <root>`. Report the per-candidate operations and evidence.

## Completion

Every durable candidate has exactly one active Memory identity or an explicit NONE; duplicates and contradictions are reconciled as UPDATE or DELETE with retained audit; enabled navigation reflects the writes; and vault validation passes.
