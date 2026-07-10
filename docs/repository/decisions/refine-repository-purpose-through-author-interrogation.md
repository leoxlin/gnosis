---
type: Repository Decision
title: Refine repository purpose through author interrogation
description: Replace passive purpose recording with a one-question-at-a-time interrogation that reaches author-confirmed shared understanding before editing.
tags: [purpose, skills, workflow, interrogation, repository-decision]
timestamp: 2026-07-10T13:02:01Z
---

# Decision

Rename `record-purpose` to `refine-purpose` and require it to interrogate the author until both sides share a precise understanding of the repository's outcome, beneficiaries, sub-purposes, and boundaries. The skill may edit the purpose only after the author confirms that understanding.

# Context

Repository purpose is author-owned intent rather than a fact an agent can infer completely from source files. The former recorder allowed drafting before the purpose branches and their dependencies had been pressure-tested. The revised workflow adapts the one-question-at-a-time protocol from Matt Pocock's [grilling skill](https://github.com/mattpocock/skills/blob/main/skills/productivity/grilling/SKILL.md) while retaining the repository-purpose schema and single-record constraint.

# Alternatives considered

- **Keep `record-purpose` and require final confirmation** - rejected because confirmation after drafting does not force unresolved branches into the discussion.
- **Ask a batch of purpose questions** - rejected because dependent decisions should be resolved in order and each answer should shape the next question.
- **Depend on an external grilling skill at runtime** - rejected because `refine-purpose` must behave consistently wherever `gnosis-forge` is installed.

# Consequences

- The skill answers discoverable factual questions from repository evidence and reserves author-owned decisions for the author.
- It asks exactly one question per turn, recommends an answer with rationale, and waits before continuing.
- It explores every material purpose branch and obtains explicit shared-understanding confirmation before editing.
- `refine-purpose` is the current purpose workflow; `record-purpose` remains only as a historical name in earlier records.

# Related decisions

- [Name the knowledge-driven development bundle `gnosis-forge`](name-knowledge-driven-development-bundle-gnosis-forge.md)
- [Bootstrap `gnosis` knowledge first on OKF](bootstrap-knowledge-first.md)
- [`gnosis` purpose](../purpose.md)
