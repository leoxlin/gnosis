---
type: Documentation
title: Basic Usage
description: Install the `gnosis` CLI locally and use the core vault setup, validation, index, and scaffold commands.
tags: [documentation, usage, cli, vault]
timestamp: 2026-07-09T23:21:52Z
---

# Basic Usage

# Audience

Authors, maintainers, and agents who need to install the `gnosis` CLI locally and work with an OKF vault.

# Subject

Basic `gnosis` CLI usage for local installation, vault setup, validation, index generation, and scaffold repair.

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

Include reusable project concepts for documentation, purpose, decisions, directives, and deltas:

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

# Maintenance

Keep this page aligned with the CLI command surface in `cmd/gnosis/main.go`, the mise tasks in `mise.toml`, and the repository README.
