---
type: Gnosis Process
title: systematic-debugging
description: Use for any bug, failing test, build failure, performance problem, or unexpected technical behavior before proposing a fix.
invocation: model
effects: [vault-write, workspace-write]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
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
4. **Root-cause fix:** Create the smallest failing reproduction and invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/execution/test-driven-development.md' --pretty`. Implement one change at the source, then verify the reproduction and the relevant suite.
5. After three failed fix attempts, stop and question the architecture with the author. Invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/planning/writing-directives-and-decisions.md' --pretty` before changing the design.
6. If evidence shows an external or timing-dependent cause, record what was ruled out in the task evidence, add appropriate handling and observability, and verify that behavior.

## Completion

Evidence identifies the root cause or establishes a bounded external cause, a regression test demonstrated the failure before the fix, the minimal correction passes focused and relevant regression checks, and any governing directive reflects the actual state.

Adapted from `systematic-debugging`, analyzed in [Superpowers (obra)](../../../references/obra-superpowers.md).
