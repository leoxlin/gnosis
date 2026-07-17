---
type: Directive
title: Write diataxis documentation
description: Exempt documentation/ from the vault and write comprehensive diataxis docs under docs/documentation.
status: done
---

# Goal

Implement [Exempt project documentation from vault knowledge](../decisions/exempt-project-documentation-from-vault-knowledge.md) in the page walks, then write comprehensive user documentation under `docs/documentation/` following the diataxis structure: tutorials, how-to guides, reference, and explanation.

# Architecture

One shared helper marks the reserved directory; the three existing vault walks (page loading in `search.go`, index generation in `index.go`, validation in `validate.go`) and the index entry listing skip exactly `<vault-root>/documentation`. Documentation files are plain Markdown with relative links and no frontmatter.

# Tech stack

Go 1.25 (existing dependencies only); Markdown docs.

# Global constraints

- Follow [Exempt project documentation from vault knowledge](../decisions/exempt-project-documentation-from-vault-knowledge.md): root-only exemption; vault pages never link into `documentation/`.
- Diataxis: each file serves exactly one quadrant (learning-oriented tutorials, task-oriented how-to, information-oriented reference, understanding-oriented explanation).
- Docs describe the state after D1–D4 land: 11 concept types, `get directives`, `remember`/`recall`, tiered `query-vault`, `link-pages`, consolidation `maintain-vault`.
- Lower-case `gnosis` everywhere; never modify `README.md`.
- Relative links between documentation files; every linked file must exist.

# Scope

- `internal/vault/index.go`: add the reserved-directory helper and apply it in `indexDirectories` and `indexEntries`.
- `internal/vault/search.go`, `internal/vault/validate.go`: apply the helper in their walks.
- Tests: additions to `internal/vault/index_test.go`, `internal/vault/search_test.go`, `internal/vault/validate_test.go`.
- Create 22 documentation files under `docs/documentation/` with the exact contents in Tasks 2–5.

# Dependencies

- [Add the knowledge concept types](add-knowledge-concept-types.md) @ sha256:c7a98986dce1284500e4351ad0934b53004e4d4cf5784b4e9eb5a41dd77f0b4f — required contract: the 11 concept types and their fields are effective; prerequisite must be `done`.
- [Adopt OpenSpec strategies in intent tools](adopt-openspec-intent-strategies.md) @ sha256:6a8b1f53578f14e7c5de40c34dc0d44ee707ce96742155a5d2ebd75dcbd2281e — required contract: `gnosis get directives` and the directive contract exist as documented; prerequisite must be `done`.
- [Add scoped agent memory](add-scoped-agent-memory.md) @ sha256:59f60bbbbc3e21bd2d619f6b33d3ad12441a365a25c1626d25f9713554fb85d7 — required contract: `remember` and `recall` exist as documented; prerequisite must be `done`.
- [Improve wiki maintenance to obsidian-wiki parity](improve-wiki-maintenance.md) @ sha256:8fcf5b4ac511f7ffd6dbc648e0181d4329fb0b262ba03eefa5f136a6934dbc85 — required contract: tiered `query-vault`, `link-pages`, and consolidation `maintain-vault` exist as documented; prerequisite must be `done`.

# Implementation plan

### Task 1: Exempt documentation/ from the vault

**Load:** `internal/vault/index.go:75-95,150-197`, `internal/vault/search.go:260-280`, `internal/vault/validate.go:59-84`.
**Files:** modify `internal/vault/index.go`, `internal/vault/search.go`, `internal/vault/validate.go`, `internal/vault/index_test.go`, `internal/vault/search_test.go`, `internal/vault/validate_test.go`.
**Interfaces:** produces `exemptVaultDir(root, path string) bool`; all walks skip the root documentation directory.

- [x] Red: append the three tests below to their files; run `go test ./internal/vault/ -run 'Documentation'`; expect failures (documentation files are still walked).
- [x] Green: in `internal/vault/index.go`, add next to `ignoredVaultDir`:

```go
// documentationDirName is the reserved vault-root directory whose subtree is
// project documentation, not vault knowledge.
const documentationDirName = "documentation"

// exemptVaultDir reports whether path is the vault root's reserved
// documentation directory.
func exemptVaultDir(root, path string) bool {
	return path == filepath.Join(root, documentationDirName)
}
```

Change the `indexDirectories` skip to `if path != root && (ignoredVaultDir(entry.Name()) || exemptVaultDir(root, path)) {`, and in `indexEntries` change the directory branch guard to `if ignoredVaultDir(name) || exemptVaultDir(root, filepath.Join(dir, name)) {`. In `internal/vault/search.go` and `internal/vault/validate.go`, change each walk skip to `if path != source.path && (ignoredVaultDir(entry.Name()) || exemptVaultDir(source.path, path)) {`.

