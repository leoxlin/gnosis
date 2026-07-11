---
type: Gnosis Process
title: writing-directives-and-decisions
description: Use to turn creative repository work into an author-approved design, durable decisions, and executable directives.
invocation: model
effects: [vault-write]
relationships:
  - type: instance_of
    target: ../../../concepts/gnosis-process.md
---

# writing-directives-and-decisions

`writing-directives-and-decisions` turns an idea into an author-approved design, records the durable choices that govern it, and creates an executable [Gnosis Directive](../../../concepts/gnosis-directive.md). Its hard gate is unchanged: implementation begins only after the design is understood and approved.

## Use when

- Creating a feature, component, integration, or product behavior.
- Changing existing behavior or choosing among materially different designs.
- A request is underspecified enough that implementation would encode author-owned assumptions.
- An approved design or bounded requirements need an executable multi-step implementation handoff.
- Exact paths, interfaces, tests, commands, and acceptance evidence must be settled before implementation.

Scale the conversation to the decision, but do not skip it because the change appears small.

## Knowledge inputs

- The [repository purpose](../../purpose.md) and boundaries.
- Active decisions and concepts relevant to the proposed behavior.
- The exact `Gnosis Decision` and `Gnosis Directive` Concept Type definitions.
- Existing implementation, tests, and repository conventions.
- Recent path-scoped history only when current knowledge and code leave a design choice unexplained.

## Process

1. Read the purpose and relevant knowledge with `gnosis read --id '<gnosis URI>'`. Read the Decision and Directive definitions with `gnosis read --id 'gnosis://core/concepts/gnosis-decision.md'` and `gnosis read --id 'gnosis://core/concepts/gnosis-directive.md'`. Map discoverable facts before asking the author; if the request changes repository purpose, invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/vault/refine-purpose.md' --pretty` first.
2. Check scope. Decompose independent subsystems into separately deliverable designs before refining details.
3. Ask one author-owned question per turn about outcome, constraints, and success criteria. Resolve each answer before asking its dependent question.
4. Present two or three viable approaches with trade-offs and a recommendation.
5. Present the design in sections proportionate to complexity. Cover boundaries, components, interfaces, data flow, failure behavior, and testing; revise until the author approves the whole design.
6. Self-review the approved design for placeholders, contradictions, ambiguous requirements, and scope that should be split.
7. Apply the Decision Concept Type boundary to every settled choice. Build each qualifying record from that definition and persist it with `gnosis write --type 'Gnosis Decision' --title '<title>' <draft-file>`. Leave non-qualifying design details for the directive.
8. Map every file to create, modify, or test and the responsibility it will hold. Follow existing patterns and introduce only boundaries required by the design.
9. Divide work into independently reviewable tasks. Each task carries its own red-green-refactor cycle and ends in a testable result.
10. Write exact steps: file paths, interfaces consumed and produced, test code or precise test behavior, commands, expected results, minimal implementation, full verification, and commits where the repository workflow uses them.
11. Build the implementation handoff from the Directive Concept Type definition and persist it with `gnosis write --type 'Gnosis Directive' --title '<title>' <draft-file>`. The directive is the single source of implementation requirements.
12. Self-review the directive for design coverage, placeholders, contradictory names or types, undefined dependencies, untestable criteria, and tasks that are too broad.
13. Offer the direct or delegated modes of [execute-directive](../execution/execute-directive.md), then run `gnosis validate --vault <root>`.

## Completion

The author has approved a coherent design, every qualifying durable decision has been recorded, and each independently deliverable design has one validated open directive with a complete executable task sequence and acceptance criteria. No implementation has started.

Adapted from `brainstorming` and `writing-plans`, analyzed in [Superpowers (obra)](../../../references/obra-superpowers.md).
