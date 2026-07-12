---
type: GnosisDirective
title: Make read and write URI-only
description: Make canonical gnosis URIs the sole read and write document selectors.
status: done
---

# Goal

Make `gnosis read` accept only one positional `gnosis://` URI, and make `gnosis write` accept one positional target URI with content from stdin or `--filename`.

# Scope

- Remove `read` support for `--id`, `--type`, `--title`, and `--pretty`; retain `--json`, which always uses indented JSON.
- Replace `write` type/title and positional-file selection with `gnosis write <gnosis-uri>` and optional `--filename <path>`.
- Rename `--overwrite` to `--update`; preserve local writes and require it to shadow an imported or bundled document at the same URI.
- Validate target URI authority, canonical path, local-vault containment, and the document's typed destination.
- Update every documentation command reference outside `README.md` to the new interfaces and TypeName names.

# Implementation plan

### Task 1: Lock the URI-only command contracts with tests
**Load:** `cmd/gnosis/main.go`, `cmd/gnosis/main_test.go`, `internal/vault/write_test.go`.
**Files:** modify `cmd/gnosis/main_test.go`, `internal/vault/write_test.go`.
**Interfaces:** `read <gnosis-uri> [--json]`; `write <gnosis-uri> [--filename <path>] [--update]`.

- [x] Replace legacy selector/formatting tests with canonical URI, always-pretty JSON, stdin, filename, update, and invalid-target coverage.
- [x] Run `go test ./cmd/gnosis ./internal/vault`; expect failures until production code changes.

### Task 2: Implement URI-targeted reads and writes
**Load:** `cmd/gnosis/main.go`, `internal/vault/write.go`, URI/config helpers.
**Files:** modify `cmd/gnosis/main.go`, `internal/vault/write.go`.
**Interfaces:** `vault.WriteDocument(root, uri, content, update)` writes only to a canonical URI in the current local vault.

- [x] Simplify read argument parsing and make `--json` indented.
- [x] Parse write target URIs, derive their local filesystem target, verify content type against its ConceptType path, and protect imported/bundled collisions unless `--update` is passed.
- [x] Run `go test ./cmd/gnosis ./internal/vault`; expect success.

### Task 3: Align durable documentation
**Load:** every Markdown command reference outside `README.md`.
**Files:** modify affected records under `docs/` and `plugins/`.
**Interfaces:** all documented `read` and `write` invocations use only the new options.

- [x] Replace stale read selectors/pretty formatting and write type/title/overwrite forms, including historical directive records requested by the author.
- [x] Run `rg` for the removed options in command invocations and `gnosis validate --vault .`; expect no stale commands and a valid vault.

# Acceptance criteria

- `gnosis read gnosis://<vault>/<path>` is the only supported document selector and `--json` is indented.
- `gnosis write gnosis://<local-vault>/<path>` accepts stdin; `--filename` accepts exactly one file; `--update` replaces the legacy collision opt-in.
- Writes reject a noncanonical, foreign, unsafe, or ConceptType-path-inconsistent target URI and preserve imported/bundled pages unless `--update` is supplied.
- No Markdown command reference outside `README.md` retains a removed `read`/`write` option or stale TypeName command.
- Focused and full Go tests plus vault validation pass.

# Evidence

- `go test ./...` passed on the final workspace state.
- `gnosis validate --vault .` validated 41 Markdown files.
- A command-reference scan found no `gnosis read` use of `--id`, `--title`, `--type`, or `--pretty`, and no `gnosis write` use of `--type`, `--title`, or `--overwrite`.
