---
type: Repository Process
title: brainstorming
description: Use before creative repository work that adds features, components, functionality, or changed behavior.
---

# brainstorming

Brainstorming turns an idea into an author-approved design before implementation. Its hard gate is unchanged: implementation begins only after the design is understood and approved.

## Use when

- Creating a feature, component, integration, or product behavior.
- Changing existing behavior or choosing among materially different designs.
- A request is underspecified enough that implementation would encode author-owned assumptions.

Scale the conversation to the decision, but do not skip it because the change appears small.

## Knowledge inputs

- The [repository purpose](../purpose.md) and boundaries.
- Active decisions and concepts relevant to the proposed behavior.
- Existing implementation, tests, and repository conventions.
- Recent path-scoped history only when current knowledge and code leave a design choice unexplained.

## Process

1. Read the knowledge inputs and map discoverable facts before asking the author. If the request conflicts with or changes repository purpose, resolve that purpose with the author before designing.
2. Check scope. Decompose independent subsystems into separately deliverable designs before refining details.
3. Ask one author-owned question per turn about outcome, constraints, and success criteria. Resolve each answer before asking its dependent question.
4. Present two or three viable approaches with trade-offs and a recommendation.
5. Present the design in sections proportionate to complexity. Cover boundaries, components, interfaces, data flow, failure behavior, and testing; revise until the author approves the whole design.
6. Self-review the approved design for placeholders, contradictions, ambiguous requirements, and scope that should be split.
7. For every settled choice, apply the Repository Decision boundary. Record a decision only for a non-obvious durable choice whose rationale or constraints must outlive this task. Leave routine design details for the directive.
8. Invoke [writing-plans](writing-plans.md). Do not write a separate specification document.

## Completion

The author has approved a coherent design, every qualifying durable decision has been recorded, and `writing-plans` has begun translating the design into a directive. No implementation has started.

Adapted from [`brainstorming`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/brainstorming/SKILL.md), analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
