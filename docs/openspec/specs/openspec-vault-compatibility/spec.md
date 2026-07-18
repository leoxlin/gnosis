# openspec-vault-compatibility Specification

## Purpose
Define how native OpenSpec artifacts are co-located inside a gnosis vault, discovered by both tools, projected as read-only knowledge, and kept under OpenSpec ownership.

## Requirements

### Requirement: OpenSpec has one co-located source of truth
The repository SHALL store OpenSpec configuration, current specs, active changes, and archived changes in `docs/openspec`, and the repository-root `openspec` path SHALL resolve to that same tree without copied state.

#### Scenario: Inspect the repository layout
- **WHEN** a caller resolves both `docs/openspec` and the repository-root `openspec` path
- **THEN** both paths identify the same physical OpenSpec tree and no second artifact copy exists

### Requirement: Native OpenSpec commands remain usable
The repository SHALL support unmodified OpenSpec 1.6 lifecycle commands from the repository root against the co-located tree.

#### Scenario: Resolve OpenSpec context
- **WHEN** a caller runs `openspec context` from the repository root
- **THEN** OpenSpec resolves the repository as its project root and uses the artifacts physically stored below `docs/openspec`

#### Scenario: Complete a change
- **WHEN** a valid change is created, applied, validated, and archived from the repository root
- **THEN** OpenSpec updates and archives only the co-located tree

### Requirement: gnosis projects native OpenSpec Markdown as typed knowledge
gnosis SHALL recognize native main specs, delta specs, proposals, designs, and task lists at standard OpenSpec paths without requiring YAML frontmatter, and SHALL expose deterministic synthetic type, title, description, tags, URI, origin, and revision metadata while preserving the original Markdown bytes.

#### Scenario: Read a current specification
- **WHEN** a caller reads `gnosis://local/openspec/specs/<capability>/spec.md`
- **THEN** gnosis returns the native OpenSpec Markdown as an `OpenSpecSpec` page with concrete provenance

#### Scenario: Search change history
- **WHEN** a caller performs lexical retrieval for text recorded in an archived OpenSpec proposal or design
- **THEN** the matching typed artifact is returned with its `gnosis://local/openspec/changes/archive/...` identity

#### Scenario: Encounter unrelated frontmatter-free Markdown
- **WHEN** a vault contains frontmatter-free Markdown outside a recognized OpenSpec artifact path
- **THEN** gnosis retains its ordinary missing-frontmatter validation failure

### Requirement: OpenSpec owns artifact mutation
gnosis SHALL treat recognized OpenSpec artifact destinations as read-only and MUST NOT execute, proxy, embed, or require OpenSpec at runtime.

#### Scenario: Apply an OpenSpec artifact through gnosis
- **WHEN** a caller targets a recognized OpenSpec proposal, design, tasks, or spec path with `gnosis apply page`
- **THEN** gnosis rejects the mutation with guidance to use OpenSpec and leaves the file unchanged

#### Scenario: Run gnosis without OpenSpec installed
- **WHEN** gnosis reads, validates, serves, or searches a vault containing native OpenSpec artifacts
- **THEN** those operations work from filesystem and Markdown conventions alone
