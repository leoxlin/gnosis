---
type: Decision
title: Use Git working trees for the GitHub Wiki backend
description: Synchronize GitHub Wiki vaults through Git while keeping gnosis's knowledge engine filesystem-based.
---

# Decision

Support GitHub Wiki as a read/write `gnosis` vault backend by resolving an explicitly configured `OWNER/REPOSITORY` to its `https://github.com/OWNER/REPOSITORY.wiki.git` remote and a cached local Git working tree.

Pull with fast-forward-only semantics before each command that loads the vault. After a successful mutating command, commit the resulting vault changes and push them to the wiki. Use the installed `git` executable and its existing identity and GitHub credentials. Stop on clone, pull, commit, conflict, authentication, or push failure without force-updating or discarding either side.

Keep parsing, retrieval, validation, linking, and document writes filesystem-based. The backend boundary supplies the synchronized local root and publishes successful changes; it does not introduce a GitHub API representation of vault pages.

# Why

GitHub Wiki already exposes each wiki as a Git repository. Reusing Git preserves the plain Markdown source of truth, works with public and private repositories through users' existing credentials, provides conflict detection and history, and avoids a new SDK or parallel persistence implementation.

A local working tree lets every existing vault operation retain its current filesystem contract. Fast-forward-only pulls and ordinary pushes fail safely when the local and remote histories diverge.

# Constraints

- GitHub Wiki is a primary vault backend, not a remote import in the composed `[[vaults]]` list.
- Configuration names the backend and GitHub `OWNER/REPOSITORY`; backend cache paths are implementation details outside the authored vault.
- Reads pull before loading. Successful writes commit all backend-vault changes from that command and push immediately.
- Read-only commands never create commits. A successful mutation that produces no file change does not commit or push.
- Synchronization never force-pushes, resets, merges, rebases, stores credentials, or resolves conflicts automatically.
- General remote imports and non-GitHub Git backends remain out of scope.
