# vault-procedures Specification

## Purpose
Define how vault Procedure records are validated, discovered, selected, invoked, and exposed through the gnosis plugin.
## Requirements
### Requirement: Procedure records have one executable contract
gnosis SHALL require each Procedure to have a selection-focused description, non-empty tags, a model or explicit invocation mode, and either one complete Inputs/Process/Completion contract or two or more consecutive complete steps.

#### Scenario: Validate a malformed procedure
- **WHEN** a Procedure omits required metadata, duplicates sections, or has invalid step numbering
- **THEN** vault validation and exact invocation report the same contract failure

### Requirement: Discovery returns eligible exact procedures
gnosis SHALL discover only valid model-invocable Procedure records matching every requested tag and SHALL include their concrete URI, origin, revision, description, and tags.

#### Scenario: Filter the vault family
- **WHEN** an agent lists procedures with tags `gnosis,vault`
- **THEN** gnosis returns the eligible effective vault procedures and omits explicit-only records

### Requirement: Invocation binds one effective revision
gnosis SHALL invoke a Procedure only by canonical gnosis URI, apply effective-vault precedence, and return its exact current execution contract without mutating the vault.

#### Scenario: Local procedure overrides the bundle
- **WHEN** a higher-precedence vault supplies the same relative Procedure path
- **THEN** discovery and invocation use that effective page and report its concrete provenance

### Requirement: The core procedure bundle is vault-only
gnosis SHALL bundle the Procedure Concept Type and the `create-concept-type`, `ingest-knowledge`, `maintain-vault`, `query-vault`, and `refining-procedure` workflows, and SHALL NOT bundle repository planning, implementation, intent, or delivery workflows.

#### Scenario: Query the retired development family
- **WHEN** a clean vault lists bundled procedures with tags `gnosis,development`
- **THEN** gnosis returns a definitive empty result

### Requirement: The plugin exposes one vault gateway
The gnosis plugin SHALL expose only `using-gnosis`, which discovers the `gnosis,vault` family once, selects the smallest applicable procedure set in the controlling agent, reads exact contracts, and follows their completion gates.

#### Scenario: Install the plugin
- **WHEN** an agent host discovers the gnosis plugin skills
- **THEN** it finds the vault gateway and no gnosis development gateway
