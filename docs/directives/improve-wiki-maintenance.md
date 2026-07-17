---
type: Directive
title: Improve wiki maintenance to obsidian-wiki parity
description: Ship tiered retrieval, cross-linking, and consolidation procedures plus provenance markers for the vault wiki.
status: done
---

# Goal

Implement [Adopt obsidian-wiki maintenance conventions](../decisions/adopt-wiki-maintenance-conventions.md): a tiered `query-vault`, a new `link-pages` cross-linking procedure, a consolidation-capable `maintain-vault`, and inline provenance markers in `ingest-knowledge`.

# Scope

- Replace `docs/procedures/vault/query-vault.md` and `docs/procedures/vault/maintain-vault.md` with the complete contents below.
- Create `docs/procedures/vault/link-pages.md` with the complete content below.
- Apply one exact edit to `docs/procedures/vault/ingest-knowledge.md`.
- No Go changes and no new special files: conventions live entirely in Procedure records, per the governing decision.

# Global constraints

- Follow [Adopt obsidian-wiki maintenance conventions](../decisions/adopt-wiki-maintenance-conventions.md).
- Procedure records must satisfy the Procedure contract and pass vault validation with zero warnings.
- All writes go through `gnosis apply page`; read procedures never write.
- The compact `QueryResult` contract is unchanged.

# Dependencies

- [Add the knowledge concept types](add-knowledge-concept-types.md) @ sha256:c7a98986dce1284500e4351ad0934b53004e4d4cf5784b4e9eb5a41dd77f0b4f — required contract: content Concept Types carry the `status`/`tier`/`superseded_by` lifecycle fields these procedures audit; evidence: `gnosis get pages gnosis://local/concepts/concept.md`; prerequisite must be `done`.
- [Add scoped agent memory](add-scoped-agent-memory.md) @ sha256:59f60bbbbc3e21bd2d619f6b33d3ad12441a365a25c1626d25f9713554fb85d7 — required contract: `docs/procedures/vault/recall.md` is effective, because the tiered `query-vault` links to it; evidence: `gnosis get procedures gnosis://local/procedures/vault/recall.md`; prerequisite must be `done`.

# Implementation plan

### Task 1: Tiered query-vault

**Load:** `docs/procedures/vault/query-vault.md` (current), `docs/references/obsidian-wiki.md` (cost ladder).
**Files:** modify `docs/procedures/vault/query-vault.md`.
**Interfaces:** consumes a question; produces a cited answer; read-only.

- [x] Replace `docs/procedures/vault/query-vault.md` with exactly this content, then apply: `gnosis apply page gnosis://local/procedures/vault/query-vault.md --filename docs/procedures/vault/query-vault.md`:

````markdown
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
````

- [x] Run `gnosis get procedures gnosis://local/procedures/vault/query-vault.md --full`; expect the contract loads.
- [x] Commit: `feat: tiered query-vault retrieval`.

### Task 2: The link-pages cross-linking procedure

**Load:** `docs/references/obsidian-wiki.md` (cross-linker), `internal/vault/links.go` (link formats: `relative` default, `absolute` optional), `docs/concepts/concept.md` (typed relationships).
**Files:** create `docs/procedures/vault/link-pages.md`.
**Interfaces:** consumes a page set (default: recently changed pages); produces link and relationship edits via `gnosis apply page`.

- [x] Write `docs/procedures/vault/link-pages.md` exactly as below, then apply: `gnosis apply page gnosis://local/procedures/vault/link-pages.md --filename docs/procedures/vault/link-pages.md`:

````markdown
---
type: Procedure
title: link-pages
description: Use when pages should be cross-linked to existing vault knowledge they mention without linking.
tags: [gnosis, vault]
invocation: model
---

# link-pages

`link-pages` discovers unlinked mentions of known pages and turns high-confidence ones into exact links and typed relationships, respecting the vault's configured link format.

## Inputs

- Vault configuration, especially `link_format` and `link_format_strict`, and agent rules.
- The effective page list with titles and aliases, and the bodies of the pages in scope.
- Concept Type definitions for typed `relationships` conventions.

## Process

1. Resolve the vault and its link format. List candidate targets with `gnosis get pages --vault <root> --fields uri,title` and collect each page's `aliases` on demand.
2. Scope the source pages (default: pages changed since the last pass or named by the author). Skip `index.md`, `log.md`, and pages under `documentation/`.
3. In each source body — never in frontmatter or code fences — find exact, case-sensitive, word-boundary mentions of known titles and aliases that are not already linked and not self-references.
4. Score each mention: a mention in the first paragraph or repeated mentions are high-confidence; a single passing mention late in the body is not. Link only high-confidence mentions, at most the first occurrence per target per page, and at most five new links per page.
5. Write links in the vault's configured format: relative Markdown links for `relative`, canonical `gnosis://` URIs for `absolute`.
6. Add a typed `relationships` entry (`extends`, `implements`, `uses`, `contradicts`, `derived_from`, `related_to`) only when the surrounding sentence states the relationship explicitly; otherwise report the suggestion instead of writing it.
7. Persist each changed page with `gnosis apply page '<URI>' --filename <draft-file>`. When `vault_log` is enabled, add one concise entry per pass to the nearest `log.md`.
8. Run `gnosis validate vault --vault <root>` and report inserted links, added relationships, and skipped low-confidence mentions.

