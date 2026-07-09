---
type: Repository Purpose
title: Gnosis
description: Gnosis seeks to create a single interface for all agentic memory, unifying LLM wikis, vector RAG, knowledge graphs, structured stores, and future memory backends.
tags: [gnosis, okf, llm-wiki, vector-rag, knowledge-graph, agentic-memory, ontology, go, repository-purpose]
timestamp: 2026-07-09T03:55:13Z
---

# Gnosis

## Purpose

Gnosis is building a unified interface for **agentic memory**. It brings together diverse memory systems — including LLM wikis, vector RAG, knowledge graphs, structured stores, episodic or long-context memory, and whatever memory backends emerge next — behind one memory model, one set of ingest tools, and one query surface.

It supports pluggable backends so users are not locked into a single store or viewer. It encodes concepts from knowledge management and information science — such as ontologies, provenance, and conceptual decomposition — directly into agent workflows, starting with an [OKF-compatible](../references/okf.md) markdown foundation.

## Sub-purposes

| Area | Purpose |
|---|---|
| Unified agentic-memory interface | Present one model for memory that spans LLM wikis, vector RAG, knowledge graphs, structured stores, episodic memory, and future memory systems. |
| Backend support | Provide pluggable backends for wiki, vector, graph, structured, and other memory stores so storage and retrieval can be swapped. |
| Ingest tools | Import, transform, validate, and ground knowledge from raw sources into the memory model. |
| Query tools | Retrieve and synthesize across heterogeneous memory stores for agents. |
| Ontology & OKF foundations | Encode knowledge-management concepts like ontologies, schemas, and provenance into agent-readable concepts. |
| `cmd/gnosis` | Deliver the command-line surface for setup, validation, ingest, query, and backend operations. |
| `internal/vault` | Offer reusable Go libraries for OKF bundle handling, validation, and scaffolding. |
| `skills/gnosis-vault` | Give agents portable instructions for working with Gnosis vaults. |

## Boundaries

Gnosis is **not**:

- A model provider or inference platform.
- A single-backend or single-viewer tool.
- A general-purpose chat or note-taking application.
- A fixed, external ontology authority.

## Relationship to other goal types

- OKRs and project milestones live in the vault's `okr/` and `projects/` directories, not in this concept.
- User-facing features are tracked as work items, not as repository purposes.
- This purpose persists across sprints and release cycles; it changes only when the fundamental reason for Gnosis changes.

## See also

- [Repository Purpose](../ontology/repository-purpose.md) — the ontological category this concept instantiates.
- [Open Knowledge Format (OKF)](../references/okf.md) — the format Gnosis implements.
- Gnosis README — human entry point for the repository.
