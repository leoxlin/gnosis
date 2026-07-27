# Remember and recall

In this tutorial you will store one durable preference in a gnosis vault and
retrieve it through the same user-and-agent scope. It takes about five minutes.

Complete [Get started with gnosis](get-started.md) first and remain in the
`gnosis-demo` directory.

## Select a memory scope

Every memory operation requires both identities:

```bash
export GNOSIS_MEMORY_USER_ID="tutorial-user"
export GNOSIS_MEMORY_AGENT_ID="tutorial-agent"
```

Because no external memory variables are set, gnosis uses the writable vault
as the memory backend.

## Store a preference

```bash
gnosis add memory "I prefer concise answers."
```

The result reports one memory with the `vault` backend and an `ADD` event. A
typed page now exists under `memories/`; its frontmatter records the two
identities, timestamps, and a content hash.

Run the same command again:

```bash
gnosis add memory "I prefer concise answers."
```

The second result reports `NOOP`. Exact active duplicates do not create another
page.

## Retrieve the preference

```bash
gnosis search memory "concise answers"
```

The result contains only active memories for `tutorial-user` and
`tutorial-agent`. The default limit is five.

Change `GNOSIS_MEMORY_AGENT_ID` and repeat the search:

```bash
export GNOSIS_MEMORY_AGENT_ID="another-agent"
gnosis search memory "concise answers"
```

The earlier preference is absent because the identity scope changed. Restore
the original value if you want to inspect it again.

You have stored a durable, deduplicated record and observed that retrieval is
isolated by user and agent. Next, choose a task from the
[how-to guides](../how-to/index.md).