## Completion

Every high-confidence unlinked mention in scope is linked in the configured format or reported; typed relationships are explicit-only; and vault validation passes.
````

- [x] Run `gnosis get procedures gnosis://local/procedures/vault/link-pages.md --full`; expect the contract loads.
- [x] Commit: `feat: add link-pages procedure`.

### Task 3: Consolidation in maintain-vault and provenance markers in ingest-knowledge

**Load:** `docs/procedures/vault/maintain-vault.md`, `docs/procedures/vault/ingest-knowledge.md` (current), `docs/references/obsidian-wiki.md` (lint/consolidate).
**Files:** modify `docs/procedures/vault/maintain-vault.md`, modify `docs/procedures/vault/ingest-knowledge.md`.
**Interfaces:** produces the consolidation-capable maintenance procedure and marker-carrying ingest.

- [x] Replace `docs/procedures/vault/maintain-vault.md` with exactly this content, then apply: `gnosis apply page gnosis://local/procedures/vault/maintain-vault.md --filename docs/procedures/vault/maintain-vault.md`:

````markdown
---
type: Procedure
title: maintain-vault
description: Use when auditing or repairing the integrity of a vault.
tags: [gnosis, vault]
invocation: model
---

# maintain-vault

`maintain-vault` repairs high-confidence structural and semantic problems and consolidates the wiki, while preserving uncertainty and author-owned meaning decisions.

## Inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Structural validation results and the affected pages.
- Concept Type definitions, linked records, and sources supporting conflicting claims.

## Process

1. Resolve the vault, read its agent rules and configuration, then run `gnosis validate vault --vault <root>` for the structural baseline.
2. Audit consolidation findings, using each type's `status`/`tier` lifecycle fields:
   - Orphans: pages with no inbound links that are not type definitions or entry points; rescue by linking from the nearest parent or report.
   - Near-duplicates: pages sharing one identity; merge into the richer page, set `status: archived` plus `superseded_by` on the loser, and repair inbound links.
   - Stale pages: `core`/`supporting` pages whose claims drifted from their sources; refresh or demote `tier` and report.
   - Contradictions: clusters of `^[ambiguous]` markers or conflicting claims; add explicit conflict callouts and report for author judgment.
   - Tag fragmentation: near-identical tags (case, plural, separator variants); normalize to the most-used form.
   - Broken typed `relationships`: invalid targets or relations the Concept Type does not sanction; repair or remove.
3. Apply high-confidence repairs in place through `gnosis apply page`. Preserve unknown metadata and source-backed disagreements; report identity or meaning conflicts that require author judgment.
4. Run `gnosis index vault --vault <root>` when `vault_index` is enabled and record material repairs only when `vault_log` is enabled.
5. Re-run `gnosis validate vault --vault <root>` and produce the consolidation report: every finding with its affected paths and the action taken or the author decision needed.

## Completion

Structural validation passes and every consolidation finding is repaired or reported with its affected paths and dispositions.
````

- [x] In `docs/procedures/vault/ingest-knowledge.md`, in `## Process` item 2, replace `Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference. When the request identifies one concept, retain exactly one concept identity.` with `Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference and tag claims inline: unmarked for extracted, \`^[inferred]\` for agent generalizations, \`^[ambiguous]\` for unresolved source disagreement. When the request identifies one concept, retain exactly one concept identity.`
- [x] Apply: `gnosis apply page gnosis://local/procedures/vault/ingest-knowledge.md --filename docs/procedures/vault/ingest-knowledge.md`.
- [x] Run `gnosis validate vault`; expect `status: valid`, `warnings: 0`. Run `gnosis get procedures --tags gnosis,vault`; expect `link-pages` listed alongside the updated procedures.
- [x] Commit: `feat: consolidate maintain-vault and provenance markers`.

# Acceptance criteria

- Tiered retrieval ships — run `gnosis get procedures gnosis://local/procedures/vault/query-vault.md --full`; expect the six-pass cost ladder and the `recall` route.
- Cross-linking ships — run `gnosis get procedures gnosis://local/procedures/vault/link-pages.md --full`; expect mention scoring, format rules, and the five-link cap.
- Consolidation ships — run `gnosis get procedures gnosis://local/procedures/vault/maintain-vault.md --full`; expect the six audit categories and the consolidation report; run `gnosis get procedures gnosis://local/procedures/vault/ingest-knowledge.md --full`; expect the `^[inferred]`/`^[ambiguous]` markers.
- Vault integrity holds — run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- No regressions — run `mise run checks`; expect all green.
