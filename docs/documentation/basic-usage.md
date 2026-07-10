---
type: Documentation
title: Basic Usage
description: Install the `gnosis` CLI, configure vault behavior, run core commands, and check repository quality.
tags: [documentation, usage, cli, vault]
timestamp: 2026-07-10T11:28:39Z
---

# Basic Usage

# Audience

Authors, maintainers, and agents who need to install the `gnosis` CLI locally and work with an OKF vault.

# Subject

Basic `gnosis` CLI usage for local installation, vault setup, validation, optional index generation, and scaffold repair.

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

Repair the base vault shape without overwriting existing files:

```bash
gnosis scaffold -vault ./my-vault
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
`vault_index` and `vault_log` default to `true`. When disabled, setup and
scaffold do not create the corresponding files and validation does not require
them; `gnosis index` is a successful no-op when `vault_index` is false. Changing
an option to false does not delete existing files. Unknown settings and unsafe
roots are errors.

## Output and failures

Changed paths and success summaries are written to standard output. Warnings,
validation errors, invalid flags, and usage failures are written to standard
error. Top-level and subcommand help is successful and writes to standard
output. Commands reject unexpected positional arguments.

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
