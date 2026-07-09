# `gnosis`

`gnosis` is a knowledge-first interface for agentic memory. It gives authors
and agents one way to organize, validate, query, and eventually synthesize
across wiki, vector, graph, structured, episodic, and future memory backends.

The repository is bootstrapped as knowledge first: the `docs/` directory is an
OKF v0.1 bundle that records purpose, concepts, decisions, directives, and
deltas before the implementation code. The first wiki backend is
Obsidian-compatible markdown.

## Why `gnosis`

`gnosis` comes from the Greek word for knowledge or knowing. The name matters
because this project treats knowledge as active context: not only facts to
retrieve, but intent, semantics, decisions, and instructions that agents can use
to guide work.

For agents, the line between semantics and function is blurring. A skill,
schema, directive, or decision can be plain text, but once an agent can read it
and act from it, that text becomes functional. `gnosis` treats those texts as
first-class knowledge objects rather than incidental documentation.

Knowledge can bootstrap itself. This repository begins with its own purpose,
concepts, decisions, directives, and deltas before building the tools that will
operate over that knowledge.

Access to knowledge should be uniform. `gnosis` provides a singular point for
ingesting, validating, linking, querying, and synthesizing across memory
backends.

## Bootstrapping `gnosis`

`gnosis` is related to the sister repository `praxis`: `gnosis` grounds
knowledge, while `praxis` synthesizes from knowledge into action. The
repositories meet at the boundary between what is known and what should be done.

This repository is built around the concepts of `gnosis` itself: repository
purpose, decisions, directives, and deltas. The author creates and maintains the
purpose, and both author and agent are guided by it. When the purpose changes,
that change is recorded explicitly so future work inherits the new center.

The bootstrap logic is:

1. The author creates intent.
2. The author and agent collaborate on a directive.
3. The agent implements the directive.
4. Author and agent corrections are captured as decisions or deltas.

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

Include reusable project concepts for purpose, decisions, directives, and deltas:

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
