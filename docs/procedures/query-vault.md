---
type: Procedure
title: query-vault
description: Use when answering a question from recorded vault knowledge.
tags: [gnosis, vault]
invocation: model
---

# query-vault

`query-vault` answers from the smallest relevant set of recorded knowledge, following a cost ladder so query cost stays flat as the vault grows. It never writes.

## Inputs

- Vault configuration and agent rules.
- Knowledge-query results, candidate identity and provenance, and only the concept pages they identify as necessary.
- Citations and source material recorded by those concept pages.

## Process

1. Resolve the vault and read its configuration and agent rules. Route questions about preferences, persona, or agent memories to [recall](recall.md) instead.
2. **Catalog pass.** When `vault_index` is enabled, read the root `index.md` and use titles and descriptions to shortlist candidates before any search.
3. **Lexical pass.** Run `gnosis search knowledge --backend lexical --vault <root> "<question>"`.
   - If `index_only` is true and a candidate exists, answer from its description and cite its page without opening the body.
   - For a non-empty `path`, use the returned chain and open only the listed `should_read` pages when the link structure alone does not explain the relationship.
   - If no candidates are returned, continue to the next pass before declaring a gap.
4. **Vector pass.** Only when semantic retrieval is configured, run `gnosis search knowledge --backend vector --vault <root> "<question>"` and merge candidates by URI with the lexical results.
5. **Read pass.** Open at most the top three `should_read` candidates with `gnosis get pages '<URI>' --full`, preferring `tier: core` pages; grep a relevant section before reading whole pages when a candidate is long.
6. **Multi-hop pass.** For exact relationship questions, use `gnosis graph neighbors '<URI>' --vault <root>` or `gnosis graph path '<FROM_URI>' '<TO_URI>' --vault <root>` with bounded depth.
7. If the `gnosis` command is unavailable, fall back to the vault index when `vault_index` is enabled, then search titles, descriptions, tags, and filenames before opening pages.
8. Answer from recorded knowledge and cited sources. Label agent synthesis `^[inferred]`, unresolved conflicts `^[ambiguous]`, and report knowledge gaps instead of scanning every page.
9. Cite the concept paths that support the answer.

## Completion

The answer is grounded in the vault, the cost ladder was respected (no full-vault scans), material conflicts or gaps are visible, and no files changed.
