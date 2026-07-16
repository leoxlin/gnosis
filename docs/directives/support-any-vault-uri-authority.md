---
type: Directive
title: Support the any-vault URI authority
description: Resolve `gnosis://_/...` selectors, links, and writes through configured vault precedence without emitting wildcard identities.
status: done
---

# Goal

Support `gnosis://_/path/to/doc.md` as a portable any-vault URI that resolves through the effective vault order while preserving concrete document identities.

# Scope

- Accept `_` for read-like selectors and authored Markdown or relationship links.
- Resolve reads and links to the first effective page with the requested vault-relative path and return or render its concrete URI.
- Accept `_` for writes by targeting the first configured filesystem-backed vault; retain Concept Type path validation and require `--update` before shadowing a lower-precedence page.
- Reject `_` as a configured vault name and never emit it as a document identity.
- Preserve exact behavior for concrete vault authorities and reject wildcard query or fragment suffixes wherever selectors and write targets already reject suffixes.
- Do not modify `README.md` or add unrelated modernization.

# Purpose/Decision Changes

- `../decisions/define-gnosis-uri-format.md` @ `sha256:9ac5de52daff9868545610ad4b0952c30c4c9e08b149042bb28370c3a02d444f` → `../decisions/use-any-vault-uri-authority.md` @ `sha256:2fe25bf26178a0c0e6c71cab540dc396365a8597037e527db7693ddea8289c95` — reserves `_` for precedence-based resolution while keeping emitted identities concrete.

# Implementation plan

### Task 1: Lock any-vault behavior with regression tests
**Load:** `internal/vault/agent.go`, `internal/vault/links.go`, `internal/vault/write.go`, `internal/vault/config.go`, and their existing focused tests.
**Files:** modify `internal/vault/agent_test.go`, `internal/vault/render_test.go`, `internal/vault/write_test.go`, `internal/vault/config_test.go`.
**Interfaces:** `ReadPage(root, "gnosis://_/<path>")`; authored `gnosis://_/<path>` links; `WriteDocument(root, "gnosis://_/<path>", content, update)`; vault configuration validation.

- [x] Add one selector test with overlapping local/imported paths and an imported-only path; require precedence selection and concrete returned URIs.
- [x] Add one authored-link test; require wildcard body and relationship targets to resolve and rendered Markdown or graph edges to use the concrete selected URI.
- [x] Add one write test; require a new wildcard target to land in the first configured vault and a lower-precedence collision to retain the `--update` safeguard.
- [x] Extend invalid vault-name coverage with `_`.
- [x] Run `go test ./internal/vault -run 'Test(AnyVault|WriteDocumentAcceptsAnyVault|LoadEffectiveVaultRejectsNoncanonicalVaultName)' -count=1`; expect failures caused by missing `_` behavior.

### Task 2: Resolve `_` at the shared URI boundaries
**Load:** the failing tests plus `internal/vault/links.go`, `internal/vault/agent.go`, `internal/vault/write.go`, `internal/vault/config.go`, and every caller of `selectPage`, `canonicalGnosisParts`, and `documentResolver.resolve`.
**Files:** modify `internal/vault/links.go`, `internal/vault/agent.go`, `internal/vault/write.go`, `internal/vault/config.go`.
**Interfaces:** keep canonical parsing strict; reserve `_` as an any-vault authority; keep `Document.URI` concrete.

- [x] Define the reserved any-vault authority beside the URI helpers.
- [x] In `selectPage`, validate canonically and match `_` by `Document.Path`; retain exact concrete-URI matching otherwise.
- [x] In `documentResolver.resolve`, translate a wildcard link path through the effective logical-path index before graph, validation, and rendering consume it.
- [x] Reject `_` in `isCanonicalVaultName`.
- [x] In `WriteDocument`, parse the target authority, map `_` to `vault.sources[0]`, preserve concrete targets' current-local-vault restriction, build the candidate with the selected source's concrete identity, and retain external-collision validation.
- [x] Run `gofmt` on changed Go files, then rerun the focused test command; expect success.
- [x] Run `go test ./internal/vault -count=1`; expect success.

# Acceptance criteria

- `ReadPage` and every read-like operation using `selectPage` accept `gnosis://_/<path>`, select the effective page by configured precedence, and return its concrete URI — run the focused selector test; expect local-over-import and imported-only cases to pass.
- Authored body and relationship links using `_` resolve through the same effective path view, and rendered or graph output contains the concrete selected URI — run the focused authored-link test; expect no wildcard identity in output.
- `WriteDocument` accepts `_`, targets the first configured filesystem-backed vault, and still requires `update` before shadowing a lower-precedence page — run the focused write test; expect the new file in the first vault and guarded shadowing.
- `_` cannot be configured as a vault name and is never emitted as a page identity — run the focused configuration and selector/link tests; expect rejection and concrete identities.
- Existing concrete URI behavior remains green — run `go test ./internal/vault -count=1`; expect success.
- Formatting, vet, full tests including race detection, build, and vault validation pass — run `mise run checks`; expect exit status 0.

# Evidence

- Baseline `go test ./... -count=1` passed before production changes.
- The focused regression command failed before implementation at selectors, authored links, vault-name validation, and writes because `_` behavior was absent.
- The same focused command passed after implementation.
- `go test ./internal/vault -count=1` passed after implementation and review.
- `mise run checks` passed formatting, vet, uncached full tests, race tests, build, and validation of 41 Markdown files.
- `go run ./cmd/gnosis read 'gnosis://_/purpose.md' --json` returned concrete identity `gnosis://local/purpose.md`.
- `git diff --check` passed, and `README.md` remained untouched.
