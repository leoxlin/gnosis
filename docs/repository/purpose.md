---
type: Repository Purpose
title: "`gnosis`"
description: "`gnosis` creates a singular interface for agentic knowledge and memory, where text, skills, repository intent, and backend stores can bootstrap one another."
tags: [gnosis, okf, llm-wiki, vector-rag, knowledge-graph, agentic-memory, ontology, go, repository-purpose]
timestamp: 2026-07-09T03:55:13Z
---

# `gnosis`

## Purpose

`gnosis` exists to make agentic knowledge uniform, durable, and usable through a singular interface. It brings together LLM wikis, vector RAG, knowledge graphs, structured stores, episodic or long-context memory, and whatever memory backends emerge next behind one memory model, one ingest path, and one query surface.

The name comes from Greek `gnosis`, meaning knowledge or knowing. That etymology matters because the project treats knowledge as active context: not only facts to retrieve, but structured intent, semantics, decisions, and instructions that agents can use to guide work.

For agents, the line between semantics and function is blurring. A skill, schema, directive, or decision can be plain text, but once an agent can interpret it and act from it, that text becomes functional. `gnosis` treats those texts as first-class knowledge objects rather than incidental documentation.

Knowledge can bootstrap itself. The repository begins by recording its own purpose, concepts, decisions, directives, and deltas in an [OKF-compatible](../references/okf.md) bundle before building the tools that will later operate over that bundle.

## Bootstrapping model

`gnosis` is related to the sister repository `praxis`: `gnosis` grounds knowledge, while `praxis` synthesizes from knowledge into action. The two repositories meet at the boundary between what is known and what should be done.

This repository is created around the concepts of `gnosis` itself: repository purpose, decisions, directives, and deltas. The author creates and maintains the purpose, and both author and agent are guided by it. When the purpose changes, that change is recorded explicitly so future work inherits the new center.

The bootstrap loop is:

1. The author creates intent.
2. The author and agent collaborate on a directive.
3. The agent implements the directive.
4. Author and agent corrections are captured as decisions or deltas.

## Sub-purposes

| Area | Purpose |
|---|---|
| Singular knowledge interface | Present one model for agentic knowledge that spans LLM wikis, vector RAG, knowledge graphs, structured stores, episodic memory, and future memory systems. |
| Backend support | Provide pluggable backends for wiki, vector, graph, structured, and other memory stores so storage and retrieval can be swapped without changing the knowledge model. |
| Ingest and validation | Import, transform, validate, and ground knowledge from raw sources into the memory model. |
| Query and synthesis | Retrieve and synthesize across heterogeneous memory stores for agents and authors. |
| Bootstrap workflow | Encode the purpose-to-directive-to-delta loop so author intent, implementation, and corrections remain durable. |
| Ontology and OKF foundations | Encode knowledge-management concepts like ontologies, schemas, provenance, and conceptual decomposition into agent-readable concepts. |


## Boundaries

`gnosis` is **not**:

- A model provider or inference platform.
- A single-backend or single-viewer tool.
- A general-purpose chat or note-taking application.
- A fixed, external ontology authority.

## Relationship to other goal types

- OKRs and project milestones live in the vault's `okr/` and `projects/` directories, not in this concept.
- User-facing features are tracked as work items, not as repository purposes.
- This purpose persists across sprints and release cycles; it changes only when the fundamental reason for `gnosis` changes.

## See also

- [Repository Purpose](../concepts/repository-purpose.md) — the ontological category this concept instantiates.
- [Open Knowledge Format (OKF)](../references/okf.md) — the format `gnosis` implements.
- `gnosis` README — human entry point for the repository.
