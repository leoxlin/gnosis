---
name: query-vault
description: Answer questions from a gnosis OKF/LLM wiki. Use when querying vault knowledge, tracing linked concepts, comparing recorded claims, or identifying knowledge gaps.
---

# Query Vault

1. Resolve the vault and read its configuration and agent rules.
2. Before opening concept pages, run `gnosis graph-query -vault <root> -pretty "<question>"`.
   - If `index_only` is true and a candidate exists, answer from its description and cite its page without opening the body.
   - For a non-empty `path`, use the returned chain and open only the listed `should_read` pages when the link structure alone does not explain the relationship.
   - Otherwise, inspect only the returned `should_read` pages, starting with a targeted section search before a full read.
   - If no candidates are returned, report the knowledge gap instead of scanning every page.
3. If the `gnosis` CLI is unavailable, fall back to the vault index when `vault_index` is enabled, then search titles, descriptions, tags, and filenames before opening pages.
4. Answer from recorded knowledge and cited sources. Label synthesis, conflicts, and missing evidence clearly.
5. Cite the concept paths that support the answer.

Keep the query read-only. Finish when the answer is grounded in the vault, material conflicts or gaps are visible, and no files changed.
