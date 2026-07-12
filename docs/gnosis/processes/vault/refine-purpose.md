---
type: GnosisProcess
title: refine-purpose
description: Use when the repository purpose must be created or changed through author-confirmed understanding.
tags: [gnosis-vault]
invocation: model
effects: [vault-write]
use_when:
  - Creating the repository purpose.
  - Changing its outcome, beneficiaries, sub-purposes, or boundaries.
  - Resolving a material ambiguity in author-owned repository intent.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# refine-purpose

`refine-purpose` establishes shared, author-confirmed understanding before it changes the repository purpose.

## Knowledge inputs

- The current repository purpose, when it exists.
- Relevant active decisions, concepts, implementation facts, and repository instructions.
- Required record shape: `type`, `title`, `description`, `# Purpose`, optional `# Sub-purposes`, and `# Boundaries`; exclude architecture, plans, milestones, and tasks.

## Process

1. Read the current purpose with `gnosis read --id '<purpose URI>'` when it exists. Gather other discoverable facts and distinguish them from author-owned intent.
2. Ask exactly one author-owned question at a time. Recommend an answer with rationale, then wait for the response before asking a dependent question.
3. Explore every material branch of the intended outcome, beneficiaries, sub-purposes, and boundaries until no unresolved author-owned choice remains.
4. Summarize the proposed purpose and obtain explicit confirmation that the author and agent share the same understanding.
5. Only after confirmation, build the single purpose record in the required shape and persist it with `gnosis write --type 'Gnosis Purpose' --title '<title>' <draft-file>`.
6. Run `gnosis validate --vault <root>`.

## Completion

The author has explicitly confirmed a precise shared understanding, and the validated Gnosis Purpose record reflects it.
