---
type: Repository Process
title: subagent-driven-development
description: Use when an open directive has independently reviewable implementation tasks that should execute in the current session with isolated agent context.
---

# subagent-driven-development

Subagent-driven development assigns each directive task to a fresh implementer, gates it with an independent task review, and finishes with a whole-branch review. Tasks remain sequential so every implementer starts from a reviewed repository state.

## Use when

- An open directive contains a complete implementation plan.
- Tasks are independently reviewable even though they build on one another.
- The current harness supports subagents and isolated context.

Use [executing-plans](executing-plans.md) when subagents are unavailable. Use [dispatching-parallel-agents](dispatching-parallel-agents.md) only for genuinely independent problem domains, not adjacent implementation tasks sharing a branch.

## Knowledge inputs

- The complete directive, its acceptance criteria, and its linked decisions.
- The Repository Process pages governing each task, especially testing and verification.
- Current repository instructions, implementation, tests, workspace state, and merge base.
- Per-task briefs, reports, diff packages, and transient progress state.

## Process

1. Establish an isolated workspace through [using-git-worktrees](using-git-worktrees.md), read the directive once, and preflight its tasks for contradictions with global constraints or one another. Resolve blocking plan defects with the author before dispatch.
2. Use a repository-ignored scratch directory for task briefs, reports, diff packages, and a progress ledger. These files support execution recovery; they are not Repository Process, Decision, or Directive records and are not committed.
3. For each incomplete task, create a brief containing that task's exact directive text, its place in the design, interfaces inherited from prior tasks, and only the relevant decisions and processes. Do not pass accumulated conversation history.
4. Dispatch one fresh implementer at a time in the reviewed workspace. Choose an explicitly adequate model when the harness supports model selection. Require clarification before guesses, red-green-refactor evidence, focused and full verification, a commit, self-review, and a file-based report with status `DONE`, `DONE_WITH_CONCERNS`, `NEEDS_CONTEXT`, or `BLOCKED`.
5. Respond to status precisely: supply missing context, increase reasoning capacity, split an oversized task, or return a flawed directive to the author. If execution cannot proceed, set the directive to `blocked` with the concrete blocker.
6. For completed work, create an exact base-to-head diff package and dispatch a fresh read-only reviewer with the task brief, relevant decisions, report, and diff. Require both directive compliance and code-quality verdicts.
7. Send all Critical and Important findings to a fixer, require fresh covering tests in the report, and re-review until both verdicts approve. Resolve findings that conflict with the directive or an active decision with the author.
8. Mark the task complete in transient progress only after approval, recording its commit range. Resume after compaction from that ledger and git, never from memory or by redispatching completed tasks.
9. After all tasks, request a broad merge-base-to-head review, resolve and re-review its blocking findings, then run [verification-before-completion](verification-before-completion.md) and [finishing-a-development-branch](finishing-a-development-branch.md).
10. Set the directive to `done` only after its acceptance criteria have fresh evidence and the chosen delivery result is preserved.

## Completion

Every directive task has an implementation report, clean task-level compliance and quality verdicts, and a recorded commit range; the whole branch has independent review and fresh verification; and the directive accurately reads `blocked` or `done`.

Adapted from [`subagent-driven-development`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/subagent-driven-development/SKILL.md), analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
