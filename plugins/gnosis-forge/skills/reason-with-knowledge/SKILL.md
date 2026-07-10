---
name: reason-with-knowledge
description: Reason about repository work from its purpose, durable decisions, concepts, current implementation, and git history. Use when planning or implementing changes with knowledge-first context.
---

# Reason with Knowledge

1. Read `docs/repository/purpose.md`, then only the active decisions and concept pages relevant to the task.
2. Treat recorded knowledge as intent and constraints. Inspect current code and tests for implementation truth.
3. Use path-scoped `git log`, `git show`, or `git blame` only when current knowledge and implementation do not explain a choice.
4. Record only non-obvious, durable choices with `record-decision`. Never create a routine completion record. Never invoke `record-directive` unless the user explicitly requests a directive for automated-agent work.
5. Keep implementation and knowledge aligned. Follow `gnosis.toml`; regenerate indexes only when `vault_index` is enabled, and do not require a log when `vault_log` is disabled.
6. Run `mise run checks` before handoff.

Finish when the change agrees with purpose and active decisions, any new durable constraint is recorded, and the repository check passes.
