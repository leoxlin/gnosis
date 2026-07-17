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
| `gnosis apply workspace` | Write `gnosis.toml` imports or a GitHub wiki primary vault | `--import <path>` (repeatable), `--github-wiki owner/repo`, `--name`, `--force` |
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
