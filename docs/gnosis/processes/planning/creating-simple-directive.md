---
type: Gnosis Process
title: creating-simple-directive
description: Use only after requirements refinement routes one bounded delivery with no material design or dependency work.
invocation: explicit
effects: [read, vault-write]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
---

# creating-simple-directive

## Use when

- Refinement selected one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency.

## Knowledge inputs

- The exact requirements packet and its purpose/decision URI-revisions.
- Exact affected code, tests, and repository rules.
- The [Gnosis Directive](../../../concepts/gnosis-directive.md) definition.

## Process

1. If the use condition fails, stop and invoke [creating-complex-directives](creating-complex-directives.md) with the unchanged packet.
2. Identify only the files and tests required for the smallest root-cause change.
3. Draft one `draft` directive with exact files/interfaces, atomic steps containing complete code or patches, commands/results, criterion-level evidence, and applicable safety constraints. Use red-green-refactor plus focused and surrounding tests for behavior; exact validation otherwise.
4. Invoke [finalizing-directives](finalizing-directives.md) with the unpersisted draft; skip independent planning reviews.

## Completion

One validated `open` directive exists with an exact URI/revision.
