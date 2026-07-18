## Why

gnosis currently duplicates repository intent and delivery state across bespoke Purpose, Decision, Directive, and development Procedure records. OpenSpec already provides a coherent proposal, specification, design, task, and archive lifecycle, so repository development should use it while gnosis focuses on portable vault knowledge and vault procedures.

## What Changes

- **BREAKING** Remove the bundled Purpose, Decision, and Directive concept definitions, their repository records, and all development procedures built around them.
- Make OpenSpec the repository source of truth for project context, requirements, technical choices, implementation tasks, and completed-change history.
- Preserve current product behavior as six capability specifications and consolidate active historical rationale into this change's design.
- Keep the Procedure concept, the five vault procedure records, and the `using-gnosis` vault gateway.
- Remove the development gateway, Directive-based coding-agent integration, and obsolete Superpowers rationale.
- Pin the OpenSpec toolchain and validate OpenSpec artifacts in normal project checks.

## Capabilities

### New Capabilities

- `vault-management`: Typed Markdown vaults, canonical identities, validation, composition, precedence, indexes, and logs.
- `knowledge-retrieval`: Lexical and semantic retrieval, derived indexing, provenance, and bounded results.
- `agent-cli`: Resource-oriented commands and compact, predictable agent-facing output.
- `knowledge-serving`: Read-only HTTP and MCP access to vault knowledge.
- `github-wiki-sync`: Safe Git-backed synchronization for GitHub Wiki vaults.
- `vault-procedures`: Validation, discovery, precedence, and exact invocation of vault procedures.

### Modified Capabilities

None. This change establishes the initial OpenSpec baseline.

## Impact

- Removes bundled document URIs and plugin entry points without compatibility aliases.
- Changes embedded documentation, repository docs, CLI examples, plugin metadata, and tests; ordinary Go APIs and command grammar remain unchanged.
- Adds Node 22.12.0 and `@fission-ai/openspec` 1.6.0 as development tools only. gnosis does not gain an OpenSpec runtime dependency or adapter.
