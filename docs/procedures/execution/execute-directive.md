---
type: Procedure
title: execute-directive
description: Use when an open directive with an implementation plan must be executed directly or through sequential delegated tasks.
tags: [gnosis-execution]
invocation: model
effects: [vault-write, workspace-write, external]
use_when:
  - An open directive contains an executable implementation plan.
  - Work will execute directly, continue in another session, or use fresh agents for independently reviewable tasks.
  - Use dispatching-parallel-agents only for genuinely independent problem domains; tasks sharing a branch remain sequential.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/procedure.md
---

# execute-directive

`execute-directive` critically reviews an open implementation directive, selects direct or delegated execution, and preserves verification and delivery evidence through completion.

## Knowledge inputs

- The exact open directive and its linked decisions.
- Relevant process records named by the directive or required by its tasks.
- Current implementation, tests, repository instructions, workspace state, and merge base.
- For delegated execution, task briefs, reports, diff packages, and transient progress state.

## Process

1. Read the directive with `gnosis read '<directive URI>' --json`; stop unless its current status is `open`. Inspect linked decisions and implementation. Require every prerequisite directive to be `done` and its supplied contract present in the checkout. Resolve contradictions, obsolete assumptions, or unclear steps with the author before implementation.
2. When isolation is needed, invoke `gnosis process invoke --id 'gnosis://core/procedures/execution/using-git-worktrees.md' --pretty` and complete that process before continuing.
3. Select the smallest adequate mode:
   - **Direct:** execute tasks in order in the controlling session.
   - **Delegated:** assign each independently reviewable task, one at a time, to a fresh implementer with only its directive text, inherited interfaces, relevant decisions, and exact process URIs. Keep briefs, reports, diff packages, and progress in repository-ignored scratch files.
4. For every task, invoke each governing process by exact URI with `gnosis process invoke --id '<process URI>' --pretty`. Require its completion evidence and the task's focused and surrounding verification before advancing.
5. In delegated mode, require a file-based implementation report and exact base-to-head diff package. Assess each task against its acceptance criteria and verification evidence, resolve blocking findings, and record the reviewed commit range before dispatching the next task.
6. If execution cannot continue, preserve the complete directive, set its status to `blocked`, record the concrete blocker and evidence, persist it with `gnosis write '<directive URI>' --filename <draft-file>`, and stop instead of guessing.
7. After all tasks, assess the complete merge-base-to-head range when the change requires independent review. Resolve and reassess all blocking findings.
8. Invoke `gnosis process invoke --id 'gnosis://core/procedures/execution/verification-before-completion.md' --pretty` against every acceptance criterion, then invoke `gnosis process invoke --id 'gnosis://core/procedures/execution/finishing-a-development-branch.md' --pretty`.
9. Set the directive to `done` only when fresh evidence satisfies its acceptance criteria and the selected delivery outcome preserves the verified work. A discarded implementation remains or returns to `open`.

## Completion

The directive is `blocked` with actionable evidence or `done` with every task, required assessment, acceptance criterion, and selected delivery outcome freshly verified. Delegated tasks have durable reports and reviewed commit ranges, and no claim rests on an unchecked plan assertion.
