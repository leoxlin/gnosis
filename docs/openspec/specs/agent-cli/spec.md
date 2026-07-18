# agent-cli Specification

## Purpose
Define the predictable, resource-oriented command grammar and compact output contract that make gnosis reliable for agents and shell automation.
## Requirements
### Requirement: Commands use a resource-oriented grammar
gnosis SHALL expose ordinary operations as `gnosis <verb> <resource> [identity] [flags]`, keep one persistent `--vault` context flag, and reject removed command forms rather than retaining aliases.

#### Scenario: Use a supported command
- **WHEN** an agent runs a documented verb-resource command with valid positional identities
- **THEN** gnosis performs that operation without requiring interactive input

#### Scenario: Use a removed command
- **WHEN** an agent invokes a retired command or flag
- **THEN** gnosis returns a usage error rather than silently translating it

### Requirement: Ordinary output is compact TOON
gnosis SHALL render ordinary success, no-op, empty, help, and error responses as valid TOON with deterministic field ordering.

#### Scenario: Decode command output
- **WHEN** a normal command writes to stdout
- **THEN** the complete stdout value decodes as TOON and contains no progress diagnostics

### Requirement: Lists and details minimize follow-up calls
gnosis SHALL include total counts, definitive empty messages, compact default fields, requested `--fields` ordering, bounded detail previews, and a `--full` escape hatch.

#### Scenario: Read a long page
- **WHEN** a caller requests a long page without `--full`
- **THEN** gnosis returns a character-bounded preview, total size, truncation state, and exact command for retrieving the complete content

### Requirement: Errors are self-correcting
gnosis SHALL reject invalid arguments before side effects, use exit code 2 for usage errors and 1 for runtime failures, and place structured actionable errors on stdout while reserving stderr for diagnostics.

#### Scenario: Supply an unknown flag
- **WHEN** a caller supplies a flag unsupported by the selected subcommand
- **THEN** gnosis exits with code 2 and identifies the valid command surface

### Requirement: Protocol outputs bypass ordinary TOON rendering
gnosis SHALL preserve native shell completion, HTTP JSON, and MCP protocol formats rather than passing them through the ordinary CLI output boundary.

#### Scenario: Run MCP over stdio
- **WHEN** `gnosis serve mcp` is active
- **THEN** stdout contains protocol frames only
