---
type: Procedure
title: query-vault
description: Use when answering a question from recorded vault knowledge.
tags: [gnosis-vault]
invocation: model
use_when:
  - Answering a question from a vault.
  - Tracing linked concepts or comparing recorded claims.
  - Identifying a gap in the knowledge currently recorded.
---

# query-vault

`query-vault` answers from the smallest relevant set of recorded knowledge, preserving provenance, uncertainty, conflicts, and gaps.

## Knowledge inputs

- Vault configuration and agent rules.
- Knowledge-query results, candidate identity and provenance, and only the concept pages they identify as necessary.
- Citations and source material recorded by those concept pages.

## Process

1. Resolve the vault and read its configuration and agent rules.
2. Before opening concept pages, run `gnosis query graph --vault <root> --pretty "<question>"`.
   - If `index_only` is true and a candidate exists, answer from its description and cite its page without opening the body.
   - For a non-empty `path`, use the returned chain and open only the listed `should_read` pages when the link structure alone does not explain the relationship.
   - For an exact relationship question, use `gnosis graph neighbors --vault <root> --id '<id-or-URI>' --pretty` or `gnosis graph path --vault <root> --from '<id-or-URI>' --to '<id-or-URI>' --pretty`.
   - Otherwise, inspect only the returned `should_read` pages with `gnosis read '<URI>'`.
   - If no candidates are returned, report the knowledge gap instead of scanning every page.
3. If the `gnosis` command is unavailable, fall back to the vault index when `vault_index` is enabled, then search titles, descriptions, tags, and filenames before opening pages.
4. Answer from recorded knowledge and cited sources. Label synthesis, conflicts, and missing evidence clearly.
5. Cite the concept paths that support the answer.

## Completion

The answer is grounded in the vault, material conflicts or gaps are visible, and no files changed.
