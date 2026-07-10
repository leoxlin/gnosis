---
name: record-directive
description: Create a durable implementation handoff for automated or unattended agents. Use only when the user explicitly invokes this skill or explicitly asks to record a repository directive; never infer it from ordinary planning or implementation work.
---

# Record Directive

This skill is explicit-only. Do not invoke it merely because a task could benefit from a plan or handoff.

1. Read relevant purpose and active decisions. Read `docs/concepts/repository-directive.md` only when classification or shape needs clarification.
2. Write `docs/repository/directives/<kebab-name>.md` with `status: open`, a concrete goal, bounded scope, material dependencies, and testable acceptance criteria sufficient for an automated agent.
3. Do not implement the directive unless the user separately asks for execution.
4. Regenerate indexes only when `vault_index` is enabled, then validate the vault. Do not create a completion record; git carries delivered work.

Finish when another maintainer can implement and verify the work without replaying triage.
