# gnosis

gnosis is a local-first knowledge system for people and agents. It stores typed
knowledge as Markdown, assigns every page a stable `gnosis://` URI, and provides
tools to validate, search, connect, compose, and serve that knowledge.

Vaults remain readable in an editor, portable in git, and usable without a
database. Agents can discover `Procedure` pages and load them as explicit
execution contracts instead of relying on conversation history.

## Start here

gnosis requires Go 1.25 or later.

```bash
git clone https://github.com/leoxlin/gnosis.git
cd gnosis
go install ./cmd/gnosis
```

Create a vault and include the bundled concept definitions:

```bash
gnosis create vault --vault ./knowledge --name knowledge --concepts
cd knowledge
gnosis get concepts
gnosis validate vault
```

The vault is an ordinary directory of Markdown files. A page is identified by
its vault name and path:

```text
gnosis://knowledge/concepts/local-first.md
         └─ vault ─┘└──── page path ────┘
```

Use lexical search without configuring any external service:

```bash
gnosis search knowledge "How should I maintain this vault?" --backend lexical
```

For a guided first experience, continue with
[Get started with gnosis](docs/documentation/tutorials/get-started.md).

## What gnosis provides

- Typed Markdown pages with schema-bearing Concept Type definitions.
- Canonical URIs and validated relative or absolute links.
- Lexical search, optional vector search, and exact graph traversal.
- Deterministic views composed from local, imported, and bundled vaults.
- Twelve included Concept Type definitions and eight executable vault
  procedures.
- Scoped agent memory backed by the vault or by Mem0.
- A browser document atlas, a JSON API, and MCP tools.
- Read-only projection of the registered OpenSpec store.

Markdown is authoritative. Vector indexes are derived and disposable. OpenSpec
remains the authoring interface for its own artifacts.

## Documentation

The documentation follows the
[Diátaxis framework](https://diataxis.fr/):

- [Tutorials](docs/documentation/tutorials/index.md) provide guided learning
  experiences.
- [How-to guides](docs/documentation/how-to/index.md) solve specific tasks.
- [Reference](docs/documentation/reference/index.md) describes commands,
  configuration, types, and procedures.
- [Explanation](docs/documentation/explanation/index.md) discusses the design
  and its trade-offs.

The [documentation home](docs/documentation/index.md) routes readers by need.

## Agent plugin

The repository includes a gnosis plugin for Codex, Claude, and Kimi. Its
gateway skill finds applicable vault procedures and loads their complete
contracts. The `gnosis` executable must be available on `PATH`.

For Codex:

```bash
codex plugin marketplace add .
codex plugin add gnosis@gnosis
```

For Claude:

```bash
claude plugin marketplace add . --scope project
claude plugin install gnosis@gnosis --scope project
```

For Kimi:

```text
/plugins install ./plugins/gnosis
```

## Development

Install the pinned tools and run the complete project gate:

```bash
mise install
mise run checks
```

`mise run checks` checks formatting and vets the Go code, runs the tests normally and
with the race detector, builds the CLI, verifies the committed UI bundle, and
validates this repository as a gnosis vault.

Repository changes use [OpenSpec](https://github.com/Fission-AI/OpenSpec):

```bash
openspec new change <change-name>
openspec status --change <change-name>
openspec validate --all --strict --no-interactive
```

See [Plan with OpenSpec](docs/documentation/how-to/plan-with-openspec.md) for the
project workflow and [Code architecture](docs/documentation/explanation/architecture.md)
for the package layout.
