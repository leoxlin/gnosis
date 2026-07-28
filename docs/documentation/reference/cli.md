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
| `gnosis plan knowledge-change <uri>` | Validate and diff one complete page without writing | `--filename <file>`/`-f`, exactly one of `--expected-absent` or `--expected-revision <revision>` |
| `gnosis apply knowledge-change <uri>` | Revalidate and apply one accepted plan | plan flags plus `--digest <digest>` |

`apply page` reads standard input when `--filename` is omitted. `--update`
allows an intentional local shadow of a lower-precedence page.

The knowledge-change commands form a stateless two-phase contract. Planning
returns the classified create, update, archive, no-op, or invalid operation;
deterministic Markdown diff; validation findings; affected relationships; and
a digest bound to the URI, complete candidate, expected state, and relevant
configuration. It changes no page, index, log, commit, or remote.

Apply requires the same URI, candidate, expected state, and digest. It refreshes
the target, rejects a changed digest or stale expected revision/absence, and
revalidates before writing through the existing atomic vault writer. Updates
preserve unrecognized frontmatter unless the candidate supplies that field.
Archival retains the page; physical deletion is unsupported. A remote push
failure leaves the one local cache commit intact and returns an error.

### Read and discover

| Command | Purpose | Flags |
|---|---|---|
| `gnosis get vaults` | List effective vaults and precedence | `--fields vault,kind,root,precedence` |
| `gnosis get concepts [type]` | List Concept Types or records of one exact type | `--fields uri,type,title,description,revision,trust` |
| `gnosis get pages [uri]` | List pages or read one exact page | `--fields uri,type,title,description,revision,trust`, `--full`, `--resolve-current` |
| `gnosis get procedures [uri]` | List invocable Procedures or read one contract | `--tags <tag,...>`, `--fields uri,type,title,description,revision,trust,invocation,tags`, `--full` |
| `gnosis get history <uri>` | Read bounded newest-first page history | `--cursor <cursor>`, `--limit <1-100>` |
| `gnosis get diff <uri>` | Diff two exact content revisions | `--from <revision>`, `--to <revision>`, `--limit <1-100000>` |
| `gnosis get changes` | Read committed effective-vault changes after a cursor | `--cursor <cursor>`, `--limit <1-100>` |

`--full` and `--resolve-current` require a URI. `--fields` applies only to list
output. Every tag passed to `--tags` must match. `--resolve-current` retains the
requested page and adds its bounded supersession result.

History entries keep Git commit identity separate from the SHA-256 content
revision. When authoritative local Markdown differs from the latest commit,
`get history` returns an explicit working entry; that entry is not a commit and
cannot become a change-feed cursor. A non-Git page returns its current revision
with `history_unavailable` instead of invented history.

`get diff` accepts only revisions available for the selected canonical page and
returns no partial diff when either endpoint is unknown. `get changes` with no
cursor establishes a committed baseline. Later calls classify additions,
updates, archival, supersession, effective removal, and origin replacement.
Change cursors are opaque and bound to the current repository identities and
effective-vault composition. A rewritten or pruned baseline expires; establish
a new baseline instead of attempting to repair Git history.

### Search and traverse

| Command | Purpose | Flags |
|---|---|---|
| `gnosis search knowledge <question>` | Rank relevant knowledge pages | `--backend vector\|lexical`, `--top <n>`, `--max-read <n>`, `--depth <n>`, `--fields uri,type,title,description,revision,score,trust` |
| `gnosis context knowledge <question>` | Resolve bounded cited evidence without generating an answer | `--strategy lexical\|vector\|hybrid`, exact constraint flags, `--max-evidence <n>`, `--max-chars <n>`, `--depth <n>` |
| `gnosis graph neighbors <uri>` | List adjacent typed links | `--direction out\|in\|both`, `--relation <type>` repeatable |
| `gnosis graph path <from-uri> <to-uri>` | Find a bounded path | `--direction out\|in\|both`, `--relation <type>` repeatable, `--depth <n>` |
| `gnosis audit knowledge` | Report deterministic read-only knowledge-health findings | `--class <class>`, `--type <type>`, `--tier <tier>`, `--page-limit <n>`, `--finding-limit <n>`, `--stale-after <duration>`, `--cursor <cursor>` |

Knowledge search defaults to the vector backend. Select `--backend lexical`
for live, service-free BM25F-style retrieval.

Knowledge audits default to every finding class except staleness. Selecting
`stale` requires a positive duration plus at least one type or tier filter.
Page bounds reject an over-broad request; finding bounds return an opaque
cursor over the severity-and-URI ordered result. Audits report evidence and
Procedure routing without changing pages, indexes, logs, configuration,
commits, or remotes.

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
| `gnosis serve mcp` | Serve fourteen default MCP tools over stdio | `--allow-knowledge-writes` |
| `gnosis serve http` | Serve the atlas, JSON API, and streamable MCP | `--address <host:port>`, `--allow-knowledge-writes` |
| `gnosis version` | Print the installed version | — |
| `gnosis completion <shell>` | Generate a shell completion script | shell-specific flags |

The HTTP address defaults to `127.0.0.1:8080`.

Both MCP transports advertise effective vault pages as `text/markdown`
resources. Direct resource discovery is ordered by canonical URI and returns at
most 100 descriptors per page with an opaque continuation cursor. Clients may
also discover the `gnosis://{vault}/{+path}` resource template. Resource reads
return the effective page's rendered Markdown, concrete origin, and revision;
`get_page` remains the model-controlled tool for exact reads.
The same catalog includes `trace_graph` for bounded neighbors and paths plus
`get_procedures` for all-match discovery and exact validated contracts.
`get_history`, `get_diff`, and `get_changes` expose the same history contracts
as the CLI.
`get_evidence_context` returns the same bounded evidence contract as
`gnosis context knowledge`.
`audit_knowledge` returns the same read-only health contract as
`gnosis audit knowledge`.
`propose_knowledge_change` is always available and has no write side effects.
`apply_knowledge_change` is absent unless the server operator starts the
transport with `--allow-knowledge-writes`. Enabling the tool authorizes its
server-side capability but does not replace the MCP host's responsibility to
obtain per-call user approval.

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
