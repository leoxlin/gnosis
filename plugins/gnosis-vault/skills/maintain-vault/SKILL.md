---
name: maintain-vault
description: Audit and repair a gnosis OKF/LLM wiki. Use for vault linting, validation, stale indexes, broken links, orphan or duplicate concepts, and conflicting knowledge.
---

# Maintain Vault

1. Resolve the vault, read its agent rules and configuration, then run `gnosis validate -vault <root>` for the structural baseline.
2. Audit frontmatter, links, orphan pages, near-duplicate identities, stale summaries, and conflicting claims. Audit indexes or logs only when their matching options are enabled.
3. Apply high-confidence repairs in place. Preserve unknown metadata and source-backed disagreements; report identity or meaning conflicts that require author judgment.
4. Regenerate indexes when `vault_index` is enabled and record material repairs only when `vault_log` is enabled.
5. Re-run validation and summarize remaining semantic findings.

Finish when structural validation passes and every semantic finding is repaired or reported with its affected paths.
