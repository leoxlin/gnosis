## Context

gnosis stores portable typed Markdown knowledge and currently bundles three repository-intent types plus development procedures that plan and execute them. OpenSpec 1.6.0 is already initialized with the standard `spec-driven` schema and provides the proposal, requirements, design, tasks, apply, sync, and archive lifecycle that those records duplicate.

The migration changes the bundled knowledge and repository workflow, not the generic typed-page engine. Existing user-defined types remain ordinary Markdown; gnosis simply stops shipping or governing the retired lifecycles.

## Goals / Non-Goals

**Goals:**

- Make OpenSpec the only repository-development intent and delivery system.
- Preserve the current product contract in six testable capability specs.
- Preserve useful historical rationale in one archived migration design.
- Keep the Procedure system focused on repeatable vault work.
- Remove obsolete records, gateways, examples, fixtures, and integration coverage completely.

**Non-Goals:**

- Add OpenSpec commands, adapters, scaffolding, or Go dependencies to gnosis.
- Reject user-authored pages merely because their custom type name matches a retired built-in.
- Preserve compatibility aliases or the Directive lifecycle.
- Fork or customize the packaged OpenSpec schema.

## Decisions

### Use OpenSpec only for repository development

`openspec/config.yaml` carries stable project context and artifact rules. Main specs carry current observable requirements. Each change proposal carries motivation and scope, `design.md` carries technical choices and alternatives, and `tasks.md` carries implementation state. Completed changes are validated, synced, and archived.

This avoids a Go adapter and avoids wrapping OpenSpec with another Procedure lifecycle. The alternative—native gnosis commands or full Procedure wrappers—would duplicate state and drift from the installed workflow.

### Keep only vault procedures in the bundled model

The Procedure Concept Type and five vault workflows remain bundled. The development procedure directory and explicit development gateway are deleted. `using-gnosis` continues to select exact effective Procedure revisions for vault work.

The generic procedure engine remains tag- and ontology-agnostic; the shipped bundle and plugin are what become vault-only.

### Establish one capability-oriented baseline

Six capability specs group stable behavior by responsibility rather than by command or historical decision. This keeps requirements testable without creating one monolith or a file per CLI operation.

### Consolidate, do not reproduce, historical decisions

The prior Decision files have these dispositions:

| Previous record | Disposition |
| --- | --- |
| `bootstrap-knowledge-first` | Active OKF/Markdown source-of-truth constraints move to `vault-management`; its SDLC ontology is retired. |
| `define-gnosis-uri-format` | Superseded grammar is folded into the current concrete and any-vault URI requirements. |
| `use-any-vault-uri-authority` | Active behavior moves to `vault-management`. |
| `resolve-imported-vaults-by-local-order` | Active behavior moves to `vault-management`. |
| `keep-search-sources-and-retrieval-backends-replaceable` | Backend-independent contracts remain in `knowledge-retrieval`; its initial-backend state was superseded. |
| `use-pgvector-semantic-retrieval` | Current semantic-index behavior moves to `knowledge-retrieval`, aligned with current code where vector is the default. |
| `use-agent-native-resource-cli` | Active behavior moves to `agent-cli`. |
| `serve-read-only-knowledge-over-mcp-stdio` | Active read-only and channel constraints move to `knowledge-serving`, expanded to current HTTP behavior. |
| `use-git-working-trees-for-github-wiki-backend` | Active behavior moves to `github-wiki-sync`. |
| `consolidate-runtime-adapters-in-gnosis-plugin` | Superseded selection model is retained only as history here. |
| `select-procedures-in-controlling-agent` | Direct controller selection remains in `vault-procedures`; the development gateway is retired. |
| `stage-directive-planning` | Retired and replaced by the standard OpenSpec workflow. |

No repository Directive contains durable work to migrate; the only Directive fixture belongs to the integration being removed.

### Pin the development toolchain

`mise.toml` pins Node 22.12.0 and `@fission-ai/openspec` 1.6.0 and adds strict OpenSpec validation to project checks. OpenSpec remains absent from the compiled binary and `go.mod`.

### Remove the obsolete coding-agent integration

The Harbor fixture solely validates the development gateway and Directive status transition, and its verifier already expects removed CLI forms. It is deleted rather than rebuilt around third-party OpenSpec behavior.

## Risks / Trade-offs

- [Downstream links to bundled intent URIs break] → Treat this as an explicit breaking removal and provide no misleading aliases.
- [Historical rationale is over-compressed] → Preserve each prior record's disposition in this archive while relying on Git for verbatim history.
- [Documentation or fixtures retain the old model] → Perform a repository-wide retired-name and retired-path sweep outside this archive.
- [Plugin metadata is cached] → Update the Codex cachebuster and validate every remaining manifest and skill.
- [OpenSpec becomes unavailable on a fresh clone] → Pin its exact npm package and Node runtime in mise.

## Migration Plan

1. Add project context, artifact rules, and pinned OpenSpec tooling.
2. Remove the retired concept definitions, records, development procedures, gateway, integration, and supporting reference.
3. Update the Procedure definition, plugin metadata, README, CLI examples, and neutral test fixtures.
4. Validate Go behavior, plugin packaging, OpenSpec artifacts, and the gnosis vault.
5. Mark all tasks complete, sync the six delta specs into main specs, and archive this change.

Rollback is a Git revert of the migration commit before downstream vaults rely on the reduced bundle. There is no runtime data migration or staged rollout.

## Open Questions

None.
