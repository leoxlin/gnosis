---
type: Documentation
title: Basic Usage
description: Install the `gnosis` CLI, configure vault behavior, query knowledge, run core commands, and check repository quality.
tags: [documentation, usage, cli, vault]
timestamp: 2026-07-11T18:50:19Z
---

# Basic Usage

# Audience

Authors, maintainers, and agents who need to install the `gnosis` CLI locally and work with an OKF vault.

# Subject

Basic `gnosis` CLI usage for local installation, vault setup, validation, low-context queries, and optional index generation.

# Content

## Setup

Install the `gnosis` binary to `~/.local/bin` using mise:

```bash
mise run localbin
```

Make sure `~/.local/bin` is on your `PATH`, then run `gnosis --help` to see available commands.

## Quick start

Create a standalone OKF vault:

```bash
gnosis scaffold -vault ./my-vault
```

Include reusable project concepts for purpose, decisions, directives, and repository processes:

```bash
gnosis scaffold -vault ./my-vault -concepts
```

Create an import-only workspace that makes existing vaults available from one directory:

```bash
gnosis setup -vault ./knowledge-workspace -import ./my-vault
```

Validate it:

```bash
gnosis validate -vault ./my-vault
```

Regenerate directory indexes:

```bash
gnosis index -vault ./my-vault
```

Read one complete document by its exact concept type and title:

```bash
gnosis read -vault ./my-vault -type 'Repository Process' -title 'using-gnosis-forge'
```

Query the live vault with a compact human-readable result:

```bash
gnosis query search -vault ./my-vault "what do I know about rate limiting?"
```

Get the machine-readable retrieval and graph pre-pass used by agents:

```bash
gnosis query graph -vault ./my-vault -pretty "how is rate limiting connected to retries?"
```

Discover a process for an agent request, then invoke the selected exact revision:

```bash
gnosis process discover -vault ./my-vault -type 'Vault Process' -pretty "answer from this vault"
gnosis process invoke -vault ./my-vault -id 'gnosis://my-vault/processes/query-vault.md' -pretty
```

You can also run tasks directly through mise without installing the binary:

```bash
mise run build   # build to ./dist/gnosis
mise run test    # run all Go tests
```

These mise tasks stage the bundled documentation before compiling gnosis.

## Configuration

`gnosis` checks configuration in this order, without searching parent directories:
`./gnosis.local.toml`, `./gnosis.toml`, then `~/.config/gnosis.toml`. The first
file found is used. If none exists, gnosis uses its defaults with an empty
`[vault]` section, so only its default imports are available. A standalone
vault configuration declares its name and one local knowledge root:

```toml
[vault]
vault_name = "my-vault"
vault_root = "docs"
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
```

`link_format` must be `relative` or `absolute`. `vault_root` must be a non-empty
relative path contained by the directory holding `gnosis.toml`.
`vault_index` and `vault_log` default to `true`. When disabled, scaffold does not
create the corresponding files and validation does not require them; `gnosis index` is a successful no-op when `vault_index` is false. Changing
an option to false does not delete existing files. Unknown settings and unsafe
directories are errors.

An import-only workspace omits the local vault fields and declares each vault
in the top-level `vaults` array:

```toml
[[vaults]]
vault_name = "my-vault"
vault_root = "../my-vault"
```

Both `[vault]` and `[[vaults]]` are optional. gnosis always includes its bundled
forge and vault documentation. Local vault pages and pages from declared vaults
take precedence over bundled pages with the same vault-relative path.

Each declared vault root must point to a local directory. Local vault pages take
precedence, followed by declared vaults in their configured order. When pages
share the same path relative to a `vault_root`, the first page wins.

## Querying

`gnosis query search` and `gnosis query graph` search every configured local and
imported vault directory directly, so results do not depend on `vault_index` and
always reflect the current files. Both rank titles, aliases, tags, descriptions,
types, paths, and body text while returning only compact metadata by default.

Common options, which must appear before the quoted question, are:

```text
-top <n>       candidate limit (default 3)
-max-read <n>  maximum recommended page reads (default 3; zero disables them)
-depth <n>     maximum link traversal depth (default 3)
```

