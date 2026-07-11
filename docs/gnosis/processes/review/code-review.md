---
type: Gnosis Process
title: code-review
description: Use when implementation needs independent review or review feedback must be evaluated before changes are accepted.
invocation: model
effects: [read, vault-write, workspace-write]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
---

# code-review

`code-review` applies one evidence standard to both sides of review: independent assessment of an exact change range and technical evaluation of the findings it produces.

## Use when

- An implementation task, complex fix, major feature, or branch needs independent review before acceptance.
- A human or automated reviewer reports issues or proposes changes.
- Review feedback may conflict with requirements, supported behavior, architecture, or active decisions.

## Knowledge inputs

- The governing directive or exact task brief and acceptance criteria.
- Relevant active decisions, repository constraints, implementation, tests, and supported environments.
- For independent review, exact base and head revisions, a contextual diff package, implementation report, and fresh test evidence.
- For feedback evaluation, the complete review and evidence cited by each finding.

## Process

1. Identify the mode and evidence boundary:
   - **Independent review:** record the exact base and head revisions and package the commit list, diff stat, and full contextual diff. Give a fresh read-only reviewer a self-contained brief without the controller's conclusions.
   - **Feedback evaluation:** read the complete review, restate each technical requirement in repository terms, and clarify ambiguous or dependent items before changing code.
2. Judge every item against requirements, current code, tests, actual call sites, supported environments, and active decisions. Treat external feedback as a hypothesis until repository evidence supports it.
3. Cover both requirement compliance and implementation quality, including correctness, boundaries, failure behavior, security, compatibility, tests, maintainability, and production readiness. Every finding names file-level evidence and an actual severity of Critical, Important, or Minor.
4. If a finding conflicts with an active decision, present the conflict to the author. If the author settles a replacement, create the record from the `Gnosis Decision` Concept Type definition and persist it with `gnosis write --type 'Gnosis Decision' --title '<title>' <draft-file>`.
5. End an independent review with an unambiguous approval or needs-fixes verdict. For received feedback, respond with technical acknowledgment, evidence-based rejection, or a precise unresolved question.
6. Resolve accepted Critical and Important findings one at a time in risk order. For behavior changes, invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/execution/test-driven-development.md' --pretty`; run focused and surrounding verification, then re-review the corrected range.
7. At branch completion, review the full merge-base-to-head range even when individual tasks were reviewed. If later evidence disproves an earlier conclusion, record the corrected technical understanding and continue.

## Completion

Every review item has an evidence-backed disposition, all blocking findings are resolved and re-reviewed, and the exact reviewed range has an explicit verdict covering both requirement compliance and implementation quality.

Adapted from `requesting-code-review` and `receiving-code-review`, analyzed in [Superpowers (obra)](../../../references/obra-superpowers.md).
