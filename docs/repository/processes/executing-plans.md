---
type: Repository Process
title: executing-plans
description: Use when an open directive with an implementation plan must be executed directly or in a separate session.
---

# executing-plans

`executing-plans` loads an implementation directive, reviews it critically, and executes its tasks in order with the verification gates it specifies.

## Use when

- An open directive contains an executable implementation plan.
- Subagent-driven execution is unavailable, inappropriate, or intentionally not selected.
- Work is continuing in a separate session and needs a durable handoff.

## Knowledge inputs

- The open Repository Directive, including its implementation plan and acceptance criteria.
- Active decisions and Repository Process pages linked by or relevant to that directive.
- Current implementation, tests, repository instructions, and workspace state.

## Process

1. Read the complete directive and its linked decisions. Inspect current implementation where needed to verify that the plan still matches reality.
2. Review the plan for missing dependencies, contradictions, obsolete assumptions, or unclear steps. Resolve blocking concerns with the author before implementation.
3. Establish the workspace through [using-git-worktrees](using-git-worktrees.md) when isolation is needed.
4. Execute tasks in order. For each task, follow the named processes, run every specified verification, and confirm its independently testable result before moving forward.
5. If work cannot continue, set the directive to `blocked`, add the concrete blocker and evidence, and stop instead of guessing. Return it to `open` when the blocker is resolved.
6. After all tasks, use [verification-before-completion](verification-before-completion.md) against every acceptance criterion, then follow [finishing-a-development-branch](finishing-a-development-branch.md).
7. Set the directive to `done` only when fresh evidence satisfies its criteria and the selected delivery outcome preserves the verified work. Discarding the work leaves or returns it to `open`.

## Completion

The directive is either `blocked` with an actionable reason or `done` with fresh verification evidence and preserved work. No task or completion claim rests on an unchecked plan assertion.

Adapted from [`executing-plans`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/executing-plans/SKILL.md), analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
