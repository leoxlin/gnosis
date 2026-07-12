---
type: Procedure
title: manage-decisions
description: Use when a Decision record must be created, read, corrected, superseded, or deleted.
tags: [gnosis-vault]
invocation: model
---

# manage-decisions

`manage-decisions` handles the Decision lifecycle while preserving settled intent and history.

## Knowledge inputs

- The requested operation, resolved vault, repository instructions, and vault configuration.
- The effective Decision Concept Type definition.
- Identity-query results, exact candidate records, provenance, inbound links, and evidence for the choice.

## Process

1. Resolve the vault and requested operation. Read the effective Decision Concept Type definition. When the operation or target identity is ambiguous, ask the author one question at a time and recommend a default.
2. Query before reading records. Use an identity-focused `gnosis query graph --vault <root> '<question>'`, then read only its applicable `should_read` records. For an exact target, read it as JSON to retain its URI, revision, origin, and unknown metadata.
3. Follow the matching branch:
   - **Read:** Return the exact choice, rationale, constraints, supersession history, provenance, conflicts, and gaps. Make no vault change and stop.
   - **Create:** Establish the non-obvious choice, material alternatives, rationale, and constraints. Resolve every author-owned choice, obtain explicit confirmation, reject duplicate identity, and build the minimum complete Decision record.
   - **Update:** Classify the change before writing. Apply a non-semantic correction in place while preserving unknown metadata. For any changed choice, rationale, or constraint, obtain explicit author confirmation and create a new Decision whose `supersedes` field links the prior Decision; preserve the prior record unchanged.
   - **Delete:** Trace inbound links and supersession history. Prefer correction or supersession whenever the record has governed work. Delete only a confirmed local duplicate or invalid record after the author explicitly approves the exact deletion and every inbound reference is repaired or intentionally removed. Report imported or bundled targets to their owning vault instead of deleting them.
4. Persist each created or corrected record with `gnosis write '<decision URI>' --filename <draft-file>`. For deletion, remove only the confirmed local origin file identified by the exact JSON read.
5. When `vault_index` is enabled, run `gnosis index --vault <root>`. Run `gnosis validate --vault <root>` after every write or deletion.

## Completion

The read result is provenance-grounded with no vault change, or the confirmed Decision lifecycle change preserves identity and history, repairs every affected link, and passes vault validation.
