---
type: Directive
title: Add GitHub Wiki backend
description: Add transparent read/write GitHub Wiki storage for a primary gnosis vault.
status: done
---

# Goal

Allow a workspace to configure a GitHub Wiki as its primary `gnosis` vault and use existing commands against it with safe pull-before-read and commit/push-after-write synchronization.

# Scope

- Add a `github-wiki` primary-vault backend configured by GitHub `OWNER/REPOSITORY`.
- Resolve the backend to a user-cache Git working tree and reuse the existing filesystem vault engine.
- Add setup support that writes the backend configuration.
- Synchronize successful `gnosis write` and `gnosis index` mutations.
- Preserve local vaults and local composed imports unchanged.
- Do not add general remote imports, GitHub API access, automatic conflict resolution, credential storage, background refresh, or force operations.

# Dependencies

- [gnosis purpose](../purpose.md) @ `sha256:b432c4496daa35faed91cb294f41fa67522b1d9959daf29575baea09e59cf38f` — storage backends remain human-readable, portable, and author-controlled.
- [Use Git working trees for the GitHub Wiki backend](../decisions/use-git-working-trees-for-github-wiki-backend.md) @ `sha256:2c4dbc29c2b06966fc0fbea768e8799967219980722bbce74a90ddd2edeebb93` — supplies the synchronization and safety contract.

# Implementation plan

### Task 1: Resolve and synchronize the backend

**Load:** `internal/vault/config.go`, `internal/vault/search.go`, `internal/vault/write.go`, `internal/vault/index.go`, `internal/vault/config_test.go`, `docs/decisions/use-git-working-trees-for-github-wiki-backend.md`, and `../procedures/development/implementing-directive.md`.
**Files:** modify `internal/vault/config.go`, `internal/vault/search.go`, `internal/vault/write.go`, `internal/vault/index.go`; create `internal/vault/backend.go`, `internal/vault/backend_test.go`; test `internal/vault/config_test.go`.
**Interfaces:** extend `VaultConfig` with backend/repository fields; add internal backend preparation and publish operations returning a filesystem root; preserve existing exported vault operations.

- [x] Add a failing test that configures `backend = "github-wiki"` with `repository = "OWNER/REPOSITORY"` and proves vault loading clones or fast-forward pulls a Git working tree through the real Git executable; run `go test ./internal/vault -run 'TestGitHubWiki|TestLoadEffectiveVault'`; expect failure because the fields are rejected or no backend exists.
- [x] Add the minimum configuration validation, GitHub Wiki URL construction, user-cache working-tree preparation, and filesystem-root resolution needed to pass the focused test. Use `git clone`, `git -C <root> pull --ff-only`, and no new dependency.
- [x] Add a failing test that changes a backend vault through the real write interface and proves publication creates and pushes a commit; run `go test ./internal/vault -run 'TestGitHubWiki'`; expect failure because successful writes are not published.
- [x] Publish successful document and index mutations with `git add --all`, a fixed gnosis commit message, and `git push`; skip publication when `git status --porcelain` is empty. Keep failures explicit and never force, reset, merge, or rebase.
- [x] Run `gofmt` on changed Go files and `go test ./internal/vault`; expect all package tests to pass.
- [x] Commit: `feat: add GitHub Wiki vault backend`.

### Task 2: Configure a GitHub Wiki workspace

**Load:** `cmd/gnosis/setup.go`, `internal/vault/config.go`, the Task 1 interfaces, and `../procedures/development/implementing-directive.md`.
**Files:** modify `cmd/gnosis/setup.go`, `internal/vault/config.go`; create or modify focused tests beside the owning package.
**Interfaces:** `gnosis setup --github-wiki OWNER/REPOSITORY --name VAULT [--vault PATH] [--force]` writes a primary-vault `gnosis.toml`; existing `--import` behavior remains unchanged.

- [x] Add a failing focused test proving `setup --github-wiki OWNER/REPOSITORY --name VAULT` writes `backend = "github-wiki"`, the repository identity, and normal vault policy fields; run the focused test; expect failure because the flag and writer do not exist.
- [x] Add the two setup flags and the minimum config writer. Reject mixing `--github-wiki` with `--import`, require a canonical vault name, and leave existing import setup unchanged.
- [x] Run `gofmt` on changed Go files, the focused setup/config tests, then `go test ./...`; expect all tests to pass.
- [x] Commit: `feat: configure GitHub Wiki workspaces`.

# Acceptance criteria

- A GitHub Wiki workspace can be configured with `gnosis setup --github-wiki OWNER/REPOSITORY --name VAULT`; run its focused command test and inspect `gnosis.toml`; expect an exact `github-wiki` primary-backend configuration.
- A read against a configured backend clones an absent cache or runs a fast-forward-only pull on an existing cache before loading pages; run the real-Git backend test; expect remote changes to be visible.
- A successful `gnosis write` against the backend commits and pushes the changed wiki page; run the real-Git write test and inspect the remote clone; expect the authored content and one gnosis commit.
- A successful `gnosis index` publishes generated index changes, while a no-change mutation creates no commit; run the backend publication tests; expect remote indexes and unchanged history for the no-op case.
- Local vaults and composed local imports retain their behavior; run `go test ./...`; expect zero failures.
- Formatting, static analysis, race checks, build, and vault validation pass; run `mise checks`; expect exit status 0 with no warnings or failures.

# Completion evidence

- Task 1: commit `299d766`; the real-Git regression test failed before backend configuration existed and passed after clone, pull, document publication, index publication, and no-op publication behavior were implemented.
- Task 2: commit `73225f8`; command-level setup tests failed before the flags existed and passed with exact configuration and invalid-combination checks.
- Complete verification: `mise checks` exited 0 after formatting, `go vet`, the normal suite, the race suite, build, and validation of 35 Markdown files.
- Review range: `8d6a19d..73225f8`; no unresolved blocking finding.
- Delivery: verified commits retained on the current `main` branch; no remote push or pull request was requested.
