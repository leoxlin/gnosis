---
type: Reference
title: Superpowers (obra)
description: Analysis of the Superpowers agentic skills framework and its opinionated, cross-harness software-development methodology.
resource: https://github.com/obra/Superpowers
source_sha: d884ae04edebef577e82ff7c4e143debd0bbec99
analyzed_at: 2026-07-10T13:55:12Z
tags: [agent-skills, software-development, methodology, tdd, subagents, code-review, cross-harness, provenance]
timestamp: 2026-07-10T13:55:12Z
---

# Superpowers (obra)

This reference analyzes [`obra/Superpowers`](https://github.com/obra/Superpowers) at commit [`d884ae04edebef577e82ff7c4e143debd0bbec99`](https://github.com/obra/Superpowers/commit/d884ae04edebef577e82ff7c4e143debd0bbec99), released as version 6.1.1. Superpowers is an MIT-licensed collection of composable agent skills plus the bootstrap and platform adapters that turn those skills into an opinionated software-development methodology.

## What it is

Superpowers treats software delivery as a sequence of mandatory agent behaviors rather than a menu of optional prompts. Its bootstrap tells the agent to check for relevant skills before responding or acting. Individual skills then govern design, planning, workspace isolation, implementation, testing, debugging, review, verification, and branch completion.

The project combines three things:

1. **A methodology** — a gated path from an initial idea through an approved design and plan to verified delivery.
2. **A skill library** — markdown instructions that encode each stage and the conditions that trigger it.
3. **Cross-harness packaging** — thin integrations that expose the same skills through Claude Code, Codex, Cursor, Kimi Code, OpenCode, Pi, and other agent harnesses.

## Core model

| Principle | How Superpowers implements it |
|---|---|
| **Skills before action** | `using-superpowers` requires the agent to invoke any potentially relevant skill before even asking a clarifying question. |
| **Design before implementation** | `brainstorming` explores intent and alternatives, presents a design incrementally, and requires human approval before planning. |
| **Plans as executable specifications** | `writing-plans` breaks an approved design into small tasks with exact paths, commands, expected results, and verification. |
| **Isolation before change** | `using-git-worktrees` creates or recognizes an isolated worktree before plan execution. |
| **Tests before production code** | `test-driven-development` enforces red-green-refactor and requires observing the intended test failure. |
| **Root cause before fixes** | `systematic-debugging` forbids proposing a fix before investigating the failure's cause. |
| **Review during execution** | `subagent-driven-development` gives each task to a fresh implementer and follows it with specification and quality review. |
| **Evidence before claims** | `verification-before-completion` requires fresh command output before an agent says work is fixed, passing, or complete. |
| **Explicit delivery choice** | `finishing-a-development-branch` verifies the suite, detects the environment, and asks the human whether to merge, open a pull request, keep, or discard the work. |

## Development lifecycle

The default lifecycle is deliberately sequential:

1. `brainstorming` turns the request into a human-approved design document.
2. `using-git-worktrees` prepares an isolated workspace.
3. `writing-plans` turns the design into a detailed implementation plan.
4. `subagent-driven-development` executes the plan in the current session, or `executing-plans` executes it in a separate session.
5. `test-driven-development`, `systematic-debugging`, and the review skills constrain implementation behavior.
6. `verification-before-completion` prevents unsupported completion claims.
7. `finishing-a-development-branch` hands the final integration decision back to the human.

This forms a process state machine even though it is implemented primarily as natural-language instructions. Skill descriptions provide triggers, skill bodies define entry and exit conditions, and explicit handoffs select the next skill.

## Architecture and portability

Superpowers separates stable process content from harness-specific delivery:

| Layer | Responsibility |
|---|---|
| **Skills** | Harness-neutral actions such as reading a file, invoking a skill, tracking tasks, or dispatching an agent. The canonical content lives under `skills/`. |
| **Tool mapping** | Translates those actions into each harness's actual tool names and constraints. |
| **Bootstrap** | Injects `using-superpowers` at session start so the model knows the skills exist and must be checked. |

The [porting guide](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/docs/porting-to-a-new-harness.md) calls automatic session-start injection the hard requirement. A platform without it cannot provide the intended automatic behavior. Skill discovery, file access, editing, and shell execution are also fundamental; subagents, task tracking, and web access can degrade to documented fallbacks.

Platform adapters use the host's own install mechanism rather than modifying a user's global configuration. Manifests and adapters live beside the canonical skills, including [the Codex plugin manifest](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/.codex-plugin/plugin.json), Claude and Cursor plugin manifests, a Kimi Code manifest with an inline tool map, an OpenCode plugin, and a Pi extension.

## Skill library

The analyzed revision contains fourteen top-level skills:

| Area | Skills |
|---|---|
| **Bootstrap and design** | `using-superpowers`, `brainstorming`, `writing-plans` |
| **Implementation** | `using-git-worktrees`, `executing-plans`, `subagent-driven-development`, `dispatching-parallel-agents` |
| **Engineering discipline** | `test-driven-development`, `systematic-debugging`, `verification-before-completion` |
| **Review and delivery** | `requesting-code-review`, `receiving-code-review`, `finishing-a-development-branch` |
| **Meta-methodology** | `writing-skills` |

The skills are intentionally assertive. They name common rationalizations, define red flags, and state forbidden shortcuts. This is behavior-shaping documentation: the wording is part of the implementation, not merely an explanation of it.

## Subagent-driven development

The [`subagent-driven-development` skill](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/subagent-driven-development/SKILL.md) is the framework's most developed orchestration pattern:

- A fresh implementer receives one task with curated context.
- The implementer tests, changes, commits, self-reviews, and writes a durable report.
- A separate reviewer checks both specification compliance and code quality from a generated diff package.
- Critical and important findings return to a fixer and then to re-review.
- A final reviewer evaluates the whole branch.

Task briefs, reports, review packages, and a progress ledger are passed as files instead of being copied through prompts. This reduces accumulated context, gives each agent a bounded role, and allows the controller to recover after conversation compaction. Dependent implementation steps remain sequential; parallel agents are reserved for genuinely independent work.

## Testing skills as executable process

Superpowers applies TDD to its own instructions. The [`writing-skills` material](https://github.com/obra/Superpowers/tree/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/writing-skills) recommends:

1. Run a pressure scenario without the skill and record how an agent fails or rationalizes a shortcut.
2. Add the minimum instructions needed to prevent that behavior.
3. Run the same scenario with the skill.
4. Capture new loopholes and refine the wording until the behavior remains compliant under pressure.

The repository separates deterministic plugin tests from behavior evaluations. Shell and Node tests cover adapters, hooks, packaging, and integrations. Real-agent behavior is evaluated with the separate `superpowers-evals` drill harness referenced by the project's [testing documentation](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/docs/testing.md); those slower evaluations are not part of normal CI at the analyzed revision.

## Relevance to `gnosis`

Superpowers is a useful complement and comparison point for [`gnosis`](../purpose.md):

- `gnosis` grounds durable purpose, concepts, sources, and decisions; Superpowers coordinates action from a request to delivered code.
- Both use human-readable skills as a portable interface for agents and keep canonical instructions independent of a particular model or harness.
- Superpowers demonstrates how skills can form a larger workflow through trigger conditions, explicit handoffs, human approval gates, verification gates, and file-based agent handoffs.
- Its pressure-testing method offers a way to test whether process skills change agent behavior, beyond checking their syntax or structure.
- Its separation of canonical skills, tool maps, and bootstrap adapters is a concrete portability pattern for multi-harness skill bundles.
- Its progress ledger solves recovery for active task orchestration, while `gnosis` keeps that ledger as transient execution scratch and leaves routine delivery history to git and CI.

`gnosis` adapts the analyzed workflows into composable [Procedure](../concepts/procedure.md) records plus the `using-gnosis` runtime gateway. Planning separates requirements, simple and complex directive creation, independent purpose/decision and engineering reviews, and finalization. Execution and code review remain unified. The gateway delegates smallest-chain selection to a fresh read-only subagent. The durable boundary is recorded in [Consolidate runtime adapters in the `gnosis` plugin](../decisions/consolidate-runtime-adapters-in-gnosis-plugin.md).

## Trade-offs and cautions

- **Strong processes create overhead.** Brainstorming, worktrees, TDD, and repeated reviews improve discipline but can be disproportionate for small or non-code changes; process invocation modes and smallest-chain selection keep that overhead bounded.
- **Prompt enforcement is not hard enforcement.** The bootstrap and emphatic language make compliance more likely, but the model can still misunderstand, skip, or conflict with instructions.
- **Portability requires adapters.** Canonical skills are shared, but every harness still needs packaging, tool translation, bootstrap delivery, and integration testing.
- **The full workflow assumes rich agent capabilities.** Subagent-driven development works best with multi-agent support; fallbacks are necessarily less isolated and less independently reviewed.
- **Behavior evaluation is costly.** Real-agent pressure tests are more meaningful than static checks, but they take longer, consume model resources, and may vary across models and harness versions.
- **Prescriptiveness can conflict with local policy.** Repository instructions and direct human requests take precedence, so integrating Superpowers with another methodology requires an explicit rule hierarchy.

## Key sources

- [Superpowers repository](https://github.com/obra/Superpowers)
- [Analyzed commit](https://github.com/obra/Superpowers/commit/d884ae04edebef577e82ff7c4e143debd0bbec99)
- [README at the analyzed commit](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/README.md)
- [`using-superpowers`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/using-superpowers/SKILL.md)
- [`brainstorming`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/brainstorming/SKILL.md)
- [`test-driven-development`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/test-driven-development/SKILL.md)
- [`systematic-debugging`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/systematic-debugging/SKILL.md)
- [`subagent-driven-development`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/subagent-driven-development/SKILL.md)
- [`writing-skills`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/writing-skills/SKILL.md)
- [Porting guide](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/docs/porting-to-a-new-harness.md)
- [Testing documentation](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/docs/testing.md)
