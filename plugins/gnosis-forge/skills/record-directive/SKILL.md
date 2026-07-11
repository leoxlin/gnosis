---
name: record-directive
description: Create a durable implementation handoff for automated or unattended agents. Use when the user explicitly requests a repository directive or selects the writing-plans repository process after approving a design.
---

# Record Directive

This skill is explicit-only. Valid triggers are an explicit author request for a directive or explicit selection of the `writing-plans` Repository Process after an approved design. Do not invoke it merely because implementation could benefit from a handoff.

1. Read relevant purpose and active decisions. Read `docs/concepts/repository-directive.md` only when classification or shape needs clarification.
2. Write `docs/repository/directives/<kebab-name>.md` with `status: open`, a concrete goal, bounded scope, material dependencies, and testable acceptance criteria sufficient for an automated agent.
3. When `writing-plans` invoked this skill, create one directive per independently deliverable design and include an ordered `# Implementation plan` with exact paths, interfaces, test-first steps, commands, expected results, and verification.
4. Do not implement the directive until the user selects an execution process.
5. Regenerate indexes only when `vault_index` is enabled, then validate the vault. Do not create a completion record; git carries delivered work.

Finish when another maintainer can implement and verify the work without replaying triage.