Test for `internal/vault/index_test.go`:

```go
func TestGenerateIndexesSkipsRootDocumentation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "concepts/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	write(t, root, "documentation/guide.md", "# Guide\n\nSee [missing](missing.md).\n")
	write(t, root, "notes/documentation/thing.md", "---\ntype: Note\ntitle: Thing\n---\n")
	written, err := GenerateIndexes(root, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "documentation", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("root documentation was indexed: %v", written)
	}
	rootIndex, err := os.ReadFile(filepath.Join(root, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rootIndex), "documentation/index.md") {
		t.Fatalf("root index links documentation: %q", rootIndex)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "documentation", "index.md")); err != nil {
		t.Fatalf("nested documentation dir was skipped: %v", err)
	}
}
```

Test for `internal/vault/search_test.go`:

```go
func TestSearchSourceExcludesRootDocumentation(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "documentation/guide.md", "# Guide\n\nNo frontmatter, no vault links.\n")
	write(t, root, "notes/documentation/thing.md", "---\ntype: Note\ntitle: Thing\n---\n")
	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		if strings.Contains(document.URI, "documentation/guide.md") {
			t.Fatalf("documentation page loaded: %+v", document)
		}
	}
	found := false
	for _, document := range documents {
		if strings.HasSuffix(document.URI, "notes/documentation/thing.md") {
			found = true
		}
	}
	if !found {
		t.Fatal("nested documentation page was excluded")
	}
}
```

Test for `internal/vault/validate_test.go`:

```go
func TestValidateSkipsRootDocumentationDir(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "documentation/guide.md", "# Guide\n\nBroken [link](missing.md) and no frontmatter.\n")
	write(t, root, "notes/documentation/thing.md", "---\ntitle: No type\n---\n")
	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range append(result.Errors, result.Warnings...) {
		if strings.Contains(problem, "documentation/guide.md") {
			t.Fatalf("documentation file validated: %v", problem)
		}
	}
	joined := strings.Join(result.Errors, "; ")
	if !strings.Contains(joined, "notes/documentation/thing.md") {
		t.Fatalf("nested documentation file skipped: %v", result.Errors)
	}
}
```

- [x] Run `go test ./internal/vault/`; expect ok. Run `gofmt -l internal/vault`; expect no output.
- [x] Create `docs/documentation/.gitkeep`-free proof: run `mkdir -p docs/documentation/tutorials docs/documentation/how-to docs/documentation/reference docs/documentation/explanation && printf '# Documentation\n\nExempt from the vault.\n' > docs/documentation/index.md`, then `mise run build && ./dist/gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- [x] Commit: `feat: exempt documentation directory from the vault`.

### Task 2: Landing page and tutorials

**Load:** `README.md` (install facts only; never modify it), `mise.toml`, the shipped CLI `--help` output.
**Files:** create `docs/documentation/index.md`, `docs/documentation/tutorials/index.md`, `docs/documentation/tutorials/get-started.md`, `docs/documentation/tutorials/remember-and-recall.md`.
**Interfaces:** produces the learning-oriented quadrant.

- [x] Write the four files with exactly these contents:

`docs/documentation/index.md`:

```markdown
# gnosis documentation

gnosis is a CLI that manages OKF-compatible knowledge vaults: plain Markdown pages with YAML frontmatter that humans and agents both own. It grounds, connects, and serves durable knowledge — concepts, events, memories, policies, and the intents that govern work — so understanding compounds instead of being rediscovered.

## Where to start

- **Learning?** Follow the [tutorials](tutorials/index.md): [get started](tutorials/get-started.md), then [remember and recall](tutorials/remember-and-recall.md).
- **Doing something specific?** Use the [how-to guides](how-to/index.md): ingest, query, intents, maintenance, semantic search, MCP, vault composition.
- **Looking something up?** See the [reference](reference/index.md): [CLI](reference/cli.md), [configuration](reference/configuration.md), [concept types](reference/concept-types.md), [procedures](reference/procedures.md).
- **Understanding the design?** Read the [explanation](explanation/index.md): [knowledge model](explanation/knowledge-model.md), [memory architecture](explanation/memory-architecture.md), [intent lifecycle](explanation/intent-lifecycle.md), [code architecture](explanation/architecture.md).

