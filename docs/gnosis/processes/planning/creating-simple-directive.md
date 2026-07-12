---
type: GnosisProcess
title: creating-simple-directive
description: Use only after requirements refinement routes one bounded delivery with no material design or dependency work.
tags: [gnosis-planning]
invocation: explicit
effects: [read, vault-write]
use_when:
  - Refinement selected one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# creating-simple-directive

## Knowledge inputs

- The exact requirements packet and its purpose/decision URI-revisions.
- Exact affected code, tests, and repository rules.
- Required directive contract: `draft` status, Goal, Scope, evidence-bearing Acceptance criteria, and an Implementation plan for multi-step work.

## Process

1. If the use condition fails, stop and invoke [creating-complex-directives](creating-complex-directives.md) with the unchanged packet.
2. Identify only the files and tests required for the smallest root-cause change.
3. Draft one `draft` directive with exact files/interfaces, atomic steps containing complete code or patches, commands/results, criterion-level evidence, and applicable safety constraints. Use red-green-refactor plus focused and surrounding tests for behavior; exact validation otherwise.
4. Invoke [finalizing-directives](finalizing-directives.md) with the unpersisted draft; skip independent planning reviews.

## Completion

One validated `open` directive exists with an exact URI/revision.
