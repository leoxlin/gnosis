# Get started with gnosis

In this tutorial you will create a vault, add one typed page, find it again,
and validate the result. The exercise takes about ten minutes.

You need gnosis on your `PATH`. From a source checkout, build it with:

```bash
mise run build
export PATH="$PWD/dist:$PATH"
```

## Create a vault

From a directory where you keep projects, run:

```bash
mkdir gnosis-demo
cd gnosis-demo
gnosis create vault --name demo --concepts
```

The command creates `gnosis.toml`, agent instructions, indexes, a log, and the
bundled Concept Type definitions. Confirm that gnosis sees the vault:

```bash
gnosis get vaults
gnosis get concepts
```

The first result identifies the `demo` vault. The second lists types such as
`Concept`, `Memory`, and `Procedure`.

## Add a page

Apply a new Concept directly from standard input:

```bash
gnosis apply page gnosis://demo/concepts/local-first.md <<'EOF'
---
type: Concept
title: Local-first knowledge
description: Knowledge kept in files under the user's control.
status: active
---

# Local-first knowledge

Local-first knowledge remains useful without a remote service.
EOF
```

gnosis validates the frontmatter and URI before writing
`concepts/local-first.md`. Open that file in any Markdown editor; there is no
proprietary storage layer.

Read the page through its canonical identity:

```bash
gnosis get pages gnosis://demo/concepts/local-first.md --full
```

The result includes both metadata and the complete Markdown.

## Find the page

Search the live vault with the built-in lexical backend:

```bash
gnosis search knowledge "What remains useful without a remote service?" \
  --backend lexical
```

The candidate list points back to
`gnosis://demo/concepts/local-first.md`. Search narrows the candidates; `get
pages` retrieves an exact page.

## Validate the vault

```bash
gnosis validate vault
```

A successful command reports no validation errors. Run it after a batch of
edits so broken frontmatter, paths, and links are caught early.

You have now completed the basic gnosis loop: create a vault, apply typed
Markdown, search for a candidate, read an exact page, and validate the result.
Continue with [Remember and recall](remember-and-recall.md).
