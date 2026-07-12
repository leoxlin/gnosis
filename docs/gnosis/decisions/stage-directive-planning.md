---
type: Gnosis Decision
title: Stage directive planning by complexity and review concern
description: Load only the planning stage and knowledge required for each directive.
---

# Decision

Route every requested plan through [requirements refinement](../processes/planning/refining-requirements.md). [Simple work](../processes/planning/creating-simple-directive.md) creates one draft and proceeds to [finalization](../processes/planning/finalizing-directives.md). [Complex work](../processes/planning/creating-complex-directives.md) creates PR-sized dependent drafts, then uses separate read-only [purpose/decision](../processes/planning/review-directive-purpose-decisions.md) and [engineering](../processes/planning/review-directive-engineering.md) reviews before finalization.

Draft directives are not executable. Finalization alone changes them to `open`.

# Why

Small changes avoid planning overhead. Complex plans gain independent scrutiny while exact process and directive URIs keep agent context bounded.

# Constraints

- The controller owns writes and feedback disposition.
- Every review binds exact directive, repository, purpose, decision, and source revisions.
- Material rewrites repeat affected reviews; dependency-contract changes also re-review dependents.
- Purpose or decision changes are author-settled, persisted, and linked from `# Purpose/Decision Changes`.
