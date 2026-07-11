---
type: Repository Process
title: verification-before-completion
description: Use immediately before claiming work is correct, complete, fixed, passing, or ready for delivery.
invocation: model
effects: [read]
relationships:
  - type: instance_of
    target: ../../concepts/repository-process.md
---

# verification-before-completion

Completion claims require fresh evidence. Confidence, prior output, partial checks, and delegated reports do not establish the current repository state.

## Use when

- Reporting a task, fix, directive, build, or test suite complete.
- Committing, opening a pull request, merging, or moving to another task.
- Accepting work returned by an agent or automation.
- Changing a directive to `done`.

## Knowledge inputs

- The directive's goal, scope, implementation tasks, and acceptance criteria.
- Relevant decisions and repository-defined quality gates.
- The exact commands or observations that prove each claim.
- The current diff and delegated artifacts when work came from another agent.

## Process

1. Translate every prospective claim and acceptance criterion into the command or observation that would prove it now.
2. Run each complete verification command against the current state. Read its full output, exit status, failure counts, warnings, and scope.
3. Inspect delegated changes and claims independently; a report selects what to verify but is not verification.
4. Compare evidence line by line with the directive and relevant decisions. Passing tests do not prove untested requirements, and one layer's success does not prove another layer.
5. When evidence fails, state the observed status and keep or set the directive to `open` or `blocked` as appropriate. Continue through the governing implementation or debugging process.
6. Make the completion claim, or set the directive to `done`, only when the cited fresh evidence proves the whole claim.

For a regression test, demonstrate red on the broken behavior and green on the corrected behavior; a test that only passes once does not prove it detects the regression.

## Completion

Every stated success and satisfied acceptance criterion is backed by fresh, complete evidence from the current state, with failures and uncertainty reported explicitly.

Adapted from `verification-before-completion`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
