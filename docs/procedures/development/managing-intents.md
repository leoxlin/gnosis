---
type: Procedure
title: managing-intents
description: Use when a Purpose, Decision, or Directive record must be created, read, updated, or deleted.
tags: [gnosis, development]
invocation: model
---

# managing-intents

`managing-intents` applies one provenance-aware lifecycle workflow while each Concept Type defines its own identity, history, and mutation rules.

## Inputs

- The requested operation and record type, resolved vault, repository instructions, and vault configuration.
- The effective Concept Type definition, including its lifecycle rules.
- Identity-query results, exact candidate records, provenance, inbound links, and evidence for the requested change.
- Current requirements, implementation state, dependencies, and verification evidence when the target type requires them.

## Process

1. Resolve the vault, operation, and Concept Type. Read the effective Concept Type definition completely and treat its lifecycle section as the record-specific policy. When the operation or target identity is ambiguous, ask the author one question at a time and recommend a default.
2. Find candidates according to the type's identity rules. Query before reading non-singleton records with an identity-focused `gnosis query graph --vault <root> '<question>'`, then read only applicable `should_read` records. List singleton records directly. Read an exact target as JSON to retain its URI, revision, origin, status, and unknown metadata. Stop on an identity conflict.
3. Follow the matching branch without weakening the Concept Type lifecycle:
   - **Read:** Return the exact record content, lifecycle state and history, provenance, inbound links, conflicts, and evidence gaps. Make no vault change and stop.
   - **Create:** Gather discoverable facts, resolve author-owned choices, reject duplicate identity, and build the minimum complete record. Obtain explicit author confirmation when the type requires it. If the type delegates creation to another procedure, invoke that procedure with the request and current knowledge bindings instead of writing directly.
   - **Update:** Classify the change as semantic, non-semantic, or a lifecycle transition before writing. Preserve applicable unknown metadata. Apply the type's in-place, supersession, replanning, history, confirmation, and evidence rules; invoke any procedure that owns the required transition. Reject a transition or mutation the type does not allow.
   - **Delete:** Trace inbound links and type-specific history. Prefer the type's correction, supersession, or retention path. Delete only the exact confirmed local origin when the Concept Type permits it, after explicit author approval and repair or intentional removal of every inbound reference. Report imported or bundled targets to their owning vault.
4. Persist direct creation or in-place correction with `gnosis write '<record URI>' --filename <draft-file>`. For deletion, remove only the confirmed local origin file identified by the exact JSON read. A delegated procedure persists the changes it owns.
5. When `vault_index` is enabled, run `gnosis index --vault <root>`. Run `gnosis validate --vault <root>` after every write or deletion.

## Completion

The read result is provenance-grounded with no vault change, or the requested lifecycle change follows the effective Concept Type, preserves required identity and history, repairs every affected link, and passes vault validation.
