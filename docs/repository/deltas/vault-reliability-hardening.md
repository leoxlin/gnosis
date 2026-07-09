---
type: Repository Delta
title: Vault reliability hardening
description: Hardened configuration, parsing, generated writes, CLI behavior, and repository quality checks.
tags: [delta, reliability, vault, cli, validation]
timestamp: 2026-07-09T23:51:24Z
status: completed
---

# Fulfilled directives

- [Harden vault reliability](../directives/harden-vault-reliability.md)

# Change summary

- Configuration loading now rejects unknown settings, invalid link formats,
  duplicate roots, and roots outside the configuration directory.
- YAML frontmatter and Markdown links are parsed by maintained,
  standards-aware libraries; local links cannot escape their vault root.
- Scaffold and index output is written atomically and reports only files whose
  content changed.
- CLI execution has injectable streams, command coverage, successful help,
  strict positional arguments, and consistent diagnostics.
- `mise run check` and GitHub Actions enforce formatting, vet, tests, race
  tests, build, and documentation validation.

# Verification

- `mise run check`
- `go run ./cmd/gnosis validate`

# Deviations

None.

# Related decisions

- [Bootstrap `gnosis` knowledge first on OKF](../decisions/bootstrap-knowledge-first.md)
