---
type: Repository Process
title: receiving-code-review
description: Use before acting on review feedback, especially when a suggestion is unclear, broad, or inconsistent with repository constraints.
---

# receiving-code-review

Review feedback is technical evidence to evaluate, not an instruction to accept performatively. Correctness for this repository governs the response.

## Use when

- A human or automated reviewer reports issues or proposes changes.
- Feedback may conflict with existing behavior, scope, compatibility, or architecture.
- Several review items interact and partial implementation could create inconsistency.

## Knowledge inputs

- The complete review, not an isolated excerpt.
- The governing directive and relevant active decisions.
- Current implementation, tests, supported environments, and actual call sites.
- The reviewed diff and any evidence cited by the reviewer.

## Process

1. Read all feedback and restate each technical requirement in repository terms. Clarify every ambiguous or dependent item before implementing any of the set.
2. Verify each claim against code, tests, usage, and relevant decisions. External feedback is a hypothesis until repository evidence supports it.
3. Evaluate scope and YAGNI. Confirm that a proposed abstraction, endpoint, compatibility layer, or feature has an actual consumer or accepted requirement.
4. If feedback conflicts with an active Repository Decision, stop and present the conflict to the author. Record a replacement decision only after the author settles it; do not silently override durable knowledge.
5. Respond with technical acknowledgment, evidence-based pushback, or a precise question. Let the change and verification carry agreement instead of social performance.
6. Implement accepted items one at a time in risk order: blocking correctness or security, simple isolated corrections, then complex changes. Follow [test-driven-development](test-driven-development.md) and verify each before moving on.
7. If later evidence disproves an earlier pushback, state the corrected technical understanding and proceed without defending the earlier position.

## Completion

Every feedback item is clarified, verified against repository knowledge and implementation truth, then either implemented with evidence, rejected with technical grounds, or escalated to the author for a durable choice.

Adapted from `receiving-code-review`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
