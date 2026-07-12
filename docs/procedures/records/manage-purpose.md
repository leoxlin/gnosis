---
type: Procedure
title: manage-purpose
description: Use when the repository Purpose record must be created, read, refined, or deleted.
tags: [gnosis-vault]
invocation: model
use_when:
  - Creating the repository purpose.
  - Reading the purpose to understand its outcome, beneficiaries, sub-purposes, or boundaries.
  - Changing or resolving ambiguity in author-owned repository intent.
  - Deleting the purpose after its knowledge impact is resolved.
---

# manage-purpose

`manage-purpose` handles the singleton Purpose lifecycle through shared, author-confirmed understanding.

## Knowledge inputs

- The requested operation, resolved vault, repository instructions, and vault configuration.
- The effective Purpose Concept Type definition and current Purpose record, when one exists.
- Relevant active decisions, concepts, implementation facts, provenance, and inbound links.
- Required record shape: `type`, `title`, `description`, `# Purpose`, optional `# Sub-purposes`, and `# Boundaries`; exclude architecture, plans, milestones, and tasks.

## Process

1. Resolve the vault and requested operation. Read the effective Purpose Concept Type definition, then list exact Purpose records. Treat multiple effective Purpose records as an identity conflict and stop for repair; for an exact target, read it as JSON to retain its URI, revision, origin, and unknown metadata.
2. Follow the matching branch:
   - **Read:** Return the current outcome, beneficiaries, sub-purposes, boundaries, provenance, conflicts, and gaps. Make no vault change and stop.
   - **Create or update:** Gather discoverable facts and distinguish them from author-owned intent. Ask exactly one author-owned question at a time, recommend an answer with rationale, and wait before asking a dependent question. Explore every material branch of the outcome, beneficiaries, sub-purposes, and boundaries until no unresolved author-owned choice remains. Summarize the proposed Purpose and obtain explicit confirmation of shared understanding.
   - **Delete:** Trace every inbound link. Explain the resulting loss of repository intent, obtain explicit author confirmation that the vault should no longer contain a Purpose record, and repair or intentionally remove every inbound reference. Delete only a confirmed local origin; report an imported or bundled target to its owning vault instead.
3. For creation or update, build the single Purpose record in the required shape, preserve applicable unknown metadata, and persist it with `gnosis write '<purpose URI>' --filename <draft-file>`. For deletion, remove only the confirmed local origin file identified by the exact JSON read.
4. When `vault_index` is enabled, run `gnosis index --vault <root>`. Run `gnosis validate --vault <root>` after every write or deletion.

## Completion

The read result is provenance-grounded with no vault change, or exactly one author-confirmed Purpose lifecycle change is reflected in the intended effective record, every affected link is repaired, and vault validation passes.