`query search` prints text unless `-json` or `-pretty` is supplied. `query graph` always returns JSON; `-pretty` indents it. The JSON object contains `answer_type`, `candidates`, `path`, `should_read`, and `index_only`. Each candidate carries its ID in `page`, stable URI, exact type, bounded description, effective origin, content revision, and score; page bodies are never included in query output.

An exact title, alias, or concept-path lookup with a description may return `index_only: true`, meaning an agent can answer without opening the page. Relationship questions return the shortest resolved Markdown-link path within the requested depth. Both commands are read-only and do not update indexes or logs.

`gnosis read` is also read-only. The compatibility form requires non-empty `-type` and `-title` flags and matches both exactly across configured vault roots. Agents should pass the exact effective ID or `gnosis://` URI returned by another command:

```bash
gnosis read -vault ./my-vault -id 'processes/query-vault.md'
gnosis read -vault ./my-vault 'gnosis://my-vault/processes/query-vault.md' -pretty
```

`-json` or `-pretty` with a URI or `-id` returns the page identity, origin, revision, and Markdown. `gnosis read gnosis://<vault-name>/<path>` is the direct form. Missing or ambiguous selectors are errors.

## Agent integration

gnosis exposes one shared agent contract through the CLI and an MCP server. The contract separates cheap discovery from exact invocation:

1. `discover_processes` / `gnosis process discover` ranks only exact `Vault Process` and `Repository Process` records. Results contain the selection conditions, invocation mode, possible effects, origin, revision, and stable URI without the full workflow.
2. `invoke_process` / `gnosis process invoke` loads the required sections and resolved outbound relationships for one exact URI. Invocation is read-only: the agent executes the returned contract under its existing authority and instructions.
3. `read_page`, `query_knowledge`, `trace_links`, and `find_path` provide exact reads, bounded retrieval, and deterministic directed graph traversal. Their CLI equivalents are `gnosis read`, `gnosis query graph`, `gnosis graph neighbors`, and `gnosis graph path`.

Run the MCP server over standard input and output from a vault or import workspace:

```bash
gnosis mcp serve -vault ./my-vault
```

The `gnosis` plugin declares this server in `.mcp.json`; compatible agent hosts can start it with the plugin. It publishes six read-only tools, every effective page as a `gnosis://` Markdown resource, and a live prompt for each executable process. Prompt retrieval reads the current effective page rather than copying its workflow into plugin packaging.

Trace typed outbound links or find a bounded path using an ID or URI:

```bash
gnosis graph neighbors -vault ./my-vault -id 'processes/query-vault.md' -direction out -pretty
gnosis graph path -vault ./my-vault -from 'processes/query-vault.md' -to 'concepts/retry.md' -direction out -depth 4 -pretty
```

Explicit frontmatter relationships preserve their type and direction. Markdown body links appear as `links_to`. A path result distinguishes `found`, `unknown_source`, `unknown_target`, `disconnected`, and `depth_exceeded` instead of collapsing every failure into an empty path.

Every machine-readable identity includes its effective origin (`local`, `import`, or `bundle`) and SHA-256 content revision. Local and imported precedence remains the configured vault's authority boundary; process effects never grant an agent permission it did not already have. Only exact process types are invocable, while all other records remain queryable knowledge.

List concept metadata as JSON when a host needs broad type discovery rather than process selection:

```bash
gnosis concepts -vault ./my-vault -json
gnosis concepts -vault ./my-vault -type 'Repository Decision' -pretty
```

## Output and failures

Changed paths, success summaries, and query results are written to standard output. Warnings, validation errors, invalid flags, malformed vault pages, and usage failures are written to standard error. Top-level and subcommand help is successful and writes to standard output. Commands reject unexpected positional arguments; query commands require one non-empty quoted question.

## Repository checks

Run the complete local quality gate before committing:

```bash
mise run checks
```

This checks formatting without rewriting files, then runs vet, uncached tests,
race tests, build, and validation of the repository knowledge bundle. GitHub
Actions runs the same script for pushes and pull requests.

# Maintenance

Keep this page aligned with the CLI command surface in `cmd/gnosis/main.go`,
vault configuration in `internal/vault/config.go`, the shared check script, the
mise tasks, and the repository README.
