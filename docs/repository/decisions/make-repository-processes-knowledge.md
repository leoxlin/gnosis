---
type: Repository Decision
title: Make repository processes knowledge
description: Store development workflows as repository knowledge and apply them through an explicit `using-gnosis` adapter.
supersedes: keep-repository-context-minimal.md
---

# Decision

Represent repeatable development workflows as [Repository Process](../../concepts/repository-process.md) records under `docs/repository/processes/`. Make those records the source of truth and apply them through one manually invoked `using-gnosis` runtime skill. Rename `reason-with-knowledge` to `using-gnosis`; do not package each process as a duplicate skill.

Adapt the fourteen skills in the pinned [Superpowers reference](../../references/obra-superpowers.md) as repository processes. Preserve their names and engineering disciplines except for `using-superpowers`, which becomes `using-gnosis`. Ground each process in only the knowledge it needs:

- Brainstorming and writing plans read repository purpose and relevant decisions.
- Implementation, debugging, and review read the governing directive and relevant decisions without loading purpose by default.
- Current implementation and tests remain the truth about behavior; path-scoped git history fills only unexplained gaps.

Brainstorming records no general-purpose specification. It invokes `record-decision` only for settled, non-obvious choices that meet the Repository Decision boundary. `writing-plans` invokes `record-directive` and writes one directive per independently deliverable design; selecting `writing-plans` counts as an explicit request for that directive. Execution changes the directive to `blocked` when work cannot proceed and to `done` only after fresh evidence satisfies its acceptance criteria and the chosen delivery outcome preserves the work.

# Why

Repository processes are durable knowledge: they express how this repository turns intent into action and should remain readable, linkable, and adaptable without depending on a particular agent harness. A thin explicit adapter keeps runtime discovery separate from the process source of truth and avoids fourteen plugin copies drifting away from their repository records.

Rejected alternatives:

- **Package every adapted workflow as a skill** — rejected because the plugin and vault would contain competing copies of the same process.
- **Copy Superpowers verbatim** — rejected because its specification and plan artifacts do not express gnosis's purpose, decision, and directive boundaries.
- **Automatically invoke `using-gnosis` for all repository work** — rejected because the author chose an explicit workflow entry point.
- **Load all repository knowledge for every process** — rejected because most implementation disciplines need governing constraints, not the repository's entire durable context.

# Constraints

- Every process has one `Repository Process` page with selection conditions, knowledge inputs, ordered behavior, and an observable completion state.
- Process pages may reference packaged skills such as `record-decision` and `record-directive`; they do not reproduce those skills' classification logic.
- A directive created by `writing-plans` contains the executable task sequence in addition to the normal directive record.
- Routine progress and completion history remain in git and CI. Transient coordination files are execution scratch, not vault knowledge.
- `vault_index` and `vault_log` remain disabled for this repository.

# Related decisions

- [Bootstrap `gnosis` knowledge first on OKF](bootstrap-knowledge-first.md)
- [Name the knowledge-driven development bundle `gnosis-forge`](name-knowledge-driven-development-bundle-gnosis-forge.md)
- [Keep repository context minimal](keep-repository-context-minimal.md)
- [`gnosis` purpose](../purpose.md)
