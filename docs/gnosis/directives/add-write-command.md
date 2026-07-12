---
type: GnosisDirective
title: Add `gnosis write` command
description: Write a typed Markdown document from stdin or a file into the current vault.
status: done
---

# Goal

Add `gnosis write -type <type> -title <title> [-overwrite] [filename]`.

# Scope

Resolve the Concept Type and its required vault-relative `path` frontmatter from the composed knowledge view, but write only below the local vault configured in the current directory. Read content from stdin or one file, validate its `type` and `title` frontmatter against the flags, and derive the filename from the title. Preserve local-path precedence over imported and built-in documents. Do not add a vault-path flag or update generated documentation.

# Dependencies

- [Resolve imported vaults by local order](../decisions/resolve-imported-vaults-by-local-order.md)

# Implementation plan

1. Add failing vault-package tests for resolving a local-only write destination from a Concept Type `path`, validating input frontmatter, and rejecting unsafe or missing paths.
2. Implement the vault write operation, including composed-source lookup, current-local-vault enforcement, target filename derivation, collision classification, and atomic writes.
3. Add failing CLI tests for stdin and filename input, overwrite behavior for local versus imported or built-in documents, and command argument validation.
4. Register the Cobra `write` command, normalize its legacy single-dash flags, and connect it to the vault operation.
5. Run focused and full test suites, format Go files, inspect the diff, and set this directive to `done` only after all acceptance criteria have fresh evidence.

# Acceptance criteria

- The command accepts Markdown from stdin or one filename and writes it under the current directory's configured local vault only.
- A Concept Type's required, safe `path` frontmatter determines the destination directory; the title determines the Markdown filename.
- Input frontmatter `type` and `title` must exactly match the requested values.
- The command creates a local page that shadows an identically pathed imported or built-in page; non-local existing documents require `-overwrite`.
- Focused and full Go tests pass.
