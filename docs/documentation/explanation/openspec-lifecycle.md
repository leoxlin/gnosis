# OpenSpec lifecycle

gnosis uses two kinds of execution knowledge for different purposes. Vault
Procedures describe repeatable work that an agent can discover at runtime.
OpenSpec describes changes to the gnosis codebase itself.

## Why the boundary exists

A repository change needs a proposed outcome, observable requirements,
technical decisions, an implementation checklist, and a durable completion
record. Those artifacts form a lifecycle: they begin as a change, affect the
current specifications, and end in an archive.

A vault Procedure has a different lifecycle. It remains an active reusable
contract and may be refined without representing one particular software
change. Combining these models would either turn planning history into
invocable behavior or burden procedures with delivery state.

## Artifact roles

- `proposal.md` establishes the problem, outcome, and affected capabilities.
- Delta specifications state additions, modifications, and removals as
  observable requirements.
- `design.md` records technical choices and trade-offs.
- `tasks.md` tracks delivery against the design and requirements.
- The archive preserves the completed change after its deltas are synchronized
  into the main specifications.

This repository registers the external `trium` OpenSpec store. gnosis projects
that store's Markdown into its read and search view. The projection does not
transfer ownership: OpenSpec remains the only authoring interface for these
artifacts.
