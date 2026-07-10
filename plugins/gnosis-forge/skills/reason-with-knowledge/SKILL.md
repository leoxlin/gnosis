---
name: reason-with-knowledge
description: Reason about repository work from recorded purpose, decisions, concepts, directives, and deltas. Use when planning or implementing changes with knowledge-first context.
---

# Reason with Knowledge

1. Read `docs/repository/purpose.md`, then only the `docs/repository/decisions/` and `docs/concepts/` pages relevant to the task.
2. Treat recorded knowledge as intent and constraints. Trace the concepts and decisions that bear on the work before changing it.
3. Record new purpose, decisions, directives, or completed deltas with the matching `record-*` skill.
4. Keep implementation and knowledge aligned; every concept page needs parseable YAML frontmatter with a non-empty `type`.
5. Follow `gnosis.toml` for vault roots and link format. Preserve reserved `index.md` and `log.md` files.
6. Run `mise run checks` before handoff.

Finish when the change agrees with recorded purpose and decisions, its durable knowledge is captured, and the repository check passes.
