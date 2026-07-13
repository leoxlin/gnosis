---
type: Procedure
title: debugging-methodically
description: Use for diagnosing or fixing any bug, failing test, build failure, performance problem, or unexpected technical behavior.
tags: [gnosis, development, debugging]
invocation: model
---

# debugging-methodically

Debugging methodically finds and verifies root cause before planning or changing production behavior.

## Inputs

- Any governing directive and relevant active decisions.
- Complete error output, reproduction steps, current implementation, and tests.
- Recent path-scoped changes that could explain the symptom.
- Comparable working code and authoritative technical references when a pattern is involved.

## Process

1. Set the requested outcome: diagnosis/planning or implementation. **Root cause investigation:** Read errors and stack traces completely, reproduce consistently, inspect recent relevant changes, and trace bad data or state backward to its origin. At multi-component boundaries, gather evidence for inputs, outputs, configuration, and state before localizing the failure.
2. **Pattern analysis:** Find a working analogue, read the applicable reference completely, enumerate every difference, and identify the dependencies and assumptions the broken path requires.
3. **Hypothesis testing:** State one specific causal hypothesis and its evidence. Test it with the smallest possible change, varying only one variable. A failed test produces a new hypothesis, not a stack of speculative fixes.
4. For diagnosis or planning, return the evidence-backed cause, regression boundary, constraints, and reproduction without changing production behavior. Otherwise, create the smallest failing reproduction, observe it fail for the intended reason, implement only the source-level fix needed to make it pass, refactor while green, then run focused and relevant surrounding verification. When debugging was invoked by an active directive procedure, return this evidence to that same directive and task.
5. After three failed fix attempts, stop and question the architecture with the author. Invoke `gnosis procedure invoke --uri 'gnosis://core/procedures/development/planning-directives.md'` with the existing evidence before changing the design.
6. If evidence shows an external or timing-dependent cause, record what was ruled out in the task evidence, add appropriate handling and observability, and verify that behavior.

## Completion

Diagnosis returns an evidence-backed root or bounded external cause and reproduction. An implemented fix also has a regression test observed failing first, focused and relevant passing checks, and keeps any governing directive accurate.
