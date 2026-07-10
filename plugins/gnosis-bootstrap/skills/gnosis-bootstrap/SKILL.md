---
name: gnosis-bootstrap
description: Work on the gnosis codebase from its recorded knowledge. Use for repository conventions, OKF bundle changes, implementation decisions, and knowledge-first SDLC work.
---

# Gnosis Bootstrap

1. Read `docs/repository/purpose.md`, then only the `docs/repository/decisions/` and `docs/concepts/` pages relevant to the task.
2. Treat `docs/` as design intent. Record new purpose, decisions, directives, or completed deltas with the matching `record-repository-*` skill.
3. Keep implementation and knowledge aligned; every concept page needs parseable YAML frontmatter with a non-empty `type`.
4. Follow `gnosis.toml` for vault roots and link format. Preserve reserved `index.md` and `log.md` files.
5. Run `mise run check` before handoff.

Finish when the change agrees with recorded purpose and decisions, its durable knowledge is captured, and the repository check passes.
