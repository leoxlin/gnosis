---
type: Repository Decision
title: Define `gnosis` URI format
description: Establish one stable document-link grammar for gnosis pages.
---

# Decision

Define the canonical document-link grammar as:

```text
gnosis://<vault-name>/<vault-relative-page-id>
```

`gnosis` is the scheme, the authority is the vault name, and the path is the
effective page's vault-relative Markdown ID. `core` is the authority for
bundled knowledge. Query strings and fragments may accompany a rendered link
to preserve its destination within the page.

# Why

Document links need one portable, parseable shape across CLI output, rendered
Markdown, MCP resources, and composed vaults. Using the URI authority for the
vault and the path for the page ID follows standard URI structure, keeps the
link independent of local filesystem layout, and lets `core` identify bundled
knowledge without conflating it with a workspace's local vault.

# Constraints

- Newly emitted page identities and resolved rendered links use this grammar.
- Selectors recognize this grammar only; `gnosis://vault/<vault-name>/<path>` is not a gnosis document link.
- The path is an effective vault-relative page ID, not a local filesystem path.
- Plain effective page IDs remain accepted where the command already supports IDs.

# Rejected alternatives

- **`gnosis://vault/<vault-name>/<path>`** — rejected because it overloads the URI path with the vault identity and leaves the authority uninformative.
- **Local filesystem paths as document links** — rejected because they are not portable across vault roots or composed workspaces.
- **An alias for the former path-based form** — rejected because one link grammar keeps selectors and emitted identities unambiguous.
