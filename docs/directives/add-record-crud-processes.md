---
type: Directive
title: Add record CRUD processes
description: Add focused decision and purpose lifecycle processes and migrate refine-purpose into purpose management.
status: draft
---

# Goal

Provide two canonical, model-invocable processes for managing Decision and Purpose records.

# Scope

Create `docs/procedures/records/manage-decisions.md` and `docs/procedures/records/manage-purpose.md` with the `vault` discovery tag. Absorb the author-confirmed refinement workflow from `refine-purpose` into `manage-purpose`, remove the old record, and update its incoming process link. Do not change runtime code, discovery configuration, unrelated process records, or `README.md`.

# Dependencies

- [`gnosis` purpose](gnosis://local/purpose.md) at `sha256:3334bd97fe5384ebb8ce4d50567b2ba0bb99f85b3f2161de63cbdce1d0254325`
- [Consolidate runtime adapters in the `gnosis` plugin](gnosis://local/decisions/consolidate-runtime-adapters-in-gnosis-plugin.md) at `sha256:1c7ebb07237bf6ab849172d10a9925221d6c754fd309d2840402ddc71b7a77be`
- [Procedure](gnosis://local/concepts/procedure.md) at `sha256:f538ea8e22c15ca80bd23e1efad3d929e8b792203ef9c1189a822ddf7313175d`

# Implementation plan

1. Apply the exact process-record additions and migration below.
2. Run `gnosis validate --vault .`; expect a successful validation summary covering the configured `docs` vault with no errors.
3. Run `go run ./cmd/gnosis procedure discovery`; expect `manage-decisions` and `manage-purpose` at their `records/` URIs and no `refine-purpose` entry from the current source tree.
4. Run `go test ./...`; expect every package to pass.

# Exact patch

Create `docs/procedures/records/manage-decisions.md` with this complete content:

```markdown
---
type: Procedure
title: manage-decisions
description: Use when a Decision record must be created, read, corrected, superseded, or deleted.
tags: [vault]
invocation: model
effects: [read, vault-write]
relationships:
  - type: instance_of
    target: gnosis://local/concepts/procedure.md
---

# manage-decisions

`manage-decisions` handles the Decision lifecycle while preserving settled intent and history.

## Inputs

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
```

Create `docs/procedures/records/manage-purpose.md` with this complete content:

```markdown
---
type: Procedure
title: manage-purpose
description: Use when the repository Purpose record must be created, read, refined, or deleted.
tags: [vault]
invocation: model
effects: [read, vault-write]
relationships:
  - type: instance_of
    target: gnosis://local/concepts/procedure.md
---

# manage-purpose

`manage-purpose` handles the singleton Purpose lifecycle through shared, author-confirmed understanding.

## Inputs

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
```

In `docs/procedures/planning/review-directive-purpose-decisions.md`, replace the purpose-change sentence with: `A reviewer cannot settle intent: ask the author about choices absent from approved requirements. Invoke the **Create or update** branch of [manage-purpose](../records/manage-purpose.md) for accepted purpose changes. Persist semantic decision changes as new superseding Decisions; edit in place only for non-semantic corrections.`

Delete `docs/procedures/vault/refine-purpose.md` after the replacement process contains all of its author-confirmation and record-shape safeguards.

# Acceptance criteria

- Exactly two process records exist in `docs/procedures/records/`: `manage-decisions.md` and `manage-purpose.md`.
- Both records satisfy the Procedure schema, use model invocation, retain the enabled `vault` tag, and expose their possible read/write effects.
- `manage-decisions` covers create, read, update, and delete while preserving decision history through supersession for semantic changes.
- `manage-purpose` covers create, read, update, and delete and contains the author-confirmed understanding safeguards formerly owned by `refine-purpose`.
- No live reference or discoverable process identity remains for `vault/refine-purpose.md`.
- `gnosis validate --vault .` and relevant tests pass.
