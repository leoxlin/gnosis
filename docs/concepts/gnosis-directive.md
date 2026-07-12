---
type: Concept Type
title: Gnosis Directive
description: An explicitly requested durable implementation handoff.
path: gnosis/directives
---

# Gnosis Directive

A **Gnosis Directive** is a bounded handoff for later automated or unattended execution.

`draft` is planning-only; finalization alone changes it to executable `open`.

## Use this for

- Explicitly requested work that needs durable scope and observable acceptance criteria.

Do not create one implicitly for ordinary implementation, task tracking, or completed work.

## Minimum record

- `status`, `# Goal`, `# Scope`, and evidence-bearing `# Acceptance criteria`.
- Multi-step work adds `# Implementation plan`; prerequisites add `# Dependencies`. Directive dependencies bind links, revisions, and supplied contracts.
- Complex work adds only execution-relevant architecture, stack, and global constraints; name affected components and justify new libraries.
- Each task names exact files, interfaces, required paths/process URIs, atomic steps with complete code or patches, commands, expected results, and a commit.
- Behavior tasks use red-green-refactor plus focused and surrounding green; other tasks use exact validation.
- Add `# Purpose/Decision Changes` only for persisted changes, with old→new URI/revisions and effects.

Omit empty optional sections. Plans contain no placeholders.

## Schema

```markdown
---
type: Gnosis Directive
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
```
