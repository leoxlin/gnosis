---
type: Repository Process
title: writing-plans
description: Use after an approved design or bounded requirements need an executable multi-step implementation handoff.
---

# writing-plans

`writing-plans` turns an approved design into one executable [Repository Directive](../../concepts/repository-directive.md). The familiar process name remains; the durable artifact is a directive rather than a separate plan document.

## Use when

- An approved design requires multiple implementation steps.
- Work must be handed to another agent or resumed without replaying the design conversation.
- Exact paths, interfaces, tests, commands, and acceptance evidence must be settled before implementation.

Selecting this process is an explicit request to invoke `record-directive`.

## Knowledge inputs

- The [repository purpose](../purpose.md).
- The approved design and relevant active decisions.
- Current code structure, tests, build commands, and repository conventions.
- The Repository Directive definition and the `record-directive` skill.

## Process

1. Recheck scope. Split designs that can be delivered and accepted independently; create one directive for each independent deliverable, not one directive per mechanical task.
2. Map every file to create, modify, or test and the responsibility it will hold. Follow existing patterns and introduce only boundaries required by the design.
3. Divide work into independently reviewable tasks. Each task carries its own red-green-refactor cycle and ends in a testable result.
4. Write exact steps: file paths, interfaces consumed and produced, test code or precise test behavior, commands, expected results, minimal implementation, full verification, and commits where the repository workflow uses them.
5. Invoke `record-directive` with `status: open`, the goal, bounded scope, links to governing decisions, an ordered `# Implementation plan`, and observable acceptance criteria. The directive is the single source of implementation requirements.
6. Self-review the directive for design coverage, placeholders, contradictory names or types, undefined dependencies, untestable criteria, and tasks that are too broad.
7. Offer the author the preserved execution choices: [subagent-driven-development](subagent-driven-development.md) for independent tasks in the current session, or [executing-plans](executing-plans.md) for direct or separate-session execution.

## Completion

Each independently deliverable design has one validated open directive with a complete executable task sequence and acceptance criteria, and the author has an explicit execution choice.

Adapted from [`writing-plans`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/writing-plans/SKILL.md), analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
