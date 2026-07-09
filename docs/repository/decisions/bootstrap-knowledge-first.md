---
type: Repository Decision
title: Bootstrap `gnosis` knowledge first on OKF
description: Bootstrap `gnosis` as a knowledge-first project with OKF as the core format, layering extraction and LLM-wiki strategies on top and defining an SDLC ontology using that foundation.
tags: [okf, bootstrap, repository-decision, gnosis, sdlc, ontology]
timestamp: 2026-07-09T10:57:16Z
git_hash: 12b27558ae2b415f1a578cebd1991138d76dcae1
updated_at: 2026-07-09T10:57:16Z
---

# Decision

Bootstrap `gnosis` as a **knowledge-first** project: establish the [Open Knowledge Format (OKF) v0.1](../../references/okf-v-0-1.md) as the foundational knowledge format, layer extraction and LLM-wiki strategies on top, and define an SDLC ontology using that foundation.

# Context

`gnosis` is building a unified interface for agentic memory. Before building ingest, query, or backend code, the project needs a stable, portable, and agent-readable representation of its own purpose, decisions, directives, and deltas.

# Alternatives considered

* **Code-first bootstrap** — rejected because it would encode design choices in source before they were documented, making future agents reverse-engineer intent.
* **Proprietary wiki or note format** — rejected because it locks the corpus to a single tool.
* **Custom `gnosis` knowledge format** — rejected because OKF already provides a minimal, interoperable standard.

# Trade-offs

* OKF is intentionally minimal; richer behavior must be added by `gnosis` tooling rather than the format itself.
* A file-based markdown bundle is easy to version-control but may require scaling discipline as the corpus grows.

# Consequences

* The `docs/` directory is an OKF v0.1 bundle with frontmatter-typed markdown concepts, reserved `index.md`/`log.md` files, and a `references/`/`concepts/`/`repository/` layout.
* The [`gnosis` purpose](../purpose.md) and repository ontology ([Repository Purpose](../../concepts/repository-purpose.md), [Repository Decision](../../concepts/repository-decision.md), [Repository Directive](../../concepts/repository-directive.md), [Repository Delta](../../concepts/repository-delta.md)) are defined as OKF concepts.
* Extraction strategies ([LangExtract](../../references/langextract.md), [OntoGPT/SPIRES](../../references/ontogpt-spires.md)) and LLM-wiki patterns ([Karpathy LLM Wiki](../../references/karpathy-llm-wiki.md)) are documented as references and treated as layers above the OKF foundation.
* [Obsidian](../../references/obsidian-wiki.md) is adopted as the first wiki backend, but the bundle remains backend-independent.

# Related decisions

* [`gnosis` purpose](../purpose.md)
* [Repository Purpose](../../concepts/repository-purpose.md)
* [Repository Decision](../../concepts/repository-decision.md)
* [Repository Directive](../../concepts/repository-directive.md)
* [Repository Delta](../../concepts/repository-delta.md)
