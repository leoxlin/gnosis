---
type: Gnosis Process
title: finishing-a-development-branch
description: Use after implementation and review are complete to verify delivery readiness and let the author choose integration or cleanup.
invocation: model
effects: [workspace-write, external]
relationships:
  - type: instance_of
    target: ../../concepts/gnosis-process.md
---

# finishing-a-development-branch

Finishing a branch verifies the completed work, detects who owns the workspace, presents bounded delivery choices, and performs only the author's selected integration or cleanup.

## Use when

- All implementation tasks and blocking reviews are complete.
- The work needs to be merged, proposed as a pull request, retained, or discarded.
- A directive is ready for its final delivery and status decision.

## Knowledge inputs

- The directive, its acceptance criteria, and relevant active decisions.
- Fresh whole-branch review and verification evidence.
- Current branch, merge base, worktree ownership, remote, and working-tree state.
- Repository-defined integration and release instructions.

## Process

1. Run [verification-before-completion](verification-before-completion.md) across the full suite, build, documentation checks, and every directive acceptance criterion. Stop on failure.
2. Detect normal checkout, named-branch worktree, or detached harness-managed workspace. Determine the likely base branch and ask when it is ambiguous.
3. For a named branch, present four concise choices: merge locally, push and create a pull request, keep the branch, or discard. For detached HEAD, omit local merge and offer push as a new branch, keep, or discard.
4. Execute only the selected choice:
   - **Merge:** switch safely to the base checkout, update as authorized, merge, and rerun verification on the merged result before cleanup.
   - **Pull request:** push and create the request, retaining the workspace for review iteration.
   - **Keep:** preserve branch and workspace and report their locations.
   - **Discard:** enumerate the branch, commits, and workspace to be deleted and require the exact author confirmation before destructive action.
5. Remove only worktrees created and owned by this process, only after a successful merge or confirmed discard, and only from outside the worktree. Preserve harness-managed workspaces.
6. Set the directive to `done` only when its acceptance criteria and selected delivery outcome are satisfied by fresh evidence. A retained branch may be done when delivery was not part of scope; a discarded implementation leaves or returns the directive to `open`.

## Completion

The author-selected integration state exists, verification covers that state, cleanup respected workspace ownership, and the directive status accurately represents the preserved result.

Adapted from `finishing-a-development-branch`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
