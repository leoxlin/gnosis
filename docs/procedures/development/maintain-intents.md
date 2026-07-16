---
type: Procedure
title: maintain-intents
description: Use only when the author explicitly says `maintain-intents` or `maintain intents`; never select it for implicit intent maintenance.
tags: [gnosis, development]
---

# maintain-intents

`maintain-intents` compacts durable intent out of completed work before removing the completed handoffs.

## Inputs

- The resolved vault, repository instructions, and vault configuration.
- The effective Directive and Decision Concept Type definitions.
- Every effective Directive whose status is `done`, all existing Decisions, their provenance, and inbound links.

## Process

1. Read the effective Directive and Decision Concept Type definitions, list their records, and read every Directive whose effective status is `done` plus every Decision that may overlap its choices.
2. Extract only durable, non-obvious choices that still constrain future work. Exclude status, routine implementation details, transient instructions, duplicated rationale, and facts recoverable from the current implementation or version history. Bind each retained choice to the completed Directive that evidences it.
3. Cluster extracted choices by the settled choice and constraint they preserve, not by topic alone. Merge each cluster with any matching Decision identity. Keep only the current choice, essential rationale, and constraints; drop repetition and superseded or no-longer-relevant detail. Surface unresolved contradictions to the author instead of choosing silently.
4. Build the smallest complete Decision set allowed by the effective Decision lifecycle. Reject duplicate identities, obtain any required author confirmation, preserve unknown metadata, and use correction or supersession rather than rewriting decision history. Persist and read back every created or corrected Decision before deleting a source Directive.
5. Trace inbound links to each local completed Directive and repair or intentionally remove them. After all of its retained decisions are durable, delete the Directive's exact local origin file. Do not mutate imported or bundled origins; report them to their owning vault. Leave unfinished Directives unchanged.
6. When `vault_index` is enabled, run `gnosis index vault --vault <root>`. Run `gnosis validate vault --vault <root>` after all writes and deletions.

## Completion

Every effective `done` Directive was inspected; each still-relevant durable choice appears once in the smallest lifecycle-compliant Decision set; retained Decisions contain only the essential current choice, rationale, and constraints; every deletable local `done` Directive and its inbound links are removed; non-local completed Directives are reported; unfinished Directives are unchanged; and vault validation passes.
