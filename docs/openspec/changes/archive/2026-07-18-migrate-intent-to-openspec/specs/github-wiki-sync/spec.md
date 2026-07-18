## ADDED Requirements

### Requirement: GitHub Wiki is a filesystem-backed primary vault
gnosis SHALL resolve an explicitly configured `OWNER/REPOSITORY` GitHub Wiki to an HTTPS `.wiki.git` remote and a cached local working tree while retaining the existing filesystem knowledge engine.

#### Scenario: Initialize a wiki vault
- **WHEN** a valid GitHub Wiki backend is first loaded
- **THEN** gnosis clones the wiki into its cache and reads the resulting Markdown as the configured vault

### Requirement: Reads synchronize safely
gnosis SHALL pull an existing wiki cache with fast-forward-only semantics before loading vault content.

#### Scenario: Remote history advanced
- **WHEN** the wiki remote has commits that fast-forward the cache
- **THEN** the next read pulls them before resolving pages

#### Scenario: Histories diverged
- **WHEN** the cache cannot be fast-forwarded
- **THEN** gnosis stops with the Git failure and does not merge, rebase, reset, or discard either history

### Requirement: Successful mutations publish immediately
gnosis SHALL commit all wiki-vault changes produced by one successful mutating command and push them with the installed Git identity and credentials.

#### Scenario: Apply a changed page
- **WHEN** a wiki-backed mutation succeeds and changes files
- **THEN** gnosis commits and pushes the resulting vault state

#### Scenario: Mutation is a no-op
- **WHEN** a successful operation changes no wiki file
- **THEN** gnosis creates no commit and performs no push

### Requirement: Synchronization never repairs destructively
gnosis MUST NOT force-push, reset, merge, rebase, store credentials, or resolve conflicts automatically.

#### Scenario: Push fails
- **WHEN** Git rejects the publish operation
- **THEN** gnosis reports the failure while preserving the local and remote histories for manual recovery