This documentation follows [diataxis](https://diataxis.fr/). It is project documentation, not vault knowledge: the `documentation/` directory is exempt from page loading, search, graph, and validation.
```

`docs/documentation/tutorials/index.md`:

```markdown
# Tutorials

Learning-oriented lessons that take you from zero to a working vault.

1. [Get started with gnosis](get-started.md) — install, create a vault, write and query knowledge.
2. [Remember and recall](remember-and-recall.md) — give your agent a durable, scoped memory.
```

`docs/documentation/tutorials/get-started.md`:

```markdown
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
```

`docs/documentation/tutorials/remember-and-recall.md`:

```markdown
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
```

- [x] Run `find docs/documentation -name '*.md' | sort`; expect the 4 files. Verify every relative link target exists.
- [x] Commit: `docs: add landing page and tutorials`.

### Task 3: How-to guides

**Load:** the effective procedures (`gnosis get procedures --tags gnosis,vault` and `--tags gnosis,development`), `docs/decisions/use-pgvector-semantic-retrieval.md`, `docs/decisions/use-git-working-trees-for-github-wiki-backend.md`, `docs/decisions/serve-read-only-knowledge-over-mcp-stdio.md`.
**Files:** create `docs/documentation/how-to/index.md` plus seven guides.
**Interfaces:** produces the task-oriented quadrant.

- [x] Write the eight files with exactly these contents:

`docs/documentation/how-to/index.md`:

```markdown
# How-to guides

Task-oriented recipes. Each assumes a working gnosis installation and a vault.

- [Ingest knowledge](ingest-knowledge.md) — turn sources into concept pages.
- [Query the vault](query-the-vault.md) — answer questions with tiered retrieval.
- [Work with intents](work-with-intents.md) — purposes, decisions, and directives.
- [Maintain a vault](maintain-a-vault.md) — lint, cross-link, consolidate.
- [Configure semantic search](configure-semantic-search.md) — pgvector retrieval.
- [Serve over MCP](serve-over-mcp.md) — read-only knowledge for agents.
- [Compose vaults](compose-vaults.md) — imports, bundles, and the GitHub wiki backend.
```

`docs/documentation/how-to/ingest-knowledge.md`:

```markdown
# Ingest knowledge

Turn supplied evidence — documents, transcripts, research — into durable concept pages.

## Steps

1. Load the procedure: `gnosis get procedures gnosis://_/procedures/vault/ingest-knowledge.md --full`, and follow it.
2. List exact types with `gnosis get concepts` and read the Concept Type definitions that apply (`gnosis get pages gnosis://_/concepts/<type>.md --full`).
3. Check identity before creating: `gnosis search knowledge --backend lexical "<the concept>"`. Update the matching page instead of duplicating it.
4. Write the record from its type's schema. Tag claims inline: unmarked for extracted facts, `^[inferred]` for your generalizations, `^[ambiguous]` for unresolved source disagreement.
5. Persist with `gnosis apply page '<record URI>' --filename <draft-file>`. When `vault_log` is enabled, add one newest-first entry to the nearest `log.md`.
6. When `vault_index` is enabled, run `gnosis index vault`. Always finish with `gnosis validate vault`.

## Rules of thumb

- One request about one concept changes exactly one page.
- Keep claims traceable: cite sources in the page body or `source` frontmatter.
- When no existing type fits, do not shoehorn — run the `create-concept-type` procedure.
```

`docs/documentation/how-to/query-the-vault.md`:

```markdown
# Query the vault

Answer a question from recorded knowledge without scanning the vault.

## Steps

Follow `query-vault` (`gnosis get procedures gnosis://_/procedures/vault/query-vault.md --full`), which applies this cost ladder:

1. **Catalog** — read the root `index.md` when `vault_index` is enabled.
2. **Lexical** — `gnosis search knowledge --backend lexical "<question>"`. An `index_only` answer needs no page reads; a `path` result answers relationship questions from link structure.
3. **Vector** — only when semantic retrieval is configured: `gnosis search knowledge --backend vector "<question>"`, merged by URI.
4. **Read** — open at most the top three `should_read` pages with `gnosis get pages '<URI>' --full`.
5. **Multi-hop** — `gnosis graph neighbors '<URI>'` or `gnosis graph path '<FROM>' '<TO>'` for relationship questions.

## Tips

- Questions about preferences or agent memories belong to `recall`, not the general ladder.
- No candidates means a knowledge gap — report it, or ingest the answer once you establish it.
- Always cite the page paths that support your answer; label synthesis `^[inferred]`.
```

`docs/documentation/how-to/work-with-intents.md`:

```markdown
# Work with intents

Intents are the vault's governance records: one **Purpose**, durable **Decisions**, and executable **Directives**. All CRUD goes through the `managing-intents` procedure, which defers to each Concept Type's lifecycle.

## Read current intent

    gnosis get pages gnosis://local/purpose.md --full
    gnosis get concepts Decision
    gnosis get directives

`get directives` derives `tasks_done`/`tasks_total` from checkbox steps — progress is computed, never restated.

## Record a decision

1. Query first: `gnosis search knowledge --backend lexical "<the choice>"` — decisions reject duplicate identity.
2. Draft the record (`# Decision`, `# Why`, `# Constraints`; `supersedes` link when replacing).
3. Apply with `gnosis apply page gnosis://local/decisions/<name>.md --filename <file>`.
4. A changed choice is a new Decision whose `supersedes` links the old one — never rewrite history.

## Plan and run a directive

1. Invoke `planning-directives` (`gnosis get procedures gnosis://_/procedures/development/planning-directives.md --full`): refine requirements, draft, review, finalize. It owns the `draft` → `open` transition.
2. A complete directive has Goal, Scope, checkbox tasks under `### Task N:`, evidence-bearing acceptance criteria (`#### Scenario:` blocks for behavior), and `## Added`/`## Modified`/`## Removed` deltas when it changes Purpose or Decision records.
3. Invoke `implementing-directive` for exactly one open directive at a time; it owns `open` → `blocked|done` and flips task checkboxes as work lands.
4. Periodically, when the author says `maintain-intents`, completed directives are archived: their declared deltas merge into the living records, durable choices compact into Decisions, and the directive files are removed.

## Validate

`gnosis validate vault` enforces the directive contract: valid `status`, required sections, checkbox tasks, scenario grammar, and delta headers.
```

`docs/documentation/how-to/maintain-a-vault.md`:

```markdown
# Maintain a vault

Keep the wiki linked, deduplicated, and fresh.

## Audit and repair

Run `maintain-vault` (`gnosis get procedures gnosis://_/procedures/vault/maintain-vault.md --full`):

1. Baseline: `gnosis validate vault`.
2. Audit orphans, near-duplicates, stale pages, contradictions, tag fragmentation, and broken typed relationships.
3. Apply high-confidence repairs in place through `gnosis apply page`; merge duplicates into the richer page and mark the loser `status: archived` with `superseded_by`.
4. Regenerate indexes (`gnosis index vault`) and log repairs when those options are enabled.
5. Re-validate and report every finding with its disposition.

## Cross-link pages

Run `link-pages` to convert high-confidence unlinked mentions of known titles and aliases into real links in the vault's configured format, adding typed `relationships` only where the text states them explicitly. It links at most the first mention per target and five new links per page — restraint keeps the graph readable.

## Cadence

Cross-link after large ingests; run the full consolidation pass on a schedule or when query results feel noisy.
```

`docs/documentation/how-to/configure-semantic-search.md`:

```markdown

# Configure semantic search

Lexical (BM25F) search works everywhere with no services. Vector search is an optional derived index in PostgreSQL with pgvector.

## Prerequisites

- PostgreSQL with the `pgvector` extension.
- An OpenAI-compatible embeddings endpoint.

## Environment

Credentials come from the process environment, never from vault configuration:

    export GNOSIS_DATABASE_URL="postgres://user:pass@host:5432/dbname"
    export GNOSIS_EMBEDDING_URL="https://api.example.com/v1/embeddings"
    export GNOSIS_EMBEDDING_MODEL="text-embedding-3-small"
    export GNOSIS_EMBEDDING_API_KEY="..."

## Sync and query

    gnosis index knowledge      # replace this workspace's derived chunks atomically
    gnosis search knowledge "conceptual question" --backend vector

## Notes

- Markdown stays authoritative; the database is disposable derived state. Re-run `gnosis index knowledge` after edits — stale indexes are detected by content fingerprint and reported.
- Without these variables the default vector backend fails fast; pass `--backend lexical` or configure the environment.
- Details and rationale: [knowledge model](../explanation/knowledge-model.md).
```

`docs/documentation/how-to/serve-over-mcp.md`:

```markdown
# Serve over MCP

Expose read-only vault knowledge to agents over the Model Context Protocol, or browse it over HTTP.

## stdio (agent subprocess)

    gnosis serve mcp

Tools: `get_vaults`, `get_concepts`, `get_page`, `search_knowledge`. Register it in your agent's MCP configuration as a stdio server pointing at the vault (the server honors `--vault`).

## HTTP + streamable MCP

    gnosis serve http --address 127.0.0.1:8080

- `GET /` — the atlas graph UI.
- `GET /api/v1/vaults|concepts|pages|page?uri=...|graph|search?q=...` — JSON.
- `POST /mcp` — streamable HTTP MCP endpoint with the same read-only tools.

## Guarantees

The serve paths are read-only by design; knowledge changes only through `apply page`. See the [serve-read-only decision](../../decisions/serve-read-only-knowledge-over-mcp-stdio.md).
```

`docs/documentation/how-to/compose-vaults.md`:

```markdown
# Compose vaults

Combine several vaults into one deterministic knowledge view.

## Import local vaults

    gnosis apply workspace --vault-name team --vault-root /path/to/team/docs

Each `[[vaults]]` entry in `gnosis.toml` adds an imported vault. The composed view resolves in order: the local vault first, then imports in declared order, then the embedded core bundle — the first source wins for a given vault-relative path. Use `gnosis get vaults` to inspect the effective order.

## Address pages across vaults

Canonical URIs are `gnosis://<vault>/<path>`. The `_` authority (`gnosis://_/procedures/vault/query-vault.md`) matches any vault and is the portable way to reference shared records.

## GitHub wiki backend

    gnosis apply workspace --github-wiki owner/repo --name wiki

The wiki repository is cached as a local git working tree, pulled fast-forward on load, and committed and pushed after mutations. Treat it as a shared vault: coordinate writers, because push conflicts surface as errors.

## Rules

- Imports are read-mostly: `apply page` writes only to the local vault (or to a github-wiki backend, which publishes).
- Detect cycles with `gnosis validate vault` after changing imports.
```

- [x] Verify every relative link target in the eight files exists (note `../../decisions/serve-read-only-knowledge-over-mcp-stdio.md` points into the vault — allowed, because documentation may reference vault pages; only the reverse is forbidden).
- [x] Commit: `docs: add how-to guides`.

### Task 4: Reference

**Load:** `gnosis --help` and each subcommand's `--help` (run them, never guess), `internal/vault/config.go`, `docs/concepts/*.md`, `gnosis get procedures`.
**Files:** create `docs/documentation/reference/index.md` plus four reference files.
**Interfaces:** produces the information-oriented quadrant.

- [x] Write the five files with exactly these contents (verifying every command and flag against the built binary before writing):

`docs/documentation/reference/index.md`:

```markdown
# Reference

Information-oriented lookups.

- [CLI](cli.md) — every command, flag, and output convention.
- [Configuration](configuration.md) — `gnosis.toml` fields and resolution order.
- [Concept types](concept-types.md) — the eleven bundled types and their fields.
- [Procedures](procedures.md) — every executable procedure and when it applies.
```

`docs/documentation/reference/cli.md`:

```markdown
# CLI reference

Grammar: `gnosis <verb> <resource> [flags]`, with one persistent `--vault <path>` flag (default `.`).

## Output conventions (AXI)

- All data on stdout is TOON; errors are TOON with exit code 1; usage errors exit 2 and list valid flags.
- List commands support `--fields` with defaults and an allowlist; long single-record output truncates with a `--full` escape hatch.
- Empty results always print a definitive empty state plus next-step hints.

## Commands

| Command | Purpose | Key flags |
|---|---|---|
| `gnosis` | Home view: bin, workspace, counts, hints | — |
| `gnosis create vault` | Scaffold an OKF vault | `--name`, `--force`, `--concepts` |
| `gnosis apply workspace` | Write `gnosis.toml` imports or a GitHub wiki backend | `--vault-name`, `--vault-root`, `--github-wiki owner/repo`, `--name`, `--force` |
| `gnosis apply page <uri>` | Validate and write one typed page (input `--filename` or stdin) | `--filename/-f`, `--update` |
| `gnosis get vaults` | List effective vaults in precedence order | `--fields` |
| `gnosis get concepts [type]` | List concept types or records of one exact type | `--fields` |
| `gnosis get pages [uri]` | List effective pages or read one exact page | `--fields`, `--full` |
| `gnosis get procedures [uri]` | List executable procedures or read one contract | `--tags`, `--fields`, `--full` |
| `gnosis get directives` | List directives with derived checkbox progress | `--fields` |
| `gnosis search knowledge <question>` | Bounded retrieval over the composed vault | `--backend lexical|vector`, `--top`, `--max-read`, `--depth`, `--fields` |
| `gnosis graph neighbors <uri>` | Traverse directed links | `--direction out|in|both`, `--relation`, `--depth` |
| `gnosis graph path <from> <to>` | Find a link path between two pages | `--direction`, `--relation`, `--depth` |
| `gnosis index vault` | Regenerate `index.md` files | — |
| `gnosis index knowledge` | Sync the pgvector semantic index | — |
| `gnosis validate vault` | Frontmatter, links, contracts, reserved files | exit 1 on errors |
| `gnosis serve mcp` | Read-only MCP over stdio | — |
| `gnosis serve http` | Atlas UI, JSON API, streamable MCP | `--address` |
| `gnosis version` | Print the version | — |
| `gnosis completion <shell>` | Shell completion scripts | — |

## URIs

Canonical form `gnosis://<vault>/<path>`; the `_` authority matches any vault. Relative links stay valid inside a vault; reads render them to canonical URIs.
```

`docs/documentation/reference/configuration.md`:

```markdown
# Configuration reference

One TOML file, resolved in this order (first hit wins): `gnosis.local.toml`, `gnosis.toml`, then `~/.config/gnosis.toml`. Inside a git work tree with no config file, gnosis defaults to a vault named `local` rooted at `docs/` with strict relative links and index/log disabled.

## `[vault]`

| Field | Default | Meaning |
|---|---|---|
| `vault_name` | `local` (in git) | Vault identity; used in URIs. |
| `vault_root` | `docs` (in git) | Directory holding the knowledge pages. |
| `backend` | none | Storage backend; `github-wiki` clones the repo's wiki. |
| `repository` | none | `owner/repo` for the GitHub wiki backend. |
| `link_format` | `relative` | Preferred body-link style: `relative` or `absolute`. |
| `link_format_strict` | `true` in git | Promote link-format violations from warnings to errors. |
| `vault_index` | `false` | Require and generate per-directory `index.md` files. |
| `vault_log` | `false` | Require a root `log.md`; procedures append entries. |

## `[[vaults]]`

Each block imports one vault: `vault_name`, `vault_root`, optional `backend`/`repository`. The composed view is deterministic: local first, imports in declared order, the embedded core bundle last; first source wins per vault-relative path. Cycles are validation errors.

## Environment

`GNOSIS_DATABASE_URL`, `GNOSIS_EMBEDDING_URL`, `GNOSIS_EMBEDDING_MODEL`, `GNOSIS_EMBEDDING_API_KEY` configure the optional vector backend. Secrets never live in `gnosis.toml`.

## Reserved names

`index.md` and `log.md` are reserved files; a root-level `documentation/` directory is exempt from the vault entirely.
```

`docs/documentation/reference/concept-types.md`:

```markdown
# Concept types

Eleven bundled types. Each definition lives in `docs/concepts/` (overridable in any vault) and declares a `path` where its instances belong. List them with `gnosis get concepts`.

## Intent types

| Type | Path | Purpose | Key fields |
|---|---|---|---|
| Purpose | `purpose.md` (singleton) | Enduring outcomes and boundaries | — |
| Decision | `decisions/` | Durable non-obvious choices | `supersedes` |
| Directive | `directives/` | Executable implementation handoffs | `status` (draft/open/blocked/done), checkbox tasks, deltas, scenarios |
| Procedure | `procedures/` | Agent-executable contracts | `description`, `tags`, `invocation` |

## Content types

| Type | Path | Answers | Key fields |
|---|---|---|---|
| Concept | `concepts/` | What is true? | `status`, `confidence`, `source`, `tier`, `superseded_by` |
| Entity | `entities/` | Who is involved? | `kind`, `status` |
| Resource | `resources/` | Where is something? What can the agent use? | `kind`, `resource`, `status` |
| Event | `events/` | What happened? What was observed? | `occurred_at`, `actor`, `source`, `status` |
| Memory | `memories/` | Durable scoped facts, preferences, observations | `scope`, `observed_at`, `hash`, `entities`, `status` |
| Reflection | `reflections/` | What lesson was learned? | `status`, `confidence`, `superseded_by` |
| Policy | `policies/` | What should or must be done? When does it apply? | `status`, `applies_to`, `superseded_by` |

## Shared provenance metadata

Optional on content types: `status`, `confidence` (0.0–1.0), `source`, `observed_at`, `valid_from`, `superseded_by`, `tier` (core/supporting/peripheral), `entities`. Unknown frontmatter is always preserved verbatim.

## Body conventions

Unmarked claims are extracted from sources; `^[inferred]` marks agent generalizations; `^[ambiguous]` marks unresolved disagreement. Typed `relationships:` frontmatter adds semantic edges (`extends`, `implements`, `uses`, `contradicts`, `derived_from`, `causes`, `depends_on`, `owns`, `related_to`).
```

`docs/documentation/reference/procedures.md`:

```markdown
# Procedures

Procedures are executable contracts an agent loads with `gnosis get procedures <uri> --full`. Discovery: `gnosis get procedures --tags gnosis,<family>`.

## Vault family

| Procedure | Use when |
|---|---|
| `query-vault` | Answering a question from recorded knowledge (tiered cost ladder). |
| `ingest-knowledge` | Supplied evidence should create or update concept pages. |
| `remember` | An episode or statement should become durable scoped memories. |
| `recall` | Answering from scoped Memory records. |
| `link-pages` | Pages mention known knowledge without linking it. |
| `maintain-vault` | Auditing or repairing vault integrity (consolidation pass). |
| `create-concept-type` | A vault needs a new or refined ontological category. |
| `refining-procedure` | An existing procedure must be rewritten for reliable execution (explicit request). |

## Development family

| Procedure | Use when |
|---|---|
| `planning-directives` | Turning a request into validated directive handoffs. |
| `implementing-directive` | Exactly one open directive must be implemented and verified. |
| `managing-intents` | A Purpose, Decision, or Directive record must be created, read, updated, or deleted. |
| `maintain-intents` | Explicitly requested archival of completed directives (merge deltas, compact decisions). |
| `debugging-methodically` | Diagnosing any bug or unexpected behavior before fixing. |

## Rules

- Procedures are selected and executed by the controlling agent; never spawn a selector subagent.
- Follow the `Inputs`/`Process`/`Completion` sections, or numbered `STEP` sections for multi-step procedures.
- Every procedure run ends with `gnosis validate vault` when it writes.
```

- [x] Verify: run `./dist/gnosis --help`, `./dist/gnosis get --help`, `./dist/gnosis apply --help`, `./dist/gnosis create --help`, `./dist/gnosis search --help`, `./dist/gnosis graph --help`, `./dist/gnosis index --help`, `./dist/gnosis serve --help` (after `mise run build`) and confirm every row of the CLI table; correct any drift before writing.
- [x] Commit: `docs: add reference quadrant`.

### Task 5: Explanation

**Load:** `docs/purpose.md`, `docs/decisions/*.md`, `knowledge-research.pdf` (repository root), `internal/vault/` layout.
**Files:** create `docs/documentation/explanation/index.md` plus four explanation files.
**Interfaces:** produces the understanding-oriented quadrant.

- [x] Write the five files with exactly these contents:

`docs/documentation/explanation/index.md`:

```markdown
# Explanation

Understanding-oriented design reading.

- [Knowledge model](knowledge-model.md) — types, representations, and access mechanisms.
- [Memory architecture](memory-architecture.md) — scoped agent memory as explicit records.
- [Intent lifecycle](intent-lifecycle.md) — purpose, decisions, directives, and deltas.
- [Code architecture](architecture.md) — how the CLI and vault library fit together.
```

`docs/documentation/explanation/knowledge-model.md`:

```markdown
# Knowledge model

gnosis separates three ideas that are often conflated:

- **Knowledge types** — facts, experiences, procedures, relationships, policies, preferences, observations, and reflections. These are what knowledge *means*.
- **Representations** — documents, vectors, graphs, tables, events, code. These are how knowledge is *stored*.
- **Access mechanisms** — lexical search, vector similarity, graph traversal, tool calls. These are how knowledge is *reached*.

A vector database or a wiki is never a knowledge type; it is a representation with access mechanisms. This is why gnosis keeps Markdown authoritative and treats pgvector as a disposable derived index.

## Type coverage

The bundled concept types cover the full taxonomy of agent knowledge: semantic/factual (Concept), episodic and perceptual (Event), procedural (Procedure), causal (typed `causes`/`depends_on` relationships), conditional and normative (Policy), social (Entity), resource and tool (Resource), preference/persona (scoped Memory), metacognitive (`status`/`confidence` metadata plus `^[inferred]`/`^[ambiguous]` markers), and reflective (Reflection). Working and short-term knowledge deliberately never enters the vault — it belongs to the agent's context.

## Authority and provenance

Every page carries provenance (`origin`, content `revision`, optional `source`/`observed_at`/`valid_from`) so retrieval can rank curated and observed knowledge above inference. Supersession (`superseded_by`, Decision `supersedes`) and archived records preserve negative knowledge: what was checked and no longer holds.

## Temporal roles

Events and memories carry absolute dates; consolidation turns episodes into Concepts and Reflections through explicit procedures; forgetting is archival with retained audit, never silent deletion.
```

`docs/documentation/explanation/memory-architecture.md`:

```markdown
# Memory architecture

gnosis implements agent memory in the mem0 style, adapted to a plain-file vault.

## Design

- **Memory pages** are the store: one self-contained statement per page, scoped `user | agent | session | run`, with `observed_at`, `entities`, and a content `hash`.
- **remember** is the write path: extract durable candidates, suppress exact duplicates by hash, retrieve the nearest existing memories, then reconcile each candidate as ADD (new page), UPDATE (revise in place), DELETE (archive with a reason), or NONE. Every operation is an explicit, validated page write.
- **recall** is the read path: scoped retrieval combining lexical search (vector optional), entity-match boosts, and recency, returning provenance with every answer.
- **Audit** is git history plus retained archived pages — the vault needs no separate history database.

## Why this shape

mem0's own trajectory informed it: their v3 moved to accumulate-and-rank over aggressive curation and removed external graph databases, because ranking handles currency and a link graph covers entity context. Plain pages give provenance, portability, and review for free. What gnosis deliberately omits: background summarizers, external graph stores, reranker services, and implicit memory writes — every mutation is author-visible.

## Relationship to durable knowledge

Memories are not a dumping ground. When a memory graduates into project truth, the owning procedure converts it: facts become Concepts, lessons become Reflections, settled choices become Decisions.
```

`docs/documentation/explanation/intent-lifecycle.md`:

```markdown
# Intent lifecycle

Intent is what governs work: one Purpose, many Decisions, and executable Directives.

## States

- **Purpose** is a singleton holding enduring outcomes and boundaries; it changes rarely and only with author confirmation.
- **Decisions** are append-only in effect: a changed choice creates a new record linked by `supersedes`, preserving the history of why.
- **Directives** move `draft → open → blocked|done`, and each transition is owned by a procedure — planning finalizes, implementing completes, replanning reopens. Status is never assigned from assertion.

## OpenSpec strategies

Directives behave like spec deltas rather than prose tasks:

1. **Delta semantics** — `# Purpose/Decision Changes` declares `## Added`/`## Modified`/`## Removed` effects on living intent records.
2. **Scenarios** — behavior acceptance criteria use `#### Scenario:` blocks with `**WHEN**`/`**THEN**` bullets, making done-ness unambiguous.
3. **Derived progress** — checkbox tasks let `gnosis get directives` compute progress instead of trusting status text.
4. **Strict validation** — the validator enforces the directive contract like code.
5. **Archive as merge** — `maintain-intents` folds a done directive's deltas into the living records, compacts remaining durable choices into Decisions, and removes the directive. The living records are the audit trail.

## Why procedures own transitions

Lifecycle rules live in Concept Type definitions and procedures — data, not code — so any agent that can read the vault can govern work the same way, and the rules evolve as knowledge instead of releases.
```

`docs/documentation/explanation/architecture.md`:

```markdown
# Code architecture

gnosis is a small Go module with two packages and no framework beyond cobra.

## Layout

- `cmd/gnosis/` — the CLI. Verb-resource commands, TOON output (AXI conventions), the atlas UI (`ui.html`), and HTTP/MCP servers.
- `internal/vault/` — the vault library: configuration (`config.go`), page model and frontmatter (`page.go`), multi-vault composition (`search.go`, `vaults.go`, `bundle.go`), lexical retrieval (`retrieval.go`), pgvector semantics (`semantic.go`), graph (`agent.go`, `links.go`), contracts (`procedure.go`, `directive.go`), writes (`write.go`), indexes (`index.go`), validation (`validate.go`), scaffolding (`scaffold.go`), backends (`backend.go`).
- `docs/` — the project's own vault and the embedded core bundle (`embed.go` bundles concept types and procedures into the binary).
- `plugins/gnosis/` — the agent plugin manifests and the two gateway skills.
- `integration/` — the Harbor coding-agent fixture and verifier.

## Key design choices

- **Markdown authoritative** — every store except the optional pgvector index is plain files; the database is disposable derived state.
- **Composition** — vaults layer local → imports → core bundle with first-wins precedence, giving one deterministic view without copying.
- **Contracts over code** — procedures and concept lifecycles are vault records; Go enforces only structural contracts (procedure and directive schemas, links, reserved names).
- **Replaceable boundaries** — retrieval backends and storage backends sit behind small interfaces; lexical always works, vector and github-wiki are opt-in.
- **Read-only serving** — MCP and HTTP expose knowledge without mutation paths; writes exist only through `apply page`.

## Testing

Every Go file has a sibling test; `mise run checks` is the full gate: gofmt, vet, tests with the race detector, build, and vault validation.
```

- [x] Verify every relative link target in the five files exists.
- [x] Commit: `docs: add explanation quadrant`.

### Task 6: Final verification

**Load:** all of the above.
**Files:** none.
**Interfaces:** produces the completion evidence.

- [x] Run `find docs/documentation -name '*.md' | wc -l`; expect 22 (1 landing + 3 tutorials + 8 how-to + 5 reference + 5 explanation).
- [x] Run a link audit: for each `.md` under `docs/documentation`, extract `](...)` targets and confirm each relative target file exists; expect zero broken links.
- [x] Run `./dist/gnosis validate vault`; expect `status: valid`, `warnings: 0` (documentation untouched by the vault).
- [x] Run `./dist/gnosis get pages | grep -c documentation`; expect 0 matches (documentation is not vault knowledge).
- [x] Run `mise run checks`; expect all green.
- [x] Commit: `docs: complete diataxis documentation` (only if files changed).

# Acceptance criteria

- The exemption works — run the three new Go tests (`TestGenerateIndexesSkipsRootDocumentation`, `TestSearchSourceExcludesRootDocumentation`, `TestValidateSkipsRootDocumentationDir`); expect ok; run `./dist/gnosis get pages`; expect no `documentation/` URIs.
- All four quadrants exist and interlink — inspect `docs/documentation/`; expect the landing page plus tutorials, how-to, reference, and explanation trees with zero broken relative links (Task 6 audit evidence).
- Docs match reality — every command, flag, config field, concept type, and procedure named in the docs exists in the built binary or vault (Task 4 `--help` verification evidence plus spot-checks of each reference table).
- Vault integrity holds — run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- No regressions — run `mise run checks`; expect all green.
