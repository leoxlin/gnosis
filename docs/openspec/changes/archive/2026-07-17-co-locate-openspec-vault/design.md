## Context

The repository is an implicit gnosis vault whose source root is `docs`. OpenSpec 1.6 currently owns a separate top-level `openspec` directory and hardcodes discovery to the nearest ancestor containing an `openspec/` directory. Native OpenSpec Markdown deliberately omits gnosis YAML frontmatter, so moving the directory under `docs` without compatibility work makes gnosis validation and retrieval fail.

The earlier migration intentionally kept the gnosis runtime independent from OpenSpec. This design preserves dependency independence while revising the stricter "no adapter" boundary: gnosis will understand the small, stable OpenSpec filesystem convention as a read-only Markdown projection but will not execute OpenSpec, parse its YAML configuration, or own its lifecycle.

## Goals / Non-Goals

**Goals:**

- Keep exactly one canonical OpenSpec tree inside the gnosis vault.
- Preserve ordinary OpenSpec 1.6 commands from the repository root.
- Make current specs and archived or active change artifacts valid, searchable gnosis pages without modifying their native content.
- Resolve the repository vault consistently from any descendant working directory.
- Preserve OpenSpec as the sole writer and lifecycle owner of its artifacts.

**Non-Goals:**

- Vendor, fork, invoke, or embed the OpenSpec npm package in the gnosis binary.
- Add gnosis commands that proxy OpenSpec workflows.
- Generalize frontmatter-free Markdown acceptance beyond recognized OpenSpec artifact paths.
- Make `gnosis apply page` an OpenSpec authoring API.
- Change OpenSpec's schema or artifact formats.

## Decisions

### Store one real tree at `docs/openspec` and retain a relative symlink

All current OpenSpec files move to `docs/openspec`. A repository-root symlink named `openspec` points to `docs/openspec`, satisfying OpenSpec 1.6's fixed `<project>/openspec` lookup while keeping a single source of truth.

OpenSpec commands are documented and checked from the repository root. Commands launched below `docs` may report `docs` as their planning root because OpenSpec chooses the nearest parent containing an `openspec` directory, but they address the same physical artifacts. Repository-development agents should invoke OpenSpec from the repository root so its edit scope covers the whole project.

Alternatives rejected:

- Keep two copied trees: creates immediate split-brain state.
- Patch or vendor OpenSpec: adds maintenance and dependency ownership unrelated to gnosis.
- Require `cd docs` for every command: breaks existing root-level workflows and produces a docs-only project scope.
- Add a custom wrapper or gnosis subcommand: duplicates OpenSpec's command surface and conflicts with direct OpenSpec ownership.

### Project native artifacts into gnosis metadata by path

When a Markdown page has no YAML frontmatter, gnosis will recognize only these vault-relative shapes:

- `openspec/specs/<capability>/spec.md`
- `openspec/changes/<change>/{proposal,design,tasks}.md`
- `openspec/changes/<change>/specs/<capability>/spec.md`
- the same change artifacts below `openspec/changes/archive/<archived-change>/`

gnosis synthesizes metadata in memory:

| Artifact | Type | Title basis |
| --- | --- | --- |
| Main or delta spec | `OpenSpecSpec` | capability and change identity |
| Proposal | `OpenSpecProposal` | change identity |
| Design | `OpenSpecDesign` | change identity |
| Tasks | `OpenSpecTasks` | change identity |

The original bytes remain the revision source and read output. Synthetic `type`, `title`, `description`, and `tags` exist only in the gnosis projection, so OpenSpec files remain native and OpenSpec validation stays authoritative.

Alternatives rejected:

- Add YAML frontmatter to every artifact: OpenSpec-generated main specs do not propagate delta frontmatter, and future agents could omit it.
- Exclude `docs/openspec` from the vault: defeats co-location and knowledge retrieval.
- Accept every frontmatter-free Markdown file: weakens the vault contract and hides malformed authored knowledge.

### Keep OpenSpec artifacts read-only through gnosis

`gnosis apply page` rejects destinations matching a recognized OpenSpec artifact path with an actionable error. OpenSpec remains the only authoring and lifecycle interface, while gnosis provides validation, reading, graphing, serving, and retrieval.

This is path-contract compatibility, not a runtime OpenSpec dependency. The binary does not shell out, read `config.yaml`, or import an OpenSpec library.

### Anchor implicit repository vaults at the Git worktree root

When no gnosis configuration is selected and the start path is inside a Git worktree, gnosis will use the nearest ancestor containing `.git` as its configuration root and that root's `docs` directory as the vault. Repository-local `gnosis.local.toml` or `gnosis.toml` files are discovered from the start path upward through the worktree root before the user-global fallback.

This makes commands from `docs/openspec`, `cmd/`, and other descendants resolve the same `gnosis://local/...` identities.

## Risks / Trade-offs

- [Git symlinks may be disabled on some Windows checkouts] → Fail the repository layout check clearly; native OpenSpec 1.6 has no configurable nested root to use instead.
- [OpenSpec adds a new standard artifact name] → Unknown files remain strict gnosis pages and fail without frontmatter, making the gap visible instead of silently misclassifying it.
- [A user stores unrelated files at a matching OpenSpec path] → The exception is intentionally narrow and the `openspec` namespace is reserved for OpenSpec ownership.
- [Nested OpenSpec invocation reports `docs` as project root] → Document and test repository-root invocation as canonical; both roots still address the same physical tree.
- [Synthetic metadata drifts from source content] → Derive identity solely from the stable path and keep descriptions generic; search still indexes the complete native body.

## Migration Plan

1. Add native OpenSpec artifact recognition, read-only protection, Git-root discovery, and regression tests while the current root remains usable.
2. Move the complete OpenSpec tree to `docs/openspec` and replace the old root directory with a relative symlink.
3. Update OpenSpec project context, main-spec purposes, README paths, and repository checks.
4. Validate OpenSpec and gnosis from the repository root and gnosis from a nested `docs/openspec` path.
5. Archive this change through the root symlink so its final artifacts and updated specs remain inside the vault.

Rollback restores the top-level directory, removes the symlink, and reverts the gnosis projection and Git-root discovery changes. No external data migration is required.

## Open Questions

None.
