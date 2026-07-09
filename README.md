# Gnosis

A toolkit for a unified, OKF-compatible agentic memory layer across wiki,
vector, graph, structured, and episodic backends.

It is bootstrapped knowledge-first: the `docs/` directory is an OKF v0.1 bundle
that records the repository purpose, ontology, decisions, directives, and deltas
before the implementation code. The first wiki backend is Obsidian-compatible
markdown.

## Setup

Install the `gnosis` binary to `~/.local/bin` using mise:

```bash
mise run localbin
```

Make sure `~/.local/bin` is on your `PATH`, then run `gnosis --help` to see
available commands.

## Quick start

Set up a vault:

```bash
gnosis setup -vault ./my-vault
```

Validate it:

```bash
gnosis validate -vault ./my-vault
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


