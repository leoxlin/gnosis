---
type: Repository Decision
title: Keep repository context minimal
description: Preserve purpose and durable decisions, use directives only for explicit automation handoffs, and leave delivery history to git.
---

> Superseded by [Make repository processes knowledge](make-repository-processes-knowledge.md). Its git, index, and log constraints remain; its default-context and directive-trigger rules are replaced.

# Decision

Keep the default `gnosis-forge` context to repository purpose, relevant concepts, and active durable decisions. Remove Repository Delta. Create Repository Directives only when an author explicitly requests a handoff for automated-agent execution. Leave routine changes, verification, and completion history to git and CI.

Make `index.md` and `log.md` optional vault artifacts controlled by `vault_index` and `vault_log`. Keep both options enabled by default for compatibility, but disable them for this repository.

# Why

Decisions preserve intent that implementation and git cannot reliably reconstruct. Routine delta records, indexes, and logs duplicate repository history, add maintenance work, and consume agent context without participating in the normal reasoning path.

# Constraints

- Agents consult path-scoped git history only when current knowledge, code, and tests do not explain a choice.
- `record-directive` cannot trigger implicitly.
- Disabled indexes and logs are not created by setup and are not required by validation.
- Existing users retain index and log behavior unless they opt out in `gnosis.toml`.
