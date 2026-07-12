---
type: Procedure
title: creating-directives
description: Use after requirements refinement routes a directive-ready requirements packet for simple or complex planning.
tags: [gnosis-planning]
invocation: explicit
---

# creating-directives

## Knowledge inputs

- The exact requirements packet and its purpose/decision URI-revisions.
- Requirement-linked or query-selected code, tests, docs, conventions, dependencies, and existing directive records.
- Required directive contract: `draft` status, Goal, Scope, evidence-bearing Acceptance criteria, and an Implementation plan for multi-step work.

## Process

1. Classify the packet. Use the simple branch only for one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency. Otherwise use the complex branch.
2. Shape and draft the delivery:
   - **Simple:** Identify only the files and tests required for the smallest root-cause change. Draft one directive with exact files/interfaces, atomic steps containing complete code or patches, commands/results, criterion-level evidence, and applicable safety constraints. Use red-green-refactor plus focused and surrounding tests for behavior; exact validation otherwise.
   - **Complex:** Query first; trace only evidenced code, tests, docs, and interfaces. Prefer no change, existing mechanisms, standard capabilities, native features, installed dependencies, then minimum new code. For a new library, record package/version, install and manifest changes, reason, and why existing options fail. Split only at independently useful, testable pull-request boundaries. Keep coupled work together; build an acyclic dependency graph with supplied contracts. Assume the implementer has only the directive and a checkout. Embed complete code or patches in 2–5 minute, one-action steps with exact loads, files, interfaces, commands/results, and task commits. Use red-green-refactor plus focused and surrounding green for behavior; exact validation otherwise. Never use placeholders or “similar to.”
3. Self-review coverage, names/types, dependency order, DRY, YAGNI, test quality, task size, and evidence.
4. Route the result:
   - **Simple:** Invoke [finalizing-directives](finalizing-directives.md) with the unpersisted draft; skip independent planning review.
   - **Complex:** Persist every `draft` with `gnosis write '<directive URI>' --filename <draft-file>`; bind dependency links/revisions/contracts, then read back each URI/revision with `gnosis read '<directive URI>' --json`. In dependency order, invoke [reviewing-directives](reviewing-directives.md) against the latest revisions. Pass the immutable reports and controller dispositions with the complete draft set to [finalizing-directives](finalizing-directives.md).

## Completion

Every planned delivery has one validated `open` directive; complex dependencies are current, acyclic, and executable in order.
