---
type: Reference
title: Obsidian Wiki (Ar9av)
description: Deep analysis of the obsidian-wiki framework — an agent-skill system for building and maintaining an OKF-compatible LLM wiki on top of Obsidian.
resource: https://github.com/Ar9av/obsidian-wiki
source_sha: e3ead02a320f7959317084e4377b414f737b2890
analyzed_at: 2026-07-09T03:24:27Z
tags: [obsidian, wiki, llm, knowledge-base, agent-skills, okf, provenance]
timestamp: 2026-07-09T03:24:27Z
---

# Obsidian Wiki (Ar9av)

This reference captures the architecture, conventions, and operational ideas from [`Ar9av/obsidian-wiki`](https://github.com/Ar9av/obsidian-wiki) at commit [`e3ead02a320f7959317084e4377b414f737b2890`](https://github.com/Ar9av/obsidian-wiki/commit/e3ead02a320f7959317084e4377b414f737b2890). The repository is a Python package and skill framework that implements Andrej Karpathy's [LLM Wiki pattern](./karpathy-llm-wiki.md) as an agent-maintained, Obsidian-native knowledge base.

## What it is

A **digital brain** framework: raw sources are read once, distilled into interconnected markdown pages, and kept current by AI agents. The user owns the vault as plain files; Obsidian is the viewer, and the agent is the maintainer.

The project ships as:

- A Python CLI/package (`obsidian-wiki`) for setup, cache checks, graph analysis, lint, query, and export.
- A set of **agent skills** under [`.skills/`](https://github.com/Ar9av/obsidian-wiki/tree/e3ead02a320f7959317084e4377b414f737b2890/.skills) — markdown instructions that any agent capable of reading files can execute.
- Bootstrap files for many agents (`CLAUDE.md`, `AGENTS.md`, `.cursor/rules/obsidian-wiki.mdc`, etc.).

## Core ideas

| Idea | Why it matters |
|------|----------------|
| **Compile, don't retrieve** | The wiki is a pre-synthesized artifact. Answer questions from compiled pages rather than re-reading raw sources every time. |
| **Agent skills as the interface** | Operations are encoded in `SKILL.md` files, not hard-coded tools. This makes the framework portable across Claude Code, Cursor, Codex, Gemini, Kiro, Pi, etc. |
| **Plain markdown as source of truth** | Everything lives in an Obsidian vault; no proprietary database. OKF bundles can be exported/imported losslessly. |
| **Delta-aware ingest** | A manifest and content-hash cache avoid reprocessing unchanged sources. |
| **Provenance-aware writing** | Claims are tagged as extracted, inferred, or ambiguous so users can tell signal from synthesis. |
| **Tiered retrieval** | Cheap index/summary reads are preferred over full-page reads so query cost stays flat as the vault grows. |
| **Graph-native operations** | Cross-linking, graph analysis, typed relationships, and export to JSON/GraphML/Neo4j/HTML are first-class. |

## Architecture

### Three layers

The framework is organized into three layers, described in [`.skills/llm-wiki/SKILL.md`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/llm-wiki/SKILL.md):

1. **Raw sources** — immutable input documents (PDFs, chat exports, images, code repos, web pages).
2. **The wiki** — LLM-maintained markdown pages with YAML frontmatter, `[[wikilinks]]`, and provenance markers.
3. **The schema** — conventions, page templates, categories, and workflows that tell the agent how to maintain the wiki.

### Vault layout

```
$OBSIDIAN_VAULT_PATH/
├── concepts/          # ideas, mental models
├── entities/          # people, orgs, tools, projects
├── skills/            # how-to knowledge
├── references/        # source summaries (papers use a special deep-dive template)
├── synthesis/         # cross-cutting analysis
├── journal/           # timestamped observations
├── projects/          # per-project knowledge mirrors global categories
│   └── <project>/<project>.md
├── _raw/              # staging inbox for quick captures
├── _staging/          # review queue when staged writes are enabled
├── _archives/         # timestamped snapshots for rebuild/restore
├── _meta/taxonomy.md  # controlled tag vocabulary
├── index.md           # content catalog
├── log.md             # append-only operation log
├── hot.md             # ~500-word recent-activity cache
├── _insights.md       # graph-shape analysis output
└── .manifest.json     # ingest ledger
```

### Special files

| File | Purpose |
|------|---------|
| `index.md` | Human-readable catalog by category; rebuilt after ingest. |
| `log.md` | Append-only record of every ingest, query, lint, rebuild, etc. |
| `hot.md` | Semantic snapshot of recent activity for cheap context. |
| `.manifest.json` | Tracks every ingested source, its hash, timestamps, and produced/updated pages. |
| `_meta/taxonomy.md` | Canonical tag vocabulary and aliases. |

## Agent skill system

Every operation is a skill. Skills are markdown files with YAML frontmatter (`name`, `description`) plus instructions the agent follows. Examples:

| Skill | Role |
|-------|------|
| [`wiki-setup`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-setup/SKILL.md) | Initialize vault structure, `.env`, special files, and Obsidian config. |
| [`wiki-ingest`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-ingest/SKILL.md) | Distill sources into wiki pages; supports append/full/raw modes. |
| [`wiki-query`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-query/SKILL.md) | Read-only answer synthesis with tiered retrieval and multi-hop graph traversal. |
| [`wiki-status`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-status/SKILL.md) | Delta report, token footprint, and graph insights. |
| [`wiki-lint`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-lint/SKILL.md) | Health audit and optional `--consolidate` self-healing pass. |
| [`cross-linker`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/cross-linker/SKILL.md) | Auto-discover and insert missing `[[wikilinks]]` and typed relationships. |
| [`tag-taxonomy`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/tag-taxonomy/SKILL.md) | Enforce controlled vocabulary. |
| [`wiki-export`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-export/SKILL.md) | Export graph to JSON, GraphML, Neo4j Cypher, HTML, and OKF markdown bundle. |
| [`wiki-dedup`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-dedup/SKILL.md) | Identity-resolution merges for duplicate concept pages. |
| [`wiki-research`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-research/SKILL.md) | Autonomous multi-round web research filed into the wiki. |
| [`memory-bridge`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/memory-bridge/SKILL.md) | Browse/diff wiki knowledge by which AI tool produced it. |

`setup.sh` symlinks the canonical `.skills/` directory into each supported agent's discovery path, so the same skill definitions work everywhere.

## Ingest and delta tracking

### Modes

- **Append mode (default)** — uses content hashes to process only new/modified sources.
- **Full mode** — reprocess everything.
- **Raw mode** — promotes draft files from `_raw/` into proper wiki pages.

### Manifest

[`.manifest.json`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/llm-wiki/SKILL.md) is the ingest ledger. It stores canonical absolute source paths, `content_hash`, `ingested_at`, and the wiki pages each source created or updated. This enables delta computation, re-ingest on source change, and audit.

### Source formats

The ingest skill handles markdown, text, PDFs, academic papers (with figure/equation extraction), JSON/JSONL/CSV chat exports, meeting transcripts, images via vision, code directories (via an AST extractor), and web URLs.

### Project scoping

Project-specific knowledge lands under `projects/<name>/<category>/`. The project overview page must be named `<project-name>.md` (not a generic folder note) so Obsidian's graph view labels nodes correctly.

## Query and retrieval

`wiki-query` is read-only and follows an explicit cost ladder:

1. **GraphRAG pre-pass** — `obsidian-wiki graph-query` ranks candidates from titles/tags/summaries.
2. **Index pass** — read `index.md` and grep frontmatter for title/tag/alias/summary matches.
3. **QMD semantic pass** — optional lex+vec search when `QMD_WIKI_COLLECTION` is configured.
4. **Section pass** — grep a relevant section with context.
5. **Full read** — open at most the top 3 candidates, applying `tier:` ordering.
6. **Multi-hop traversal** — bounded BFS over typed `relationships:` edges for path queries.

This tiered approach is the framework's answer to scaling: a 2000-page vault should not require a full scan for every question.

## Page metadata conventions

Every page is expected to carry frontmatter that makes the wiki self-describing.

### Provenance markers

Claims are tagged inline:

| Marker | Meaning |
|--------|---------|
| *(none)* | Extracted from a source. |
| `^[inferred]` | LLM-synthesized generalization or implication. |
| `^[ambiguous]` | Sources disagree or the claim is unclear. |

A `provenance:` block can summarize the rough fractions per page.

### Confidence and lifecycle

- `base_confidence` — computed from distinct source count and source-quality buckets (paper, official, docs, repo, blog, etc.).
- `lifecycle` — `draft | reviewed | verified | disputed | archived`. Only ingest skills write `draft`; human edits promote other states.
- `tier` — `core | supporting | peripheral`. Controls update priority and query ordering.
- `superseded_by` — wikilink to successor when a page is archived.

### Typed relationships

An optional `relationships:` frontmatter block adds semantic edges (`extends`, `implements`, `contradicts`, `derived_from`, `uses`, `replaces`, `related_to`). These are used by query, export, and cross-linking.

## Cross-linking and graph maintenance

The [`cross-linker`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/cross-linker/SKILL.md) skill scans page bodies for unlinked mentions of known page names/aliases, scores candidates, inserts `[[wikilinks]]`, and writes typed relationship entries. It respects the configured link format (`wikilink` or `markdown`) and ignores code blocks and frontmatter.

[`wiki-lint --consolidate`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-lint/SKILL.md) can then auto-fix broken links, rescue orphans, normalize tags, add contradiction callouts, demote stale peripheral pages, and produce a consolidation report.

## Quality, maintenance, and enrichment

| Capability | What it does |
|------------|--------------|
| `wiki-status` | Delta report, token footprint, stale-core/orphan/synthesis suggestions, and graph insights (`_insights.md`). |
| `wiki-lint` | Orphans, broken links, missing frontmatter, contradictions, provenance drift, tag fragmentation, visibility issues, typed-relationship validity. |
| `wiki-dedup` | Detect and merge duplicate concept pages, leaving redirect stubs. |
| `wiki-research` | Multi-round web research, synthesized into source/concept/entity/synthesis pages. |
| `wiki-capture` | Save the current conversation as a wiki note; `--quick` drafts findings to `_raw/`. |
| `daily-update` | Freshness, index, and hot-cache maintenance cycle. |
| `graph-colorize` | Rewrite Obsidian's `graph.json` to color nodes by tag, category, or visibility. |

## Export and interoperability

[`wiki-export`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/wiki-export/SKILL.md) produces:

- `graph.json` — NetworkX node-link graph.
- `graph.graphml` — Gephi/yEd/Cytoscape.
- `cypher.txt` — Neo4j `MERGE` statements.
- `graph.html` — self-contained interactive vis.js visualization.
- `okf/` — OKF v0.1 markdown bundle.

### OKF mapping

The framework defines an explicit mapping between its native frontmatter and OKF v0.1:

| OKF key | Source field | Notes |
|---------|--------------|-------|
| `type` (required) | `category` title-cased | `concepts` → `Concept`, `entities` → `Entity`, etc. |
| `title` | `title` | Verbatim. |
| `description` | `summary` | One-line preview. |
| `tags` | `tags` | Includes `visibility/*` system tags. |
| `timestamp` | `updated` | ISO 8601. |
| `resource` | first `sources:` URL | Optional. |
| *(extensions)* | `category`, `sources`, `created`, `relationships`, `lifecycle`, `tier`, `base_confidence`, etc. | Preserved verbatim for lossless round-trips. |

Body `[[wikilinks]]` are converted to file-relative markdown links during OKF export and restored on import.

## Configuration and multi-vault routing

Configuration resolves via the **Config Resolution Protocol**:

1. Inline `@name` token → `~/.obsidian-wiki/config.<name>` (per-invocation override).
2. Walk up from CWD for a `.env` containing `OBSIDIAN_VAULT_PATH`.
3. Fall back to `~/.obsidian-wiki/config`.

This lets users keep multiple named vaults and route one request to a non-default vault without changing their active config. The protocol is documented in [`llm-wiki/SKILL.md`](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.skills/llm-wiki/SKILL.md).

## Key references

- Repository: <https://github.com/Ar9av/obsidian-wiki>
- Analyzed commit: [`e3ead02a320f7959317084e4377b414f737b2890`](https://github.com/Ar9av/obsidian-wiki/commit/e3ead02a320f7959317084e4377b414f737b2890)
- [README at this commit](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/README.md)
- [`.env.example` at this commit](https://github.com/Ar9av/obsidian-wiki/blob/e3ead02a320f7959317084e4377b414f737b2890/.env.example)
- [OKF v0.1 specification](./okf-v-0-1.md)
- [Karpathy LLM Wiki pattern](./karpathy-llm-wiki.md)
