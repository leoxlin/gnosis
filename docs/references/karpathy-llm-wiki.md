---
type: Reference
title: LLM Wiki (Karpathy)
description: Andrej Karpathy's pattern for building a persistent, LLM-maintained personal knowledge base as a markdown wiki.
resource: https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
tags: [llm, wiki, knowledge-base, okf, obsidian, rag]
timestamp: 2026-07-09T03:16:14Z
---

# LLM Wiki (Karpathy)

The **LLM Wiki** is a pattern, described by Andrej Karpathy, for using a large language model as the maintainer of a persistent, compounding knowledge base. Instead of treating source documents as a retrieval corpus that the model re-reads on every question, the LLM incrementally compiles sources into a structured, interlinked collection of markdown files — a wiki — and keeps that wiki current as new sources arrive.

## Core idea

Most document-based LLM workflows resemble RAG: the model retrieves chunks from raw sources at query time and answers from scratch each time. Karpathy's pattern replaces this with a persistent artifact:

- The LLM **reads** a source once.
- It **extracts** the key information.
- It **integrates** that information into an existing wiki, updating entity pages, topic summaries, and cross-references.
- The wiki becomes the answer surface: the model reads the already-synthesized pages rather than re-deriving knowledge from raw documents.

The result is a compounding knowledge base where contradictions, connections, and synthesis accumulate across sources.

## Architecture

The pattern has three layers:

| Layer | Description | Owner |
|---|---|---|
| **Raw sources** | Immutable source documents (articles, papers, transcripts, images). | Human curates; LLM reads only. |
| **The wiki** | Markdown files summarizing entities, concepts, sources, comparisons, and synthesis. | LLM creates and maintains. |
| **The schema** | A configuration document (e.g., `CLAUDE.md`, `AGENTS.md`) defining wiki structure, conventions, and workflows. | Human and LLM co-evolve. |

## Operations

Three recurring operations keep the wiki useful:

1. **Ingest** — add a source to the raw collection and have the LLM summarize it, update related pages, and append a log entry.
2. **Query** — ask questions against the wiki; the LLM reads relevant pages, synthesizes answers, and files valuable answers back into the wiki as new pages.
3. **Lint** — periodically health-check the wiki for contradictions, stale claims, orphan pages, missing concept pages, and gaps worth investigating.

## Indexing and logging

Two special files help navigation:

- **`index.md`** — content-oriented catalog of wiki pages, organized by category, with one-line summaries.
- **`log.md`** — append-only chronological record of ingests, queries, and lint passes.

At moderate scale (roughly hundreds of pages), this index-based navigation can replace embedding-based retrieval infrastructure.

## Optional tooling

The pattern is intentionally minimal, but tooling can help as the wiki grows:

- **Obsidian** as the reader/browser; its graph view shows wiki structure.
- **Obsidian Web Clipper** for converting web articles to markdown sources.
- **qmd** for local hybrid search over markdown files.
- **Marp** or **Dataview** for presentations and frontmatter queries.
- **Git** for version history and collaboration.

## Why it works

The maintenance burden that causes human-run wikis to stagnate — cross-referencing, updating summaries, flagging contradictions, touching many files — is cheap for an LLM. The human focuses on curating sources, asking questions, and directing analysis; the LLM handles the bookkeeping.

# Citations

[1] [Andrej Karpathy, "LLM Wiki" (gist)](https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f)
