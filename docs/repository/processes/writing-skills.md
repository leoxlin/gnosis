---
type: Repository Process
title: writing-skills
description: Use when creating, editing, or validating a runtime skill or its repository-owned process knowledge.
---

# writing-skills

Writing skills applies red-green-refactor to agent behavior. Repository Process pages hold repository-owned workflow knowledge; packaged skills provide only the invocation and execution adapter that must exist at runtime.

## Use when

- Creating or changing a packaged skill.
- Moving reusable workflow detail from a skill into a Repository Process record.
- Testing whether instructions change agent behavior under realistic pressure.
- Deciding whether knowledge belongs in repository documentation, skill packaging, or deterministic tooling.

## Knowledge inputs

- The [repository purpose](../purpose.md) and active decisions about knowledge and plugin boundaries.
- The [Repository Process](../../concepts/repository-process.md) definition and any existing process with the same identity.
- Current skill-system metadata rules, runtime packaging, validation tools, and agent instructions.
- Concrete failure scenarios, baseline outputs, and prior evaluation evidence.

## Process

1. Classify the content before authoring:
   - Put a repeatable repository-owned workflow in one Repository Process page.
   - Package a skill when runtime invocation, tool integration, or a portable cross-repository procedure is required.
   - Implement a deterministic check in code when judgment is unnecessary.
2. Define concrete usage examples and the behavior that currently fails. For an existing process, read its durable record before changing the adapter.
3. **Red:** Run realistic fresh-context scenarios without the new guidance and capture the observed failure or rationalization. If the baseline does not fail, do not add guidance for that hypothetical problem.
4. **Green:** Write the smallest instruction change that corrects the observed behavior. Keep ordered runtime steps in the skill and detailed repository workflow knowledge in the Repository Process page, with one explicit context pointer from the skill.
5. Make invocation deliberate. User-invoked skills declare that policy and keep human-facing metadata concise; model-invoked skills describe concrete trigger conditions without summarizing a shortcut version of the workflow.
6. Preserve one source of truth, progressive disclosure, observable completion criteria, and only the guardrails demonstrated necessary by testing. Keep examples and support files only when they materially improve execution.
7. Run the same scenarios with the skill or process present and confirm the target behavior. Refine against newly observed loopholes without accumulating unrelated prose.
8. Validate skill frontmatter, agent metadata, plugin manifests, links, and repository checks. For local plugin development, use the supported cachebuster and reinstall flow rather than editing installation state by hand.

## Completion

The repository process and runtime adapter have non-overlapping responsibilities, the behavior change has red-and-green evaluation evidence, every packaged artifact validates, and the repository's full checks pass.

Adapted from `writing-skills`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
