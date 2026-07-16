---
type: Decision
title: Use `_` as the any-vault URI authority
description: Reserve `_` for vault-agnostic URI resolution while keeping emitted document identities concrete.
supersedes: gnosis://core/decisions/define-gnosis-uri-format.md
---

# Decision

Extend the canonical document-link grammar with `gnosis://_/<vault-relative-page-path>`. The `_` authority selects a page path from the effective composed view without naming its vault. Resolution follows configured vault precedence.

Concrete document identities remain `gnosis://<vault-name>/<vault-relative-page-path>`. gnosis never emits `_` as a document identity, and `_` is reserved from configured vault names.

Read-like selectors and authored links accept the any-vault authority. Wildcard writes target the first configured filesystem-backed vault in precedence order; shadowing a lower-precedence page retains the existing explicit update requirement.

# Why

Portable procedures and agent workflows need to address a known vault-relative path without discovering a workspace-specific vault name first. Reusing the effective view's existing precedence makes resolution deterministic and keeps returned identities provenance-specific.

# Constraints

- `_` resolves only vault-relative Markdown page paths and never becomes stored document identity.
- Read-like selectors and authored links return or render the selected page's concrete URI.
- Wildcard writes use the highest-precedence configured filesystem-backed vault and preserve Concept Type path validation and collision protection.
- Concrete URI selectors retain their exact-vault behavior.
- Query strings and fragments remain link-only suffixes and do not apply to selectors or write targets.

