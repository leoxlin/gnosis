---
type: Procedure
title: planning-directives
description: Use when requested planning must turn a prompt, issue, or bug report into validated execution handoffs.
tags: [gnosis, development]
invocation: model
---

# planning-directives

## STEP 1 - refining-requirements

### Knowledge inputs

- The author's request, repository rules, and current purpose.
- Query-selected decisions, implementation, tests, issues, and authoritative sources.

### Process

1. Query gnosis first. Read only returned records and affected paths. Research only unresolved planning claims; retain each claim's source and version. Ask one author-owned question at a time and recommend a default.
2. For a bug without current root-cause evidence, invoke [systematic-debugging](../debugging/systematic-debugging.md) in diagnosis-only mode. Reuse current debugging evidence instead of reinvoking it. If diagnosis is the requested outcome, return the evidence and stop.
3. Produce one exact requirements packet: outcome, in/out scope, constraints, acceptance evidence, governing knowledge URI/revisions, and resolved choices. Stop with a blocker if any unknown could change the plan; confirm inferred author-owned choices.
4. Pass the packet unchanged to [creating-directives](#step-2---creating-directives), identifying whether it is one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency, or complex work.

### Completion

Diagnosis is complete, or evidence-backed requirements are routed by exact process URI.

## STEP 2 - creating-directives

### Knowledge inputs

- The exact requirements packet and its purpose/decision URI-revisions.
- Requirement-linked or query-selected code, tests, docs, conventions, dependencies, and existing directive records.
- Required directive contract: `draft` status, Goal, Scope, evidence-bearing Acceptance criteria, and an Implementation plan for multi-step work.

### Process

1. Classify the packet. Use the simple branch only for one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency. Otherwise use the complex branch.
2. Shape and draft the delivery:
   - **Simple:** Identify only the files and tests required for the smallest root-cause change. Draft one directive with exact files/interfaces, atomic steps containing complete code or patches, commands/results, criterion-level evidence, and applicable safety constraints. Use red-green-refactor plus focused and surrounding tests for behavior; exact validation otherwise.
   - **Complex:** Query first; trace only evidenced code, tests, docs, and interfaces. Prefer no change, existing mechanisms, standard capabilities, native features, installed dependencies, then minimum new code. For a new library, record package/version, install and manifest changes, reason, and why existing options fail. Split only at independently useful, testable pull-request boundaries. Keep coupled work together; build an acyclic dependency graph with supplied contracts. Assume the implementer has only the directive and a checkout. Embed complete code or patches in 2–5 minute, one-action steps with exact loads, files, interfaces, commands/results, and task commits. Use red-green-refactor plus focused and surrounding green for behavior; exact validation otherwise. Never use placeholders or “similar to.”
3. Self-review coverage, names/types, dependency order, DRY, YAGNI, test quality, task size, and evidence.
4. Route the result:
   - **Simple:** Continue with [finalizing-directives](#step-4---finalizing-directives) using the unpersisted draft; skip independent planning review.
   - **Complex:** Persist every `draft` with `gnosis write '<directive URI>' --filename <draft-file>`; bind dependency links/revisions/contracts, then read back each URI/revision with `gnosis read '<directive URI>' --json`. In dependency order, continue with [reviewing-directives](#step-3---reviewing-directives) against the latest revisions. Pass the immutable reports and controller dispositions with the complete draft set to [finalizing-directives](#step-4---finalizing-directives).

### Completion

Every planned delivery has one validated `open` directive; complex dependencies are current, acyclic, and executable in order.

## STEP 3 - reviewing-directives

### Knowledge inputs

- The exact draft and requirements revisions; repository root/HEAD/diff; purpose, decision, dependency, and sibling directive URI/revisions; repository rules and dependency metadata.
- Purpose holds enduring outcomes and boundaries. Decisions hold durable non-obvious choices and use `supersedes` for semantic replacement. Directives remain `draft` until finalization.

### Process

1. Bind and read the review-process URI/revision; repository root/HEAD; directive URI/revision; exact requirements packet and hash or URI/revision; purpose and supplied decision URI/revisions; repository rules; and allowed paths. Query relevant active decisions and list every added URI/revision.
2. Perform a read-only purpose/decision pass. Check only whether the goal advances purpose and respects its boundaries, any scope or task conflicts with a decision, and the plan implies a purpose change or durable non-obvious choice. Do not write, run mutating commands, or invent scope. Produce an immutable report containing `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | section | evidence | failure | minimum correction | knowledge change`. Critical is unsafe, contradictory, or impossible; Important is likely failure or material ambiguity; Minor is nonblocking. Use `NEEDS_FIXES` iff Critical or Important exists.
3. Verify each purpose/decision finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. A reviewer cannot settle intent: ask the author about choices absent from approved requirements. Invoke the matching branch of [managing-intents](managing-intents.md) for accepted Purpose or Decision changes; the effective Concept Type defines confirmation, in-place correction, and supersession rules.
4. Add `# Purpose/Decision Changes` only for persisted changes, listing old→new URI/revisions and effects. Persist any revised `draft` with `gnosis write '<directive URI>' --filename <draft-file>`, then read it back with `gnosis read '<directive URI>' --json` and bind its current revision.
5. Bind the repository diff and current directive, requirements, purpose, decision, dependency, and sibling directive revisions. Perform a separate read-only engineering pass as a relentlessly skeptical but constructive senior engineer; assume the plan is wrong until evidence supports it. Check coverage, root cause, exact loads/files/interfaces/commands/results, test quality, task/PR size, dependency contracts, reuse before new code, standard/native/installed options before new libraries, DRY/YAGNI, safety constraints, placeholders, invented APIs, and ambiguity. Do not write, run mutating commands, or invent scope. Inspect an unlisted direct call site only to verify a finding; report each extra path and reason.
6. Produce an immutable engineering report containing `Reviewed: <bindings>`, `Verdict: APPROVE|NEEDS_FIXES`, then `F<n> | Critical|Important|Minor | task/section | evidence | failure | minimum correction`. Apply the same severity definitions and verdict rule as the purpose/decision pass. Verify each finding and record `F<n> → ACCEPT|REJECT|AUTHOR` with evidence. Return both immutable reports, all dispositions, the current draft revision, and changed knowledge revisions to finalization.

### Completion

The latest draft revision has separate purpose/decision and engineering verdicts; every finding has an evidence-backed disposition and accepted knowledge changes are versioned and linked.

## STEP 4 - finalizing-directives

### Knowledge inputs

- Exact draft URI/revisions or one unpersisted simple draft; frozen requirements; immutable review reports and dispositions; latest purpose/decision revisions.
- Required directive contract: status, Goal, Scope, evidence-bearing Acceptance criteria, an Implementation plan for multi-step work, and linked revision-bound contracts for prerequisites.

### Process

1. Persist any simple draft as `draft` with `gnosis write '<directive URI>' --filename <draft-file>`. Read every draft with `gnosis read '<directive URI>' --json`; on revision drift, stop and refresh affected reviews.
2. Reject schema or coverage gaps, ambiguous paths/interfaces, non-atomic steps, omitted code/patches, weak tests, missing expected results, unsafe omissions, cycles, and unverifiable criteria. Preserve every finding disposition.
3. Apply accepted feedback. If a simple draft becomes complex or directive boundaries/count change, return the affected set to [creating-directives](#step-2---creating-directives). After any other material correction, persist the `draft`, read its new revision, then repeat [reviewing-directives](#step-3---reviewing-directives). Stop for the author if the same accepted blocker survives one correction.
4. Finalize in dependency order. Verify current dependency contracts; when a provider contract changes, update and re-review all downstream dependents. Change `draft` to `open` only when the latest required reviews leave no accepted Critical or Important finding.
5. Persist each `open` record with `gnosis write '<directive URI>' --filename <draft-file>`, read back its URI/revision, validate the vault, and return the ordered bindings. On an execution request, offer [implementing-directive](implementing-directive.md) for only the first dependency-ready binding; every later directive requires a separate invocation after the selected directive finishes.

### Completion

Every directive is `open` at a reviewed current revision; every finding has a disposition, dependencies are current and acyclic, and execution order is explicit.
