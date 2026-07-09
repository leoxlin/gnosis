---
type: Repository Directive
title: Harden vault reliability
description: Make configuration, parsing, generated writes, CLI behavior, and repository checks predictable.
tags: [directive, reliability, vault, cli, validation]
timestamp: 2026-07-09T23:40:47Z
status: open
---

# Goal

Harden the existing OKF vault foundation so malformed input fails clearly,
generated files cannot be left partially written, CLI behavior is testable and
consistent, and the repository enforces the same quality checks locally and in
continuous integration.

# Scope

Include strict configuration validation, standards-aware YAML frontmatter and
Markdown link parsing, atomic generated-file writes, CLI tests and diagnostics,
and shared local and hosted quality checks.

Do not add ingest, query, storage backends, releases, or versioning behavior.

# Dependencies

- [Bootstrap `gnosis` knowledge first on OKF](../decisions/bootstrap-knowledge-first.md)
- [`gnosis` purpose](../purpose.md)

# Acceptance criteria

- Invalid or unsafe vault configuration is rejected with actionable errors.
- Valid YAML metadata and CommonMark links are parsed without ad hoc regex or
  line-oriented approximations.
- Local links cannot resolve outside their vault root.
- Scaffold and index writes are atomic and report only changed files.
- CLI help, arguments, output streams, and failure paths are covered by tests.
- One local command runs formatting checks, vet, tests, race tests, build, and
  documentation validation; pushes and pull requests run the same checks.
- Repository documentation records the delivered behavior and verification.

# Completion

Open.
