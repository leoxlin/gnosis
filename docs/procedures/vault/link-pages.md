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
