# `gnosis`

`gnosis` is a local-first knowledge system for people and agents. It stores
typed knowledge as ordinary Markdown, gives every document a stable URI, and
provides a CLI for validating, searching, linking, composing, and acting on
that knowledge.

The result is shared context that stays readable in Obsidian, portable in git,
and precise enough for an agent to use without relying on conversation history.

## What it does

- Scaffolds Open Knowledge Format (OKF) compatible Markdown vaults.
- Reads and writes typed documents through canonical `gnosis://` URIs.
- Validates frontmatter, vault structure, and links.
- Searches documents and follows their relationships.
- Lists records by concept type.
- Composes a workspace from multiple local vaults.
- Discovers executable `Procedure` records for agents.
- Projects co-located OpenSpec artifacts as searchable, read-only knowledge.
- Ships core concepts and procedures inside the binary.

`gnosis` is intentionally local-first. Vaults are directories of Markdown
files, configuration is TOML, and generated indexes are also Markdown. Remote
vault imports are not yet supported.

## Install

`gnosis` requires Go 1.25 or later.

```bash
git clone https://github.com/leoxlin/gnosis.git
cd gnosis
go install ./cmd/gnosis
```

The binary is installed to `$(go env GOPATH)/bin`. This repository also uses
[mise](https://mise.jdx.dev/):

```bash
mise run build       # writes dist/gnosis
mise run localbin    # writes ~/.local/bin/gnosis
```

## Quick start

Create a vault with the reusable core concept definitions:

```bash
gnosis create vault --vault knowledge --name knowledge --concepts
cd knowledge
```

The generated vault is usable as plain Markdown immediately. You can also work
with it through the CLI:

```bash
gnosis get concepts
gnosis get pages gnosis://core/concepts/procedure.md --full
gnosis search knowledge "How should I maintain this vault?" --backend lexical
gnosis validate vault
```

A document URI combines a vault name with its path:

```text
gnosis://knowledge/notes/store-data-as-markdown.md
         ^ vault     ^ path within the vault
```

After defining a `Note` Concept Type at `concepts/note.md`, apply a typed
Markdown document from a file:

```bash
gnosis apply page \
  gnosis://knowledge/notes/store-data-as-markdown.md \
  --filename ./store-data-as-markdown.md
```

The document must contain YAML frontmatter with a recognized `type` and a
`title`. `gnosis apply page` checks the document before placing it at the path
named by the URI. Native OpenSpec artifacts are the narrow exception:
gnosis projects their metadata for reading and search, but OpenSpec remains
their only authoring interface.

## Core model

A vault is a set of typed Markdown documents connected by explicit links.
Vault authors define the types appropriate to their knowledge. `gnosis`
includes one reusable Concept Type, `Procedure`, for repeatable execution
contracts that agents can discover and follow.

The core bundle also provides five procedures for creating a Concept Type,
ingesting knowledge, maintaining a vault, querying a vault, and refining a
procedure. Built-in knowledge is available to every vault without being copied
into it. A local document with the same path can refine or replace the built-in
version while remaining ordinary Markdown under version control.

## CLI

Commands follow a `gnosis <verb> <resource>` structure. Successful results,
help, and errors use compact TOON output for agents and shell tooling.

| Command | Purpose |
| --- | --- |
| `gnosis create vault` | Create a new vault. |
| `gnosis apply workspace` | Configure a workspace that imports other vaults. |
| `gnosis apply page` | Validate and apply one typed Markdown document. |
| `gnosis get vaults` | List effective vaults and their precedence. |
| `gnosis get concepts` | List known types or records of an exact type. |
| `gnosis get pages` | List effective pages or read one exact page. |
| `gnosis get procedures` | List executable procedures or read one execution contract. |
| `gnosis search knowledge` | Find relevant pages for a question. |
| `gnosis graph neighbors` | Inspect typed links adjacent to a page. |
| `gnosis graph path` | Find a typed path between two pages. |
| `gnosis index vault` | Regenerate Markdown vault indexes. |
| `gnosis index knowledge` | Synchronize the semantic knowledge index. |
| `gnosis validate vault` | Validate vault structure and links. |
| `gnosis serve http` | Serve the API, document UI, and MCP over HTTP. |
| `gnosis serve mcp` | Serve read-only gnosis tools over MCP stdio. |

Run `gnosis <command> --help` for flags and examples supported by the installed
version.

## Compose vaults

A workspace can expose several local vaults through one configuration:

```bash
gnosis apply workspace --vault ./workspace \
  --import ../team-knowledge \
  --import ../project-knowledge
```

This writes `workspace/gnosis.toml`. Documents remain addressable by their
vault names, so callers can read an exact source without depending on filesystem
layout:

```bash
cd workspace
gnosis get pages gnosis://team-knowledge/references/data-policy.md --full
```

Imports are resolved in configuration order. Local records can override
imported or built-in records when applied with `gnosis apply page --update`.

## Agent integration

The repository includes a `gnosis` plugin for Codex, Claude, and Kimi. Its one
gateway skill discovers applicable vault `Procedure` records, loads their exact
contracts, and guides the agent through them. Repository-development planning
is outside the plugin's Procedure scope.

Install the local plugin with Codex:

```bash
codex plugin marketplace add .
codex plugin add gnosis@gnosis
```

Or with Claude:

```bash
claude plugin marketplace add . --scope project
claude plugin install gnosis@gnosis --scope project
```

From Kimi, install the plugin directory directly:

```text
/plugins install ./plugins/gnosis
```

The plugin expects the `gnosis` binary to be available on `PATH`.

## Configuration

A standalone vault uses a `gnosis.toml` like this:

```toml
[vault]
vault_name = "knowledge"
vault_root = "."
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
```

Workspace configurations can instead declare one or more `[[vaults]]` entries
that point to local vault roots.

## Development

Run the complete project checks with:

```bash
mise install
mise run checks
```

The repository uses [OpenSpec](https://github.com/Fission-AI/OpenSpec) for
development proposals, requirements, technical designs, implementation tasks,
and completed-change history. Its canonical tree lives inside the gnosis vault
at [`docs/openspec/`](docs/openspec/). The repository-root `openspec` path is a
relative compatibility symlink for OpenSpec 1.6, so run lifecycle commands from
the repository root:

```bash
openspec new change <change-name>
openspec status --change <change-name>
openspec validate --all --strict --no-interactive
```

Current capability requirements live under
[`docs/openspec/specs/`](docs/openspec/specs/), and completed changes live under
[`docs/openspec/changes/archive/`](docs/openspec/changes/archive/). gnosis can
query the same native Markdown without changing it:

```bash
gnosis get concepts OpenSpecSpec
gnosis get pages \
  gnosis://local/openspec/specs/vault-management/spec.md --full
gnosis search knowledge "How are implicit repository vaults resolved?" \
  --backend lexical
```

`mise run checks` validates OpenSpec, checks formatting, runs `go vet`, runs the
test suite with and without the race detector, builds the CLI, and validates
this repository's own vault. The [`docs/`](docs/) tree remains portable vault
knowledge, including current specifications, change history, references, the
Procedure Concept Type, and vault procedures.
