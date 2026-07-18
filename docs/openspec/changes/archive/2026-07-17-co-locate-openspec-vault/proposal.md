## Why

OpenSpec is currently stored outside the repository's gnosis vault, so its requirements and change history are invisible to gnosis and cannot be queried with the rest of the knowledge base. Moving the canonical OpenSpec tree into `docs/openspec` also exposes two compatibility gaps: OpenSpec 1.6 expects a top-level `openspec` path, and gnosis rejects native OpenSpec Markdown because it intentionally has no YAML frontmatter.

## What Changes

- Move the canonical OpenSpec configuration, specs, active changes, and archive to `docs/openspec`.
- Keep a top-level `openspec` symlink to `docs/openspec` so unmodified OpenSpec 1.6 commands continue to work from the repository root without duplicating state.
- Recognize native OpenSpec specs, proposals, designs, task lists, and delta specs as read-only typed gnosis pages without rewriting their OpenSpec-owned Markdown.
- Anchor gnosis's implicit repository vault at the Git worktree root so commands run from source files or `docs/openspec` still resolve the repository's `docs` vault.
- Update repository guidance and checks to enforce the co-located layout and validate both systems.

## Capabilities

### New Capabilities

- `openspec-vault-compatibility`: Co-located OpenSpec storage, compatibility discovery, native artifact projection into gnosis, and ownership boundaries.

### Modified Capabilities

- `vault-management`: Allow recognized native OpenSpec artifacts inside the vault as a narrow exception to authored gnosis YAML frontmatter requirements and anchor implicit repository vaults at the Git root.

## Impact

- Affects OpenSpec paths, gnosis repository discovery and Markdown loading, vault validation, retrieval results, README guidance, mise checks, and regression tests.
- Adds no Go or runtime dependency on OpenSpec; gnosis recognizes its stable filesystem conventions and Markdown artifact names without executing or embedding the OpenSpec package.
- The top-level `openspec` path changes from a real directory to a symlink. The real, single source of truth becomes `docs/openspec`.
