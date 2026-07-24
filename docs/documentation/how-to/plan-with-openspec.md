# Plan with OpenSpec

Use OpenSpec for repository proposals, requirements, technical choices,
implementation tasks, and completed-change history. The repository points to
the registered `trium` store, whose checkout is maintained separately.

## Start a change

From the repository root:

    openspec new change <change-name>
    openspec status --change <change-name>

Follow the configured artifact order. A complete plan normally includes a
proposal, capability delta specs, a design, and checkbox tasks.

## Implement and inspect

Use the change's `tasks.md` as the implementation checklist and keep it aligned
with completed work. Current requirements and changes live in the `trium`
store.

Inspect artifacts through OpenSpec:

    openspec list --specs
    openspec show gnosis-vault-management --type spec

## Validate and finish

    openspec validate --all --strict --no-interactive

After implementation passes its checks, sync the delta specs into the main
specs and archive the change. The archive is the durable rationale and delivery
history.
