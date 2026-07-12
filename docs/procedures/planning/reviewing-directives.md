---
type: Procedure
title: reviewing-directives
description: Use only when the latest persisted complex draft needs purpose, decision, and engineering review.
tags: [gnosis-planning]
invocation: explicit
use_when:
  - A complex directive is drafted or materially restructured.
---

# reviewing-directives

## Knowledge inputs

- The exact draft and requirements revisions; repository root/HEAD/diff; purpose, decision, dependency, and sibling directive URI/revisions; repository rules and dependency metadata.
- Purpose holds enduring outcomes and boundaries. Decisions hold durable non-obvious choices and use `supersedes` for semantic replacement. Directives remain `draft` until finalization.

## Process

1. Bind and read the review-process URI/revision; repository root/HEAD; directive URI/revision; exact requirements packet and hash or URI/revision; purpose and supplied decision URI/revisions; repository rules; and allowed paths. Query relevant active decisions and list every added URI/revision.
2. Perform a read-only purpose/decision pass. Check only whether the goal advances purpose and respects its boundaries, any scope or task conflicts with a decision, and the plan implies a purpose change or durable non-obvious choice. Do not write, run mutating commands, or invent scope. Produce an immutable report containing `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | section | evidence | failure | minimum correction | knowledge change`. Critical is unsafe, contradictory, or impossible; Important is likely failure or material ambiguity; Minor is nonblocking. Use `NEEDS_FIXES` iff Critical or Important exists.
3. Verify each purpose/decision finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. A reviewer cannot settle intent: ask the author about choices absent from approved requirements. Invoke the **Create or update** branch of [manage-purpose](../records/manage-purpose.md) for accepted purpose changes. Persist semantic decision changes as new superseding Decisions; edit in place only for non-semantic corrections.
4. Add `# Purpose/Decision Changes` only for persisted changes, listing old→new URI/revisions and effects. Persist any revised `draft` with `gnosis write '<directive URI>' --filename <draft-file>`, then read it back with `gnosis read '<directive URI>' --json` and bind its current revision.
5. Bind the repository diff and current directive, requirements, purpose, decision, dependency, and sibling directive revisions. Perform a separate read-only engineering pass as a relentlessly skeptical but constructive senior engineer; assume the plan is wrong until evidence supports it. Check coverage, root cause, exact loads/files/interfaces/commands/results, test quality, task/PR size, dependency contracts, reuse before new code, standard/native/installed options before new libraries, DRY/YAGNI, safety constraints, placeholders, invented APIs, and ambiguity. Do not write, run mutating commands, or invent scope. Inspect an unlisted direct call site only to verify a finding; report each extra path and reason.
6. Produce an immutable engineering report containing `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | task/section | evidence | failure | minimum correction`. Apply the same severity definitions and verdict rule as the purpose/decision pass. Verify each finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. Return both immutable reports, all dispositions, the current draft revision, and changed knowledge revisions to finalization.

## Completion

The latest draft revision has separate purpose/decision and engineering verdicts; every finding has an evidence-backed disposition and accepted knowledge changes are versioned and linked.
