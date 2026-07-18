# OpenSpec lifecycle

OpenSpec is gnosis's repository-development source of truth. It keeps project
planning separate from the portable vault Procedure model.

## Artifacts

- `proposal.md` records why a change exists and what it affects.
- Delta specs record observable requirement additions, modifications, and removals.
- `design.md` records technical choices, trade-offs, and migration strategy.
- `tasks.md` records implementation progress.
- The archive preserves completed changes after their deltas are synchronized into the main specs.

## Flow

1. Create a kebab-case change with `openspec new change <name>`.
2. Write the proposal, delta specs, design, and tasks required by the configured schema.
3. Implement against the tasks and validate behavior.
4. Sync the delta specs into `docs/openspec/specs/`.
5. Archive the completed change under `docs/openspec/changes/archive/`.

## Why it is separate

Vault Procedures remain ordinary typed knowledge for repeatable vault work.
OpenSpec owns repository requirements, technical choices, delivery tasks, and
change history directly, so gnosis does not duplicate or proxy that lifecycle.
