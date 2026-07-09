---
type: Reference
title: Open Knowledge Format (OKF)
description: An open, human- and agent-friendly format for representing knowledge surrounding data and systems.
resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
tags: [okf, format, knowledge, metadata, ai]
timestamp: 2026-07-09T02:50:53Z
---

# Open Knowledge Format (OKF)

The Open Knowledge Format (OKF) is an open specification that formalizes the LLM-wiki pattern into a portable, interoperable format. It represents knowledge as a directory of markdown files with YAML frontmatter, using a small set of conventions so that wikis written by different producers can be consumed by different agents without translation.

## Principles

* **Minimally opinionated** — every concept requires only a `type` field; everything else is left to the producer.
* **Producer/consumer independence** — the format is the contract; tooling at each end is independently swappable.
* **Format, not platform** — OKF is not tied to a specific cloud, database, model provider, or agent framework.

## Structure

An OKF bundle is a directory tree of markdown files. Each file represents a concept. The file path (without the `.md` suffix) is the concept ID. Reserved filenames include:

| Filename | Purpose |
|---|---|
| `index.md` | Directory listing for progressive disclosure. |
| `log.md` | Chronological history of updates. |

## Concept documents

Every concept has:

1. A YAML frontmatter block with at least a `type` field.
2. A markdown body with free-form content and optional conventional sections such as `# Schema`, `# Examples`, and `# Citations`.

Concepts link to each other with standard markdown links. Absolute bundle-relative links beginning with `/` are recommended for stability.

## Versions

OKF is versioned as `<major>.<minor>`. The first published version is [OKF v0.1](./okf-v-0-1.md).

# Citations

[1] [OKF v0.1 specification](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
[2] [How the Open Knowledge Format can improve data sharing](https://cloud.google.com/blog/products/data-analytics/how-the-open-knowledge-format-can-improve-data-sharing)
