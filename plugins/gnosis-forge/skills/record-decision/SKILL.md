---
name: record-decision
description: Preserve a settled, non-obvious repository choice whose rationale or constraints must outlive the current task. Use for durable architecture, scope, dependency, data-model, or workflow decisions; do not use for routine implementation choices or change summaries.
---

# Record Decision

1. Read the repository purpose and related active decisions. Read `docs/concepts/repository-decision.md` only when classification or shape needs clarification.
2. Confirm the choice is non-obvious, durable, and constraining. Leave implementation history and routine delivery facts to git.
3. Write `docs/repository/decisions/<kebab-name>.md` with the decision and why it was chosen. Include only material constraints, rejected alternatives, and supersession links.
4. Regenerate indexes only when `vault_index` is enabled, then validate the vault. Do not duplicate the decision in a change log.

Finish when future work can identify the choice and its constraints without replaying the discussion.
