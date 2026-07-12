---
type: Gnosis Process
title: review-directive-engineering
description: Use only when the latest complex draft needs a maximally skeptical, independent engineering-plan review.
tags: [gnosis-planning]
invocation: explicit
effects: [read, external]
use_when:
  - A complex directive is drafted or materially restructured.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# review-directive-engineering

## Knowledge inputs

- The exact draft and requirements revisions; repository root/HEAD/diff; purpose, decision, dependency, and sibling directive URI/revisions; repository rules and dependency metadata.

## Process

1. Bind exact inputs, then dispatch a different fresh read-only sub-agent with this prompt:

   > Inputs: review-process URI/revision; repository root/HEAD/diff; directive and requirements revisions; purpose, decision, dependency, and sibling directive revisions; rules; allowed paths. Read them. Review as a relentlessly skeptical but constructive senior engineer. Assume the plan is wrong until evidence supports it. Check coverage, root cause, exact loads/files/interfaces/commands/results, test quality, task/PR size, dependency contracts, reuse before new code, standard/native/installed options before new libraries, DRY/YAGNI, safety constraints, placeholders, invented APIs, and ambiguity.
   >
   > Read-only: no writes, mutating commands, sub-agents, or scope invention. Inspect an unlisted direct call site only to verify a finding; report each extra path and reason. Return `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | task/section | evidence | failure | minimum correction`. Critical is unsafe, contradictory, or impossible; Important is likely failure or material ambiguity; Minor is nonblocking. `NEEDS_FIXES` iff Critical or Important exists.

2. Preserve the report verbatim. Verify each finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. Return the immutable report and dispositions to finalization.

## Completion

The exact draft revision has an independent engineering verdict; every finding has an evidence-backed disposition.
