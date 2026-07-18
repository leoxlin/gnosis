## ADDED Requirements

### Requirement: Lexical retrieval reads live vault knowledge
gnosis SHALL provide a dependency-free lexical backend that reads the effective Markdown vault on every query and ranks source-independent documents without creating a persistent cache.

#### Scenario: Observe a source edit
- **WHEN** a vault page changes between lexical queries
- **THEN** the next query evaluates the changed content without an index synchronization step

### Requirement: Semantic retrieval uses disposable derived state
gnosis SHALL build bounded document chunks from the effective vault, embed them through the configured OpenAI-compatible endpoint, and store their concrete URI, revision, and metadata in pgvector.

#### Scenario: Synchronize semantic knowledge
- **WHEN** `gnosis index knowledge` runs with valid environment configuration
- **THEN** it atomically replaces the workspace scope and reports document, chunk, scope, and fingerprint metadata without modifying vault Markdown

### Requirement: Retrieval backends preserve one result contract
gnosis SHALL expose vector and lexical search through the same bounded candidate, path, and read-recommendation contract; vector SHALL remain the default and lexical SHALL remain explicitly selectable.

#### Scenario: Search without a backend flag
- **WHEN** a caller performs a knowledge search without selecting a backend
- **THEN** gnosis uses vector retrieval and returns compact provenance-bearing candidates

#### Scenario: Select lexical retrieval
- **WHEN** a caller supplies `--backend lexical`
- **THEN** gnosis searches live vault files without requiring embedding or database credentials

### Requirement: Search output is bounded and actionable
gnosis SHALL enforce positive candidate and graph-depth limits, preserve duplicate names through canonical URIs, and return explicit empty, exact, list, or gap classifications with `should_read` guidance.

#### Scenario: No pages match
- **WHEN** no indexed or live document matches the question
- **THEN** gnosis returns a definitive zero-match result and an actionable fallback hint
