## MODIFIED Requirements

### Requirement: Markdown remains the authoritative vault format
gnosis SHALL store authored vault knowledge as human-readable Markdown with YAML frontmatter containing a non-empty `type`, while preserving applicable unknown metadata. gnosis SHALL also accept frontmatter-free Markdown at recognized standard OpenSpec artifact paths by projecting deterministic metadata in memory without rewriting the source.

#### Scenario: Read a typed page
- **WHEN** a configured vault contains a valid typed Markdown page
- **THEN** gnosis exposes its content, metadata, origin, and revision without converting the source to a proprietary format

#### Scenario: Read a native OpenSpec artifact
- **WHEN** a configured vault contains frontmatter-free Markdown at a recognized OpenSpec artifact path
- **THEN** gnosis exposes the original content with deterministic projected metadata, origin, and revision

#### Scenario: Reject an invalid page
- **WHEN** an applied page lacks required metadata or violates the effective Concept Type path
- **THEN** gnosis rejects the write without changing vault content

## ADDED Requirements

### Requirement: Implicit repository vaults anchor at the Git root
gnosis SHALL resolve an unconfigured invocation from any descendant of a Git worktree to the worktree root and SHALL use that root's `docs` directory as the implicit local vault.

#### Scenario: Invoke gnosis below the OpenSpec tree
- **WHEN** a caller runs gnosis from a descendant of `docs/openspec` without an explicit vault path
- **THEN** gnosis resolves the same repository vault and concrete document URIs as an invocation from the worktree root

#### Scenario: Prefer repository configuration
- **WHEN** a descendant invocation has a `gnosis.local.toml` or `gnosis.toml` on its path through the worktree root
- **THEN** gnosis uses the nearest repository configuration before considering the user-global configuration
