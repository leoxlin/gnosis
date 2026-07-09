---
name: gnosis-bootstrap
description: Use when working on the `gnosis` codebase, repository conventions, OKF knowledge bundle, or SDLC decisions.
---

# `gnosis` Bootstrap

Use this skill when modifying the `gnosis` repository itself — its Go code, documentation bundle, build tooling, or repository conventions.

## Start here

1. Read `docs/repository/purpose.md` for the repository's durable purpose.
2. Read `docs/repository/decisions/` for major design decisions.
3. Read `docs/concepts/` for the OKF concept types this repo defines.

## Knowledge-first workflow

`gnosis` is bootstrapped knowledge-first. The `docs/` directory is an OKF v0.1 bundle that records purpose, ontology, decisions, directives, and deltas before the implementation code.

- Keep docs and code in sync. Document design intent in `docs/` before or alongside code changes.
- Every markdown concept must have parseable YAML frontmatter with a non-empty `type`.
- Reserved root files `index.md` and `log.md` should always exist in each vault root.
- Prefer absolute bundle-relative markdown links.

## Repo layout

| Path | Purpose |
|---|---|
| `cmd/gnosis/` | CLI surface for setup, validation, scaffold, ingest, query, and backend operations. |
| `internal/vault/` | Go libraries for OKF bundle handling, validation, and scaffolding. |
| `docs/` | OKF v0.1 knowledge bundle. |
| `mise.toml` | Build, test, and install tasks. |
| `gnosis.toml` | Vault configuration (link format, vault roots). |

## Tooling

- Build: `mise run build`
- Test: `mise run test`
- Validate docs: `go run ./cmd/gnosis validate`
- Install locally: `mise run localbin`

## Boundaries

`gnosis` is not:

- A model provider or inference platform.
- A single-backend or single-viewer tool.
- A general-purpose chat or note-taking application.
- A fixed, external ontology authority.

Respect these boundaries when proposing features or refactors.
