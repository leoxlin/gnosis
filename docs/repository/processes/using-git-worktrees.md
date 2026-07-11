---
type: Gnosis Process
title: using-git-worktrees
description: Use when feature work needs isolation or before executing a directive in a separate workspace.
invocation: model
effects: [workspace-write]
relationships:
  - type: instance_of
    target: ../../concepts/gnosis-process.md
---

# using-git-worktrees

This process establishes a safe isolated workspace while respecting author preferences and harness-owned worktrees. Detection precedes creation; native isolation precedes manual git worktrees.

## Use when

- Starting feature work whose changes should be isolated from the current checkout.
- Executing a directive that calls for a separate workspace.
- Resuming work where the current checkout's ownership or branch state is unclear.

## Knowledge inputs

- Repository agent instructions and any declared worktree preference.
- The governing directive and relevant workflow decisions.
- Current git directory, common directory, branch, submodule, and worktree state.
- Project setup and baseline verification commands.

## Process

1. Compare the resolved git directory with the common directory, identify the current branch, and guard against mistaking a submodule for a linked worktree.
2. If already in a linked worktree, retain it and proceed to setup. Report detached or harness-managed state accurately.
3. In a normal checkout, honor an existing author preference. Otherwise ask for consent before creating isolation; if declined, work in place.
4. Prefer the harness's native worktree mechanism. Use `git worktree add` only when no native mechanism exists.
5. For a manual project-local worktree, choose the location by explicit instruction, then existing `.worktrees/` or `worktrees/`, then `.worktrees/` by default. Verify the chosen directory is ignored before creating it.
6. Run project-appropriate dependency setup in the selected workspace.
7. Run the repository's baseline test or check command. A failing baseline requires reporting the failures and author direction before feature work proceeds.

If sandbox restrictions block worktree creation, report the constraint and use the current checkout only when that fallback remains authorized.

## Completion

The agent is in an author-approved workspace with known ownership and branch state, project setup has completed, and fresh baseline verification is either clean or explicitly accepted as pre-existing.

Adapted from `using-git-worktrees`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
