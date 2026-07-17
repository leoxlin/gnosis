---
type: ConceptType
title: Directive
description: An explicitly requested durable implementation handoff.
path: directives
---

# Directive

A **Directive** is a bounded handoff for later automated or unattended execution.

`draft` is planning-only; finalization alone changes it to executable `open`.

By convention, the Directive records lives at `gnosis://<vault>/directives/`.

## Use this for

- Explicitly requested work that needs durable scope and observable acceptance criteria.

Do not create one implicitly for ordinary implementation, task tracking, or completed work.

## Minimum record

- `status`, `# Goal`, `# Scope`, and evidence-bearing `# Acceptance criteria`.
- Multi-step work adds `# Implementation plan` with `### Task N:` sections; every task carries checkbox steps (`- [ ]`) so progress is derived, never restated. Prerequisites add `# Dependencies`. Directive dependencies bind links, revisions, and supplied contracts.
- Behavior acceptance criteria use `#### Scenario: <name>` blocks with bold `**WHEN**` and `**THEN**` bullets (optional `**GIVEN**`/`**AND**`); when any scenario is present, every scenario must follow the grammar.
- When the work changes Purpose or Decision records, add `# Purpose/Decision Changes` declaring the deltas as `## Added`, `## Modified`, or `## Removed` subsections naming the exact target records; `## Modified` carries the full replacement text or exact section edits.
- Complex work adds only execution-relevant architecture, stack, and global constraints; name affected components and justify new libraries.
- Each task names exact files, interfaces, required paths/process URIs, atomic steps with complete code or patches, commands, expected results, and a commit.
- Behavior tasks use red-green-refactor plus focused and surrounding green; other tasks use exact validation.

Omit empty optional sections. Plans contain no placeholders.

## Lifecycle

- Require an explicitly requested durable implementation handoff. Creation invokes [planning-directives](../procedures/development/planning-directives.md), which owns drafting, review, persistence, and the `draft` to `open` transition.
- Apply only non-semantic corrections in place while preserving unknown metadata and status. A change to the goal, scope, dependencies, implementation plan, acceptance criteria, or declared deltas returns an unfinished Directive to `draft` and invokes `planning-directives` with its current URI, revision, original requirements, and proposed change.
- Status follows `draft` → `open` → `blocked|done`, with `blocked` → `draft` only after evidence shows the blocker is resolved. Never assign status from assertion alone: planning finalization owns `draft` → `open`, and [implementing-directive](../procedures/development/implementing-directive.md) owns evidence-backed `open` → `blocked|done` for exactly one directive per invocation. Checkbox progress is derived state and never a status. Replanning owns `blocked` → `draft` and must revalidate the requirements, dependencies, and plan before finalization reopens it. Reject every other transition.
- A completed Directive remains historical until [maintain-intents](../procedures/development/maintain-intents.md) archives it: its declared deltas merge into the living Purpose and Decision records, its still-durable choices are compacted into Decisions, and the Directive is removed. Create a new Directive for new or changed work instead of reopening it.
- Prefer correction or retention after a Directive has governed work. Delete only a confirmed local duplicate or invalid `draft` after tracing inbound links and dependency history, obtaining explicit approval for the exact deletion, and repairing or intentionally removing every inbound reference. Report imported or bundled records to their owning vault.

## Schema

```markdown
---
type: Directive
title: <name>
description: <result>
status: <draft | open | blocked | done>
---

# Goal
# Architecture
# Tech stack
# Global constraints
# Scope
# Dependencies

- <dependency link> @ <revision> — <required contract and evidence>

# Purpose/Decision Changes

## Added
## Modified
## Removed

# Implementation plan

### Task N: <deliverable>
**Load:** <exact paths/sections and process URIs>
**Files:** <create, modify, test: exact paths>
**Interfaces:** <consumes and produces: exact signatures>

- [ ] <one 2–5 minute action with complete code or patch>
- [ ] Run `<command>`; expect `<result>`.
- [ ] Commit: `<message>`.

# Acceptance criteria

- <observable outcome> — run/inspect <exact check>; expect <evidence>.

#### Scenario: <name>

- **WHEN** <trigger>
- **THEN** <observable result>
```
