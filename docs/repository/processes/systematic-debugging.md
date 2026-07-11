---
type: Repository Process
title: systematic-debugging
description: Use for any bug, failing test, build failure, performance problem, or unexpected technical behavior before proposing a fix.
invocation: model
effects: [workspace-write]
relationships:
  - type: instance_of
    target: ../../concepts/repository-process.md
---

# systematic-debugging

Systematic debugging finds and verifies the root cause before changing production behavior. Symptom patches are not a substitute for causal understanding.

## Use when

- A test, build, integration, or production behavior fails.
- Behavior is intermittent, unexpectedly slow, or environment-dependent.
- An earlier fix failed or a seemingly obvious fix has not been demonstrated.

The hard gate is root-cause evidence before a proposed fix.

## Knowledge inputs

- The governing directive and relevant active decisions.
- Complete error output, reproduction steps, current implementation, and tests.
- Recent path-scoped changes that could explain the symptom.
- Comparable working code and authoritative technical references when a pattern is involved.

## Process

1. **Root cause investigation:** Read errors and stack traces completely, reproduce consistently, inspect recent relevant changes, and trace bad data or state backward to its origin. At multi-component boundaries, gather evidence for inputs, outputs, configuration, and state before localizing the failure.
2. **Pattern analysis:** Find a working analogue, read the applicable reference completely, enumerate every difference, and identify the dependencies and assumptions the broken path requires.
3. **Hypothesis testing:** State one specific causal hypothesis and its evidence. Test it with the smallest possible change and one variable. A failed test produces a new hypothesis, not a stack of speculative fixes.
4. **Root-cause fix:** Create the smallest failing reproduction and follow [test-driven-development](test-driven-development.md). Implement one change at the source, then verify the reproduction and the relevant suite.
5. After three failed fix attempts, stop and question the architecture with the author. Use [writing-directives-and-decisions](writing-directives-and-decisions.md) before changing the design and record a decision only if a new durable architectural choice is settled.
6. If evidence shows an external or timing-dependent cause, record what was ruled out in the task evidence, add appropriate handling and observability, and verify that behavior.

## Completion

Evidence identifies the root cause or establishes a bounded external cause, a regression test demonstrated the failure before the fix, the minimal correction passes focused and relevant regression checks, and any governing directive reflects the actual state.

Adapted from `systematic-debugging`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
