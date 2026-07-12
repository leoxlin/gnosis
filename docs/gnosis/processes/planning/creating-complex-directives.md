---
type: Gnosis Process
title: creating-complex-directives
description: Use only after requirements refinement routes material design, research, governance, or dependent-delivery work.
invocation: explicit
effects: [read, vault-write, external]
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# creating-complex-directives

## Use when

- Refined requirements contain material architecture, purpose, decision, research, or cross-directive dependency work.
- Or they require multiple dependent, independently useful deliveries.

## Knowledge inputs

- The exact requirements packet and its purpose/decision URI-revisions.
- Requirement-linked or query-selected code, tests, docs, conventions, dependencies, and existing directive records.

## Process

1. Query first; trace only evidenced code, tests, docs, and interfaces. Prefer no change, existing mechanisms, standard capabilities, native features, installed dependencies, then minimum new code. For a new library, record package/version, install and manifest changes, reason, and why existing options fail.
2. Split only at independently useful, testable pull-request boundaries. Keep coupled work together; build an acyclic dependency graph with supplied contracts.
3. Assume the implementer has only the directive and a checkout. Embed complete code or patches in 2–5 minute, one-action steps with exact loads, files, interfaces, commands/results, and task commits. Use red-green-refactor plus focused and surrounding green for behavior; exact validation otherwise. Never use placeholders or “similar to.”
4. Self-review coverage, names/types, dependency order, DRY, YAGNI, test quality, task size, and evidence. Persist every `draft` with `gnosis write --type 'Gnosis Directive' --title '<title>' <draft-file>`; bind dependency links/revisions/contracts, then read back each URI/revision with `gnosis read --id '<directive URI>' --pretty`.
5. In dependency order, invoke [purpose/decision review](review-directive-purpose-decisions.md), then [engineering review](review-directive-engineering.md) against the latest revisions. Pass both immutable reports and controller dispositions with the complete draft set to [finalization](finalizing-directives.md).

## Completion

Every PR-sized delivery has one reviewed `open` directive; dependencies are current, acyclic, and executable in order.
