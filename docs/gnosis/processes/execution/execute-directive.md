---
type: Gnosis Process
title: execute-directive
description: Use when an open directive with an implementation plan must be executed directly or through sequential delegated tasks.
invocation: model
effects: [vault-write, workspace-write, external]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
---

# execute-directive

`execute-directive` critically reviews an open implementation directive, selects direct or delegated execution, and preserves verification and delivery evidence through completion.

## Use when

- An open directive contains an executable implementation plan.
- Work will execute directly, continue in another session, or use fresh agents for independently reviewable tasks.

Use [dispatching-parallel-agents](dispatching-parallel-agents.md) only for genuinely independent problem domains. Directive tasks that share a branch remain sequential.

## Knowledge inputs

- The exact open directive and its linked decisions.
- Relevant process records named by the directive or required by its tasks.
- Current implementation, tests, repository instructions, workspace state, and merge base.
- For delegated execution, task briefs, reports, diff packages, and transient progress state.

## Process

1. Read the directive with `gnosis read --id '<directive URI>'`, then inspect its linked decisions and current implementation. Resolve contradictions, obsolete assumptions, missing dependencies, or unclear steps with the author before implementation.
2. When isolation is needed, invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/execution/using-git-worktrees.md' --pretty` and complete that process before continuing.
3. Select the smallest adequate mode:
   - **Direct:** execute tasks in order in the controlling session.
   - **Delegated:** assign each independently reviewable task, one at a time, to a fresh implementer with only its directive text, inherited interfaces, relevant decisions, and exact process URIs. Keep briefs, reports, diff packages, and progress in repository-ignored scratch files.
4. For every task, invoke each governing process by exact URI with `gnosis process invoke --id '<process URI>' --pretty`. Require its completion evidence and the task's focused and surrounding verification before advancing.
5. In delegated mode, require a file-based implementation report and exact base-to-head diff package. Invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/review/code-review.md' --pretty` for each task, resolve blocking findings, and record the approved commit range before dispatching the next task.
6. If execution cannot continue, update the directive from its Concept Type definition to `blocked` with the concrete blocker and evidence. Persist the complete record with `gnosis write --type 'Gnosis Directive' --title '<title>' <draft-file>` and stop instead of guessing.
7. After all tasks, invoke `code-review` for the complete merge-base-to-head range when the change requires independent review. Resolve and re-review all blocking findings.
8. Invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/execution/verification-before-completion.md' --pretty` against every acceptance criterion, then invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/execution/finishing-a-development-branch.md' --pretty`.
9. Set the directive to `done` from its Concept Type definition only when fresh evidence satisfies its acceptance criteria and the selected delivery outcome preserves the verified work. A discarded implementation remains or returns to `open`.

## Completion

The directive is `blocked` with actionable evidence or `done` with every task, required review, acceptance criterion, and selected delivery outcome freshly verified. Delegated tasks have durable reports and approved commit ranges, and no claim rests on an unchecked plan assertion.

Adapted from `executing-plans` and `subagent-driven-development`, analyzed in [Superpowers (obra)](../../../references/obra-superpowers.md).
