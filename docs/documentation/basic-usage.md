---
type: Documentation
title: Basic Usage
description: Install the `gnosis` CLI, configure vault behavior, query knowledge, run core commands, and check repository quality.
tags: [documentation, usage, cli, vault]
timestamp: 2026-07-10T11:28:39Z
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

Set up a vault:

```bash
gnosis setup -vault ./my-vault
```

Include reusable project concepts for purpose, decisions, and explicit automation directives:

```bash
gnosis setup -vault ./my-vault -concepts
```

Validate it:

```bash
gnosis validate -vault ./my-vault
```

Regenerate directory indexes:

```bash
gnosis index -vault ./my-vault
```

Query the live vault with a compact human-readable result:

```bash
gnosis query -vault ./my-vault "what do I know about rate limiting?"
```

Get the machine-readable retrieval and graph pre-pass used by agents:

```bash
gnosis graph-query -vault ./my-vault -pretty "how is rate limiting connected to retries?"
```

You can also run tasks directly through mise without installing the binary:

```bash
mise run build   # build to ./dist/gnosis
mise run test    # run all Go tests
```

## Configuration

When `gnosis.toml` is present, `gnosis` searches for it from the requested path
up through its parent directories. Supported vault settings are:

```toml
[vault]
link_format = "relative"
link_format_strict = false
vault_roots = ["docs"]
vault_index = true
vault_log = true
```

`link_format` must be `relative` or `absolute`. Vault roots must be non-empty,
unique relative paths contained by the directory holding `gnosis.toml`.
`vault_index` and `vault_log` default to `true`. When disabled, setup does not
create the corresponding files and validation does not require them; `gnosis index` is a successful no-op when `vault_index` is false. Changing
an option to false does not delete existing files. Unknown settings and unsafe
roots are errors.

## Querying

`gnosis query` and `gnosis graph-query` search every configured vault root directly, so results do not depend on `vault_index` and always reflect the current files. Both rank titles, aliases, tags, descriptions, types, paths, and body text while returning only compact metadata by default.

Common options, which must appear before the quoted question, are:

```text
-top <n>       candidate limit (default 3)
-max-read <n>  maximum recommended page reads (default 3; zero disables them)
-depth <n>     maximum link traversal depth (default 3)
```

`query` prints text unless `-json` or `-pretty` is supplied. `graph-query` always returns JSON; `-pretty` indents it. The JSON object contains `answer_type`, `candidates`, `path`, `should_read`, and `index_only`. Candidate descriptions are bounded and page bodies are never included in command output.

An exact title, alias, or concept-path lookup with a description may return `index_only: true`, meaning an agent can answer without opening the page. Relationship questions return the shortest resolved Markdown-link path within the requested depth. Both commands are read-only and do not update indexes or logs.

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
