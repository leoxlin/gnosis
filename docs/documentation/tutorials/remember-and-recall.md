# Remember and recall

In this tutorial you will store a durable preference as a scoped agent memory and retrieve it later. Time: about 5 minutes. Prerequisite: [get started](get-started.md).

## What memory is

A Memory page is one self-contained fact, preference, or observation under a scope (`user`, `agent`, `session`, `run`). Two procedures manage memories: `remember` writes them, `recall` reads them. You perform the procedures — they are contracts your agent follows, not hidden daemons.

## Remember a preference

Following the `remember` procedure (read it with `gnosis get procedures gnosis://_/procedures/vault/remember.md --full`):

1. Extract the durable statement: "The user prefers tabs over spaces in Go-adjacent YAML."
2. Compute its hash: `printf '%s' "The user prefers tabs over spaces in Go-adjacent YAML." | sha256sum`.
3. Check for duplicates and near neighbors: `gnosis search knowledge --backend lexical "tabs over spaces"`.
4. Nothing conflicts, so ADD `memories/user-prefers-tabs.md`:

        ---
        type: Memory
        title: Prefers tabs
        description: The user prefers tabs over spaces in Go-adjacent YAML.
        scope: user
        observed_at: 2026-07-17
        hash: <the sha256 from step 2>
        entities: [yaml, go]
        status: active
        ---

        # Memory

        The user prefers tabs over spaces in Go-adjacent YAML.

5. Apply it: `gnosis apply page gnosis://local/memories/user-prefers-tabs.md --filename memories/user-prefers-tabs.md`.

## Recall it

Following `recall`: run `gnosis search knowledge --backend lexical "formatting preferences"`, keep Memory records, and read the top candidate with `gnosis get pages gnosis://local/memories/user-prefers-tabs.md --full`. Answer with provenance: scope `user`, observed 2026-07-17.

## Update and archive

When the preference changes, `remember` reconciles: UPDATE revises the page in place (git keeps the history); DELETE sets `status: archived` with a reason line instead of deleting the file. Archived memories answer "what changed?" questions and are excluded from normal recall.

## What you learned

Memory is explicit: every write is a validated vault page, dedupe is by content hash, audit is git plus retained archives. Next: the [how-to guides](../how-to/index.md).
