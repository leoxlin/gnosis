# Get started with gnosis

In this tutorial you will install gnosis, create a knowledge vault, write a typed page, and answer a question from it. Time: about 10 minutes.

## Install

Build from the repository checkout:

    mise run build

This produces `dist/gnosis`. Put it on your `PATH` or call it directly.

## Create a vault

    mkdir my-vault && cd my-vault
    gnosis create vault --concepts

This writes `gnosis.toml`, `AGENTS.md`, `log.md`, and `concepts/` + `references/` directories, and generates `index.md` files. `--concepts` also copies the bundled concept type definitions (Purpose, Decision, Directive, Procedure, Concept, Entity, Resource, Event, Memory, Reflection, Policy) into your vault so you can refine them locally.

## Write your first page

Create `concepts/okf.md`:

    ---
    type: Concept
    title: OKF
    description: The Open Knowledge Format gnosis vaults follow.
    status: draft
    ---

    # OKF

    OKF is a Markdown-plus-frontmatter bundle format for portable knowledge.

Apply it to the vault:

    gnosis apply page gnosis://local/concepts/okf.md --filename concepts/okf.md

`apply page` validates the record, checks its links, and writes it atomically. Repeating the same apply is a no-op.

## Ask a question

    gnosis search knowledge "what is OKF" --backend lexical

You get a bounded candidate list with `should_read` pointers instead of a document dump. Read exactly one page with:

    gnosis get pages gnosis://local/concepts/okf.md --full

## Check vault health

    gnosis validate vault

Errors fail the command; warnings print to stderr. Run it after every batch of writes.

## What you learned

A vault is plain Markdown; `apply page` is the only write path; `search knowledge` narrows candidates; `get pages` reads exactly one record; `validate vault` guards integrity. Next: [remember and recall](remember-and-recall.md).
