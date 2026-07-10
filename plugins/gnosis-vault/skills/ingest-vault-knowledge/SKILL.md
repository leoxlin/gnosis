---
name: ingest-vault-knowledge
description: Integrate sources, notes, conversations, or findings into a gnosis OKF/LLM wiki. Use when ingesting, capturing, compiling, or refreshing vault knowledge.
---

# Ingest Vault Knowledge

1. Resolve the vault from `gnosis.toml` or the current bundle. Read its agent rules, root `index.md` and `log.md`, relevant concept definitions, and nearby pages.
2. Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference.
3. Integrate by concept identity. Update matching pages and create only the smallest useful set of new pages. Preserve unknown frontmatter and follow the configured link format.
4. Keep claims traceable to their source. Surface contradictions or ambiguous identity instead of silently choosing a side.
5. Regenerate affected indexes with `gnosis index -vault <root>` and add a concise, newest-first entry to the nearest `log.md`.
6. Run `gnosis validate -vault <root>`.

Finish when every retained claim has a durable home and provenance, affected navigation reflects the result, and validation passes.
