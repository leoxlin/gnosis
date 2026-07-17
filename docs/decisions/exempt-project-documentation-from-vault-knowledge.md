---
type: Decision
title: Exempt project documentation from vault knowledge
description: Reserve the documentation directory inside the vault root for diataxis project docs that are not OKF pages.
---

# Decision

Reserve the directory name `documentation/` at the vault root. The vault treats everything under it as project documentation, not knowledge: page loading, search indexing, graph traversal, and validation skip it entirely, and `documentation/` files carry no OKF frontmatter.

The gnosis project's own diataxis documentation lives at `docs/documentation/` (tutorials, how-to guides, reference, explanation), inside the `docs/` vault root but outside the knowledge model.

# Why

The author requires comprehensive diataxis documentation under `docs/documentation`, and `docs/` is this repository's vault root. Without an exemption, every documentation page would need OKF frontmatter and would pollute search results, the graph, and validation with project meta-content that is not knowledge. Project documentation describes the tool; vault pages are the tool's data. Keeping them in one tree preserves single-checkout simplicity while the reserved-name exemption keeps the boundary exact.

# Constraints

- The exemption applies only to a `documentation/` directory at the vault root, not to nested directories with that name.
- Vault pages must not link into `documentation/`; documentation files reference vault pages and repository files with ordinary relative links.
- The exemption covers Markdown and any other files under `documentation/`; no frontmatter is required or interpreted there.
- The embedded bundle is unaffected: it already includes only `concepts/*.md` and `procedures/*/*.md`.
