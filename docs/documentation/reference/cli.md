# CLI reference

## Command grammar

Commands use `gnosis <verb> <resource>`. The persistent
`--vault <path-or-git-url>` flag selects the workspace and defaults to the
current directory. It accepts local paths plus explicit HTTPS and SSH Git
URLs:

```sh
gnosis --vault https://github.com/example/knowledge-vault.git get pages
gnosis --vault ssh://git@github.com/example/knowledge-vault.git get pages
```

`--help` is available at every command level.

Successful output, contextual help, and structured errors use
[TOON](https://github.com/toon-format/toon). Data is written to standard
output. Diagnostics are written to standard error. Usage failures and
operational failures exit non-zero.

## Commands

### Create and configure

| Command | Purpose | Flags |
|---|---|---|
| `gnosis create vault` | Scaffold an OKF-compatible vault | `--name <name>`, `--concepts`, `--force` |
| `gnosis apply workspace` | Write workspace composition | `--import <path-or-git-url>` repeatable, `--github-wiki <owner/repository>`, `--name <name>`, `--force` |
| `gnosis apply page <uri>` | Validate and write one typed page | `--filename <file>`/`-f`, `--update` |

`apply page` reads standard input when `--filename` is omitted. `--update`
allows an intentional local shadow of a lower-precedence page.

### Read and discover

| Command | Purpose | Flags |
|---|---|---|
| `gnosis get vaults` | List effective vaults and precedence | `--fields vault,kind,root,precedence` |
| `gnosis get concepts [type]` | List Concept Types or records of one exact type | `--fields uri,type,title,description,revision,trust` |
| `gnosis get pages [uri]` | List pages or read one exact page | `--fields uri,type,title,description,revision,trust`, `--full`, `--resolve-current` |
| `gnosis get procedures [uri]` | List invocable Procedures or read one contract | `--tags <tag,...>`, `--fields uri,type,title,description,revision,trust,invocation,tags`, `--full` |

`--full` and `--resolve-current` require a URI. `--fields` applies only to list
output. Every tag passed to `--tags` must match. `--resolve-current` retains the
requested page and adds its bounded supersession result.

### Search and traverse

| Command | Purpose | Flags |
|---|---|---|
| `gnosis search knowledge <question>` | Rank relevant knowledge pages | `--backend vector\|lexical`, `--top <n>`, `--max-read <n>`, `--depth <n>`, `--fields uri,type,title,description,revision,score,trust` |
| `gnosis graph neighbors <uri>` | List adjacent typed links | `--direction out\|in\|both`, `--relation <type>` repeatable |
| `gnosis graph path <from-uri> <to-uri>` | Find a bounded path | `--direction out\|in\|both`, `--relation <type>` repeatable, `--depth <n>` |

Knowledge search defaults to the vector backend. Select `--backend lexical`
for live, service-free BM25F-style retrieval.

### Memory

| Command | Purpose | Flags |
|---|---|---|
| `gnosis add memory <text>` | Store one memory in the configured identity scope | — |
| `gnosis search memory <query>` | Search active memories in that scope | `--limit <1-20>` |

Both commands require `GNOSIS_MEMORY_USER_ID` and
`GNOSIS_MEMORY_AGENT_ID`. The search limit defaults to 5.

### Index, validate, and serve

| Command | Purpose | Flags |
|---|---|---|
| `gnosis index vault` | Generate enabled Markdown indexes | — |
| `gnosis index knowledge` | Synchronize the vector index | — |
| `gnosis validate vault` | Validate structure, frontmatter, links, and contracts | — |
| `gnosis serve mcp` | Serve six MCP tools over stdio | — |
| `gnosis serve http` | Serve the atlas, JSON API, and streamable MCP | `--address <host:port>` |
| `gnosis version` | Print the installed version | — |
| `gnosis completion <shell>` | Generate a shell completion script | shell-specific flags |

The HTTP address defaults to `127.0.0.1:8080`.

## Remote Git targets

gnosis clones each remote into
`os.UserCacheDir()/gnosis/git-vaults/<sha256-of-normalized-url>`. The first
operation clones the repository's default branch. Later operations verify the
cached origin and run a fast-forward-only pull before resolving the vault.
Long-lived MCP and HTTP servers perform the same refresh when handling an
operation.

A remote selected directly with `--vault` is the writable target. A changed
page, index, scaffold, workspace configuration, or vault-memory operation
creates one `gnosis:` commit and pushes the current branch. A no-op creates no
commit and performs no push. If a push fails, the commit remains in the local
cache and the command returns an error.

Authentication comes from the user's existing Git and SSH configuration,
including credential helpers. gnosis does not accept embedded HTTPS
credentials. Remote targets do not select branches, revisions, subdirectories,
queries, or fragments. SCP-like `git@host:path` locators are not supported;
use an explicit `ssh://` URL. Cloning a repository that does not yet exist is
also outside this workflow.

## Canonical URIs

The canonical form is:

```text
gnosis://<vault-authority>/<path/to/page.md>
```

Selectors do not accept a query or fragment. The `_` authority selects the
first matching path in the effective view and is useful for portable
references. Reads render resolvable internal Markdown links as canonical
gnosis URIs.
