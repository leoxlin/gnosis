# Plan with OpenSpec

Use OpenSpec for repository proposals, requirements, technical choices,
implementation tasks, and completed-change history.

## Start a change

From the repository root:

    openspec new change <change-name>
    openspec status --change <change-name>

Follow the configured artifact order. A complete plan normally includes a
proposal, capability delta specs, a design, and checkbox tasks.

## Implement and inspect

Use the change's `tasks.md` as the implementation checklist and keep it aligned
with completed work. Current requirements live under `docs/openspec/specs/`;
active changes live under `docs/openspec/changes/`.

gnosis projects standard OpenSpec Markdown paths as read-only vault knowledge:

    gnosis get concepts OpenSpecSpec
    gnosis get pages gnosis://local/openspec/specs/vault-management/spec.md --full

Edit OpenSpec artifacts through the OpenSpec workflow, not `gnosis apply page`.

## Validate and finish

    openspec validate --all --strict --no-interactive

After implementation passes its checks, sync the delta specs into the main
specs and archive the change. The archive is the durable rationale and delivery
history.
