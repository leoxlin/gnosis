---
type: Procedure
title: maintain-intents
description: Use only when the author explicitly says `maintain-intents` or `maintain intents`; never select it for implicit intent maintenance.
tags: [gnosis, development]
---

# maintain-intents

`maintain-intents` archives completed handoffs by merging their declared deltas into living intent records and compacting their durable choices into Decisions before removing them.

## Inputs

- The resolved vault, repository instructions, and vault configuration.
- The effective Directive and Decision Concept Type definitions.
- Every effective Directive whose status is `done`, its declared `# Purpose/Decision Changes` deltas, all existing Decisions, their provenance, and inbound links.

## Process

1. Read the effective Directive and Decision Concept Type definitions, list their records, and read every Directive whose effective status is `done` plus every Decision that may overlap its choices.
2. Archive declared deltas first. For every `done` Directive with a `# Purpose/Decision Changes` section, apply each delta to the living records through [managing-intents](managing-intents.md): create every `## Added` record, apply every `## Modified` replacement in full, and retire, supersede, or remove every `## Removed` record according to its Concept Type lifecycle. Persist and read back each changed record. Surface any delta that conflicts with the current record state to the author instead of choosing silently.
3. Extract only durable, non-obvious choices that still constrain future work. Exclude status, routine implementation details, transient instructions, duplicated rationale, choices already captured by the merged deltas, and facts recoverable from the current implementation or version history. Bind each retained choice to the completed Directive that evidences it.
4. Cluster extracted choices by the settled choice and constraint they preserve, not by topic alone. Merge each cluster with any matching Decision identity. Keep only the current choice, essential rationale, and constraints; drop repetition and superseded or no-longer-relevant detail. Surface unresolved contradictions to the author instead of choosing silently.
5. Build the smallest complete Decision set allowed by the effective Decision lifecycle. Reject duplicate identities, obtain any required author confirmation, preserve unknown metadata, and use correction or supersession rather than rewriting decision history. Persist and read back every created or corrected Decision before deleting a source Directive.
6. Trace inbound links to each local completed Directive and repair or intentionally remove them. After all of its declared deltas are merged and its retained decisions are durable, delete the Directive's exact local origin file. Do not mutate imported or bundled origins; report them to their owning vault. Leave unfinished Directives unchanged.
7. When `vault_index` is enabled, run `gnosis index vault --vault <root>`. Run `gnosis validate vault --vault <root>` after all writes and deletions.

## Completion

Every effective `done` Directive was inspected; every declared delta is merged into the living records or reported; each still-relevant durable choice appears once in the smallest lifecycle-compliant Decision set; retained Decisions contain only the essential current choice, rationale, and constraints; every deletable local `done` Directive and its inbound links are removed; non-local completed Directives are reported; unfinished Directives are unchanged; and vault validation passes.
