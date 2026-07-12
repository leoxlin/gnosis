---
type: Gnosis Process
title: finalizing-directives
description: Use only when simple drafts or reviewed complex drafts must become validated execution handoffs.
invocation: explicit
effects: [read, vault-write]
use_when:
  - A simple directive is drafted.
  - Complex-directive reviews have returned findings or approval.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# finalizing-directives

## Knowledge inputs

- Exact draft URI/revisions or one unpersisted simple draft; frozen requirements; immutable review reports and dispositions; latest purpose/decision revisions.
- Required directive contract: status, Goal, Scope, evidence-bearing Acceptance criteria, an Implementation plan for multi-step work, and linked revision-bound contracts for prerequisites.

## Process

1. Persist any simple draft as `draft` with `gnosis write`. Read every draft with `gnosis read --id '<directive URI>' --pretty`; on revision drift, stop and refresh affected reviews.
2. Reject schema or coverage gaps, ambiguous paths/interfaces, non-atomic steps, omitted code/patches, weak tests, missing expected results, unsafe omissions, cycles, and unverifiable criteria. Preserve every finding disposition.
3. Apply accepted feedback. If a simple draft becomes complex or directive boundaries/count change, return the affected set to [creating-complex-directives](creating-complex-directives.md). After any other material correction, persist the `draft`, read its new revision, then repeat [purpose/decision](review-directive-purpose-decisions.md) and [engineering](review-directive-engineering.md) reviews. Stop for the author if the same accepted blocker survives one correction.
4. Finalize in dependency order. Verify current dependency contracts; when a provider contract changes, update and re-review all downstream dependents. Change `draft` to `open` only when the latest required reviews leave no accepted Critical or Important finding.
5. Persist each `open` record with `gnosis write`, read back its URI/revision, validate the vault, and return the ordered bindings. Offer [execute-directive](../execution/execute-directive.md) only on an execution request.

## Completion

Every directive is `open` at a reviewed current revision; every finding has a disposition, dependencies are current and acyclic, and execution order is explicit.
