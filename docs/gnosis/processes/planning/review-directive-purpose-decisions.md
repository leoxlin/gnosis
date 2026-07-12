---
type: Gnosis Process
title: review-directive-purpose-decisions
description: Use only when a persisted complex draft needs independent review against repository purpose and decisions.
invocation: explicit
effects: [read, vault-write]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
---

# review-directive-purpose-decisions

## Use when

- A complex directive is drafted or materially restructured.

## Knowledge inputs

- The exact draft URI/revision, frozen requirements contract, repository snapshot, [purpose](../../purpose.md), and supplied decision URI/revisions.
- The [Purpose](../../../concepts/gnosis-purpose.md), [Decision](../../../concepts/gnosis-decision.md), and [Directive](../../../concepts/gnosis-directive.md) definitions.

## Process

1. Bind exact inputs, then dispatch a fresh read-only sub-agent with this prompt:

   > Inputs: review-process URI/revision; repository root/HEAD; directive URI/revision; exact requirements packet and hash or URI/revision; purpose and supplied decision URI/revisions; repository rules; allowed paths. Read them and independently query relevant active decisions, listing every added URI/revision. Check only: Does the goal advance purpose and respect its boundaries? Does any scope or task conflict with a decision? Does the plan imply a purpose change or durable non-obvious choice?
   >
   > Read-only: no writes, mutating commands, sub-agents, or scope invention. Return `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | section | evidence | failure | minimum correction | knowledge change`. Critical is unsafe, contradictory, or impossible; Important is likely failure or material ambiguity; Minor is nonblocking. `NEEDS_FIXES` iff Critical or Important exists.

2. Preserve the report verbatim. Verify each finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. A reviewer cannot settle intent: ask the author about choices absent from approved requirements. Invoke [refine-purpose](../vault/refine-purpose.md) for accepted purpose changes. Persist semantic decision changes as new superseding Decisions; edit in place only for non-semantic corrections.
3. Add `# Purpose/Decision Changes` only for persisted changes, listing old→new URI/revisions and effects. Persist the revised `draft` with `gnosis write`, read it back with `gnosis read --id '<directive URI>' --pretty`, and return its current revision with the report, dispositions, and changed knowledge revisions.

## Completion

The latest draft revision has an independent verdict; every finding has a disposition and accepted knowledge changes are versioned and linked.
