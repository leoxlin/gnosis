---
type: Decision
title: Stage directive planning by complexity and review concern
description: Load only the planning stage and knowledge required for each directive.
---

# Decision

Route every requested plan through [requirements refinement](../procedures/planning/refining-requirements.md). [Directive creation](../procedures/planning/creating-directives.md) selects the simple or complex branch: simple work creates one draft and proceeds to [finalization](../procedures/planning/finalizing-directives.md), while complex work creates PR-sized dependent drafts and runs independent purpose/decision and engineering passes through [directive review](../procedures/planning/reviewing-directives.md) before finalization.

Draft directives are not executable. Finalization alone changes them to `open`.

# Why

Small changes avoid planning overhead. Complex plans gain independent scrutiny while exact process and directive URIs keep agent context bounded.

# Constraints

- The controller owns writes and feedback disposition.
- Every review binds exact directive, repository, purpose, decision, and source revisions.
- Material rewrites repeat affected reviews; dependency-contract changes also re-review dependents.
- Purpose or decision changes are author-settled, persisted, and linked from `# Purpose/Decision Changes`.
