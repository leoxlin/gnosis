---
type: Procedure
title: implementing-directive
description: Use when exactly one open directive must be implemented, verified, and delivered before any other directive starts.
tags: [gnosis, development]
invocation: model
---

# implementing-directive

## STEP 1 - selecting-directive

### Inputs

- One exact open directive URI/revision, its linked decisions, and any ordered bindings returned by planning.
- Current prerequisite statuses and supplied contracts, repository instructions, implementation, tests, workspace state, and merge base.

### Process

1. Bind exactly one directive. When planning returned multiple ordered bindings, select only the first dependency-ready binding and leave every other directive unstarted. This invocation never advances automatically to a sibling, dependent, follow-up, or next directive.
2. Read the selected directive with `gnosis get pages '<directive URI>' --full`; stop unless its current status is `open`. Record the selected URI/revision as the only directive this invocation may change.
3. Require every prerequisite directive to be `done` at its bound revision and every supplied contract to be present in the checkout. If a prerequisite is incomplete or has drifted, set only the selected directive to `blocked`, record the evidence, persist and read back that revision, validate the vault, and stop; do not implement the prerequisite in this invocation.
4. Inspect the selected directive's decisions, scope, acceptance criteria, implementation plan, affected implementation, and tests. Resolve contradictory decisions, obsolete assumptions, unsafe instructions, or steps whose intended result cannot be determined with the author before changing production behavior. If an author-owned issue remains unresolved, set only the selected directive to `blocked`, persist the issue and evidence, read back that revision, validate the vault, and stop.
5. Choose the smallest adequate execution mode: direct execution in the controlling session or delegated execution of one independently reviewable task at a time. Keep task briefs, reports, diff packages, and transient progress in repository-ignored scratch files.
6. Continue only with [preparing-workspace](#step-2---preparing-workspace). A blocked or discarded directive ends this invocation instead of causing another directive to be selected.

### Completion

Exactly one current `open` directive is active, its prerequisites and contracts are satisfied, its executable scope is understood, and every other directive remains unstarted.

## STEP 2 - preparing-workspace

### Inputs

- The selected directive, relevant decisions, repository instructions, and any declared worktree preference.
- Current git directory, common directory, branch, submodule, worktree ownership, dependency setup, and baseline verification commands.

### Process

1. Compare the resolved git directory with the common directory, identify the current branch, and distinguish a linked worktree from a submodule or normal checkout.
2. Retain an existing linked or harness-managed workspace. Report detached state and ownership accurately; never replace or clean up a harness-managed workspace.
3. In a normal checkout, honor an existing author preference. Otherwise obtain consent before creating isolation; if isolation is declined, work in place.
4. Prefer the harness's native isolation mechanism. Use `git worktree add` only when no native mechanism exists. For a manual project-local worktree, use an explicitly selected location, then an existing `.worktrees/` or `worktrees/`, then `.worktrees/` by default, and verify that the directory is ignored before creating it.
5. Run project-appropriate dependency setup in the selected workspace, then run the repository's baseline test or check command.
6. On a failing baseline, preserve the complete output and obtain author direction before implementation. Use the current checkout after an isolation failure only when that fallback remains authorized.

### Completion

The selected directive has one author-approved workspace with known ownership and branch state; setup is complete and fresh baseline verification is clean or explicitly accepted as pre-existing.

## STEP 3 - implementing-tasks

### Inputs

- The selected directive's ordered tasks, acceptance criteria, linked decisions, required process URIs, and supported-platform constraints.
- Existing interfaces, implementation boundaries, tests, focused and surrounding verification commands, and current task evidence.

### Process

1. Execute only the selected directive's tasks, in order, with at most one task active. In delegated mode, give the current task to one fresh implementer with its exact goal, scope, evidence, constraints, decisions, allowed files, inherited interfaces, required verification, report contract, and governing procedure URIs. When a task completes, mark its checkbox steps `- [x]` in the local directive file and persist with `gnosis apply page '<directive URI>' --filename <directive-file>` before starting the next task; derived progress must always match completed work.
2. For every task, invoke each governing procedure by exact URI with `gnosis get procedures '<procedure URI>' --full`. Require its completion evidence and the task's focused and surrounding verification before advancing.
3. Parallelize only independent domains inside the current task. Before dispatch, prove that no domain needs another's result and that their files and mutable resources cannot overlap. Modifying agents use separate workspaces; read-only investigators may share a checkout. Never dispatch work from another task or directive. Require each agent to report its evidence or implementation, unresolved concerns, changed or inspected paths, commands, and results; the controller reviews every artifact and owns all directive changes.
4. For each behavior change, use red-green-refactor:
   - **Red:** Write one minimal test through the real interface and run it until it fails because the required behavior is missing. A pass or setup error is not red.
   - **Green:** Write only the production code needed for that test, then run the focused and relevant surrounding tests until their output is clean.
   - **Refactor:** Improve names, boundaries, or duplication without adding behavior and while keeping the tests green. Repeat for the next required behavior or edge case.
5. Revert production code written before its failing test and reimplement it from the observed red state. Use mocks only where a real dependency cannot reasonably be exercised; never weaken a requirement to obtain green. For a non-behavior task, perform the directive's exact validation instead.
6. When a failure's root cause is not established, invoke `gnosis get procedures 'gnosis://_/procedures/development/debugging-methodically.md' --full` and vary one evidenced hypothesis at a time. Apply the resulting fix only to the selected directive and current task.
7. After each task, require focused and surrounding verification. In delegated mode, also require a file-based implementation report and exact base-to-head diff package; independently inspect the diff, resolve blocking findings, and record the reviewed commit range before starting the next task.
8. After parallel or delegated work, review every artifact and diff, check assumptions and paths for overlap, integrate compatible changes, and run the combined verification required by the current task. Individually valid reports are not evidence that the integrated state passes.
9. If implementation genuinely cannot continue, set only the selected directive to `blocked`, record the concrete blocker and evidence, persist it with `gnosis apply page '<directive URI>' --filename <directive-file>`, read back its URI/revision, validate the vault, and stop. Never switch to another directive.

### Completion

Every task in the selected directive is implemented in order with required procedure evidence, red-green-refactor or exact validation, clean focused and surrounding checks, and no unresolved overlap from concurrent work.

## STEP 4 - reviewing-implementation

### Inputs

- The selected directive and decisions, merge base, current head and diff, task reports, reviewed commit ranges, and verification evidence.
- Repository review instructions and the exact acceptance criteria, interfaces, safety constraints, and allowed paths.

### Process

1. Assess every delegated task report and diff independently; a report identifies claims to inspect but is not proof of correctness.
2. When the complete change requires independent review, assess the merge-base-to-head range for scope compliance, correctness, regression risk, test quality, interface compatibility, safety, unnecessary code, and every acceptance criterion.
3. Verify each finding against the current implementation and governing decisions. Resolve every blocking finding through [implementing-tasks](#step-3---implementing-tasks) for the same selected directive, then reassess the affected diff and verification evidence.
4. Do not begin another task with unresolved blocking findings and do not begin another directive under any outcome of this review.

### Completion

The selected directive's complete implementation has current review evidence, every blocking finding is resolved and reassessed, and the reviewed range is explicit.

## STEP 5 - verifying-directive

### Inputs

- The selected directive's goal, scope, tasks, acceptance criteria, linked decisions, and repository-defined quality gates.
- The current implementation and diff, delegated artifacts, and exact commands or observations that can prove each prospective completion claim.

### Process

1. Translate every acceptance criterion and prospective success claim into the exact command or observation that would prove it against the current state. Map each `#### Scenario:` block to its **WHEN** setup and **THEN** observation and prove each one.
2. Run every complete verification command, including the required full suite, build, and documentation checks. Read the full output, exit status, failure counts, warnings, and scope.
3. Inspect delegated changes and claims independently. Compare evidence line by line with the directive and decisions; passing tests do not prove untested requirements or another layer's behavior.
4. For a regression test, retain evidence that it failed for the broken behavior and passes for the corrected behavior. A test observed only in the passing state does not prove that it detects the regression.
5. When evidence fails and a safe correction remains within the selected directive, return to [implementing-tasks](#step-3---implementing-tasks), then repeat review and complete verification. If execution cannot continue, set only the selected directive to `blocked`, record the actionable evidence, persist and read back that revision, validate the vault, and stop. Never continue with another directive.

### Completion

Every acceptance criterion and stated success for the selected directive is backed by fresh, complete evidence from the current state, with no failure or uncertainty represented as success.

## STEP 6 - finishing-directive

### Inputs

- The selected directive and fresh complete review and verification evidence.
- Current branch, merge base, worktree ownership, remote, working-tree state, and repository-defined integration and release instructions.

### Process

1. Require [verifying-directive](#step-5---verifying-directive) to cover the full suite, build, documentation checks, and every acceptance criterion. Stop on any failure.
2. Detect a normal checkout, named-branch worktree, or detached harness-managed workspace and determine the likely base branch. Resolve an ambiguous base with the author.
3. For a named branch, present merge locally, push and create a pull request, keep the branch, or discard. For detached HEAD, omit local merge and offer push as a new branch, keep, or discard.
4. Execute only the author's selected outcome:
   - **Merge:** Switch safely to the base checkout, update as authorized, merge, and repeat complete verification on the merged result before cleanup.
   - **Pull request:** Push and create the request, retaining the workspace for review iteration.
   - **Keep:** Preserve the branch and workspace and report their locations.
   - **Discard:** Enumerate the exact branch, commits, and workspace that would be deleted and require exact author confirmation before destructive action.
5. Remove only worktrees created and owned by this procedure, only after a successful merge or confirmed discard, and only from outside the worktree. Preserve harness-managed workspaces.
6. Set the selected directive to `done` only when fresh evidence satisfies every acceptance criterion and the author-selected delivery outcome is achieved with the verified work preserved. A failed delivery leaves it `open` or `blocked` with evidence; a retained branch may be `done` when delivery was outside scope, while a discarded implementation remains or returns to `open`.
7. Persist the selected directive with `gnosis apply page '<directive URI>' --filename <directive-file>`, read back its URI/revision, validate the vault, report the preserved delivery state, and return. Starting another directive always requires a new invocation.

### Completion

The selected directive alone is `blocked`, `done`, or `open` after confirmed discard; its status matches the preserved evidence and delivery state, workspace ownership was respected, and no other directive was started.
