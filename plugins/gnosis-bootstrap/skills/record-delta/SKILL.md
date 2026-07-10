---
name: record-delta
description: Trace completed repository work. Use after implementation when changes, fulfilled directives, verification, or deviations should remain durable.
---

# Record Delta

1. Read `docs/concepts/repository-delta.md` when classification or schema needs clarification, plus the fulfilled directive and relevant decisions.
2. Derive the record from the delivered diff and verification evidence; write it to `docs/repository/deltas/<kebab-name>.md`.
3. Link fulfilled directives, record deviations, update affected indexes and `docs/log.md`, then validate the vault.

Finish when the recorded change and verification match the delivered repository state.
