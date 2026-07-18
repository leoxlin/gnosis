# vault-management Specification

## Purpose
Define the Markdown storage, identity, composition, mutation, validation, and navigation rules for gnosis vaults.

## Requirements

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

### Requirement: Documents have stable gnosis URIs
gnosis SHALL emit concrete document identities as `gnosis://<vault-name>/<vault-relative-markdown-path>` and SHALL accept `gnosis://_/<path>` only as a precedence-aware selector.

#### Scenario: Resolve an any-vault selector
- **WHEN** a read uses the `_` authority for a path present in multiple effective vaults
- **THEN** gnosis returns the highest-precedence page with its concrete vault URI

#### Scenario: Reject an invalid selector
- **WHEN** a command receives a malformed URI, reserved vault name, query, or fragment where selectors do not allow one
- **THEN** gnosis returns a usage or validation error and does not guess a target

### Requirement: Vault composition is deterministic
gnosis SHALL resolve the local vault first, then declared imports recursively in configuration order, retain the first page at each vault-relative path, de-duplicate repeated vaults, and reject import cycles.

#### Scenario: Resolve an overlapping page
- **WHEN** local and imported vaults contain the same relative page path
- **THEN** the effective view selects the first page according to configured precedence while preserving each concrete origin

#### Scenario: Detect a cycle
- **WHEN** recursive imports lead back to an already active vault
- **THEN** gnosis rejects the configuration with the cycle identified

### Requirement: Implicit repository vaults anchor at the Git root
gnosis SHALL resolve an unconfigured invocation from any descendant of a Git worktree to the worktree root and SHALL use that root's `docs` directory as the implicit local vault.

#### Scenario: Invoke gnosis below the OpenSpec tree
- **WHEN** a caller runs gnosis from a descendant of `docs/openspec` without an explicit vault path
- **THEN** gnosis resolves the same repository vault and concrete document URIs as an invocation from the worktree root

#### Scenario: Prefer repository configuration
- **WHEN** a descendant invocation has a `gnosis.local.toml` or `gnosis.toml` on its path through the worktree root
- **THEN** gnosis uses the nearest repository configuration before considering the user-global configuration

### Requirement: Page mutation is explicit and collision-safe
gnosis SHALL write only to the URI-selected filesystem-backed vault, require explicit update authorization before shadowing a lower-precedence page, and report identical content as a no-op.

#### Scenario: Repeat an identical apply
- **WHEN** the requested page already contains byte-identical Markdown
- **THEN** gnosis succeeds with a structured no-op result and performs no write

### Requirement: Navigation policy is configurable
gnosis SHALL validate and generate Markdown indexes and logs only when their effective vault settings enable them.

#### Scenario: Validate a repository vault
- **WHEN** index and log generation are disabled in a repository vault
- **THEN** validation does not require generated navigation files
