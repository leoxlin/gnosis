---
type: Repository Delta
title: Refine purpose skill
description: Renamed the purpose skill and made author interrogation a prerequisite for repository-purpose edits.
tags: [delta, plugins, skills, purpose, workflow]
timestamp: 2026-07-10T13:02:01Z
status: completed
---

# Fulfilled directives

- User request to rename `record-purpose` to `refine-purpose` and adopt the grilling interrogation workflow.

# Change summary

- Renamed the canonical skill and its discovery links to `refine-purpose`.
- Made the skill investigate repository facts, resolve purpose branches through one-question-at-a-time author interrogation, recommend an answer for every question, and wait for shared-understanding confirmation before editing.
- Updated the skill's Codex interface metadata.

# Verification

- `refine-purpose` passes `quick_validate.py`.
- Skill discovery links resolve to the renamed canonical directory.
- `mise run checks`

# Deviations

None.

# Related decisions

- [Refine repository purpose through author interrogation](../decisions/refine-repository-purpose-through-author-interrogation.md)
