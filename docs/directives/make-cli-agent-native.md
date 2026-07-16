---
type: Directive
title: Make the CLI agent-native
description: Rebuild the gnosis command surface around kubectl-style verb-resource commands and AXI-compliant TOON output, help, errors, and home context.
status: done
---

# Goal

Make every ordinary gnosis CLI interaction predictable for agents: `gnosis [verb] [resource] [name] [flags]`, compact TOON on stdout, actionable structured errors, concise local help, and a useful no-argument workspace view.

# Requirements packet

## Outcome

- Replace the mixed command grammar with one kubectl-shaped surface:

```text
gnosis
gnosis get vaults
gnosis get concepts [TYPE]
gnosis get pages [GNOSIS_URI]
gnosis get procedures [GNOSIS_URI]
gnosis search knowledge QUESTION
gnosis graph neighbors GNOSIS_URI
gnosis graph path FROM_URI TO_URI
gnosis create vault
gnosis apply workspace
gnosis apply page GNOSIS_URI
gnosis index vault|knowledge
gnosis validate vault
gnosis serve http|mcp
gnosis version
gnosis completion SHELL
```

- Make `--vault` a root persistent flag accepted before or after subcommands.
- Emit TOON for every ordinary success, empty state, error, and help response. Keep MCP frames, HTTP JSON bodies, server diagnostics on stderr, and shell completion scripts in their protocol-native formats.
- Remove `read`, `write`, `scaffold`, `setup`, `procedure`, `--json`, graph URI flags, and bare `validate` completely. Do not add aliases or deprecation shims.
- Default lists to three or four decision-bearing fields, include exact counts, support command-specific `--fields`, and state zero-result context explicitly.
- Truncate page and procedure detail content at 1,000 characters by default, report the total character count, and expose `--full` only on detail-capable commands.
- Make no-argument `gnosis` return executable identity, one-sentence description, current workspace counts, compact vault/type rows, and contextual next commands.
- Make every `--help` response identify one command, exact usage, available local/inherited flags with defaults, subcommands where applicable, and two or three runnable examples.
- Return exit 0 for success and idempotent no-ops, exit 1 for runtime failure, and exit 2 for invalid commands, flags, arguments, or option values. Render errors on stdout with inline valid usage/flags for usage failures.

## In scope

- `cmd/gnosis` command construction, output boundary, error/exit handling, and command tests.
- TOON tags on CLI-facing `internal/vault` response structs; no storage or HTTP/MCP contract change.
- Active procedure, plugin-skill, integration, and build-check invocations affected by removed commands.
- One pinned, dependency-free TOON encoder module plus conformance tests for every shape gnosis emits.

## Out of scope

- README.md.
- HTTP routes, HTTP JSON response schemas, MCP tool names or structured JSON schemas.
- Agent session-hook installation, generated skills, semantic retrieval behavior, vault data-model changes, or compatibility aliases.
- Rewriting historical completed directive evidence that records commands actually run.

## Constraints and governing sources

- Repository `AGENTS.md`: use lower-case gnosis; do not modify README.md; remove obsolete behavior and references instead of preserving compatibility.
- [Reshape CLI around resource verbs](reshape-cli-around-resource-verbs.md) @ `sha256:430e7ff899caa8593997a5796410c82e94af7d7883e7c47d035aacd898169658`: keep Cobra and command-per-file layout, preserve vector/lexical behavior, and do not restore removed aliases.
- TOON Specification v3.3, 2026-05-21, `https://github.com/toon-format/spec/blob/main/SPEC.md`: UTF-8/LF, ordered objects, exact array counts, delimiter-aware quoting, no trailing whitespace/newline.
- Kubernetes v1.36 kubectl reference, `https://kubernetes.io/docs/reference/kubectl/`: canonical grammar is `kubectl [command] [TYPE] [NAME] [flags]`; operations may retain focused subcommands where the operation itself is the stable top-level verb.
- AXI skill contract supplied by the author: TOON stdout, minimal schemas, truncation with an escape hatch, counts, definitive empty states, structured errors, no prompts, content-first root, contextual disclosure, and concise per-command help.

## Resolved choices

- `get pages [URI]` replaces both the missing page list and legacy exact `read`; `apply page URI` replaces `write`.
- `get procedures [URI]` lists executable procedures without a URI and returns one execution contract with a URI; `--tags` filters lists and `--full` prevents detail truncation.
- `create vault` replaces `scaffold`; `apply workspace` replaces `setup`; `validate vault` makes the validated resource explicit.
- `graph neighbors/path` and `serve http/mcp` remain focused operation groups, like kubectl `rollout` and `config`, but page identities become positional arguments.
- `--fields` is list-only. Long-form authored metadata remains reachable through `get pages <uri>` instead of being repeated in list output.
- Use `github.com/toon-format/toon-go` pinned at `v0.0.0-20251202084852-7ca0e27c4e8c`. It is the official, MIT-licensed, dependency-free Go implementation and exposes ordered `toon.Object` values and `toon` struct tags. The standard library has no TOON encoder. `github.com/alpkeskin/gotoon@v0.1.1` is rejected because it normalizes structs and maps into unordered maps and exposes a non-standard `[#N]` mode. Because the official module is unreleased and predates TOON v3.3, gnosis must constrain it behind one output helper and test quoting, ordering, array counts, empty arrays, and the absence of trailing whitespace/newlines for emitted shapes.

## Acceptance evidence

- Root: `go run ./cmd/gnosis` exits 0 and emits valid TOON containing `bin`, `description`, current counts, compact live rows, and contextual help.
- Grammar: root help exposes only the command tree above; tests prove every removed top-level command/flag fails with exit 2 and no compatibility alias.
- Lists: focused tests decode TOON from vault, concept, page, procedure, and search lists; counts equal result lengths, default rows have at most four fields, requested fields are honored, unknown/duplicate fields fail before vault work, and empty lists say what zero means.
- Details: page and procedure details expose a 1,000-character preview, total size, a `--full` hint only when truncated, and exact complete content under `--full`.
- Errors/help: subprocess tests prove usage errors are structured on stdout with exit 2 and inline usage/valid flags; runtime errors are structured on stdout with exit 1; stderr remains empty except server diagnostics; every command help contains exact usage, flags/defaults, and two or three examples.
- Mutations: create/apply/index/validate confirmations are structured; repeating an already-satisfied safe mutation is an acknowledged exit-0 no-op.
- Protocols: existing HTTP and MCP tests pass unchanged in schema and transport behavior; completion output remains executable shell code.
- Active knowledge: no active procedure, plugin skill, integration fixture, or build task invokes a removed command.
- Quality gate: `git diff --check`, `gofmt -l`, focused command tests, `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, and `go run ./cmd/gnosis validate vault` all pass; README.md is unchanged.

# Architecture

Keep Cobra and the current command-per-file layout. Add one `output.go` boundary that owns the TOON encoder, ordered objects, field selection, content truncation, custom help, contextual hints, and structured errors. Command handlers continue to call typed `internal/vault` APIs and build output values only after those APIs succeed. Root options own the persistent vault path. HTTP and MCP bypass the CLI output boundary.

# Implementation plan

## Task 1: Lock the final grammar, home, help, and error contract in red tests

**Load:** all `cmd/gnosis/*.go`, current command tests, Cobra flag/help APIs, and the requirements packet above.

**Files:** create `cmd/gnosis/command_test.go` and `cmd/gnosis/output_test.go`; modify focused command tests only to replace obsolete command/JSON assertions.

**Exact interfaces:**

```go
type rootOptions struct {
	vaultPath string
}

type commandError struct {
	cause error
	usage bool
	path  string
	help  []string
}

func runContext(context.Context, []string, io.Writer, io.Writer) error
func exitCode(error) int
func writeCommandError(io.Writer, error) error
func writeTOON(io.Writer, toon.Object) error
func setCommandHelp(*cobra.Command, ...string)
```

- Assert no-argument success and exact live-home keys.
- Assert root and child help decode as TOON and contain exact command, usage, flags/defaults, subcommands, and two or three examples.
- Assert final command paths succeed or reach their vault/runtime boundary, and removed paths/`--json`/old graph flags/bare validate fail as usage errors.
- Add a subprocess helper that builds gnosis once and checks stdout, stderr, and exit status for success, usage failure, runtime failure, completion, MCP, and HTTP-diagnostic channel exceptions.
- Run `go test ./cmd/gnosis -run 'Test(Command|Home|Help|Error|Exit|Removed)' -count=1`; expected red is missing new command paths and non-TOON output.

## Task 2: Add one conforming TOON boundary

**Load:** Task 1 failures, TOON v3.3 sections 2, 5-12, and the pinned encoder source/tests.

**Files:** modify `go.mod` and `go.sum`; create `cmd/gnosis/output.go`; add `toon` tags beside existing `json` tags only on CLI-facing response structs in `internal/vault/agent.go`, `internal/vault/procedure.go`, `internal/vault/retrieval.go`, and `internal/vault/semantic.go`.

**Exact interfaces and shapes:**

```go
const detailPreviewLimit = 1000

type fieldSelector struct {
	names []string
}

func parseFields(string, []string, []string) (fieldSelector, error)
func (fieldSelector) object(func(string) (any, bool)) toon.Object
func listOutput(string, int, []toon.Object, string, []string) toon.Object
func truncate(string, bool) (preview string, total int, truncated bool)
func executablePath() string
```

- `writeTOON` calls only the pinned encoder with ordered `toon.Object` roots and writes its bytes without adding a trailing newline.
- `parseFields` accepts a comma-separated list, preserves requested order, rejects empty, unknown, and duplicate names as usage errors, and applies command defaults when omitted.
- Default schemas are vaults `{vault,kind,root}`, concept types `{type,description,uri}`, document lists `{uri,title,type}`, procedure lists `{uri,title,description}`, and search candidates `{uri,title,type,score}`.
- Root/list objects put `count` before rows and `help` last. Empty results also include `message: 0 <resource context> found`.
- Conformance tests cover ordered keys, uniform tabular rows, strings requiring quotes/escapes, exact `[N]`, empty `[0]:`, UTF-8/LF, and no trailing whitespace/newline.
- Run `go test ./cmd/gnosis -run 'TestTOON|TestFields|TestTruncate' -count=1`; expect green.

## Task 3: Reparent handlers and delete obsolete entry points

**Load:** Task 1/2 tests and every current command constructor/caller.

**Files:** modify `cmd/gnosis/main.go`, `get.go`, `search.go`, `index.go`, `graph.go`, `validate.go`, `serve.go`, and `serve_http.go`; create `create.go` and `apply.go`; delete `read.go`, `write.go`, `scaffold.go`, `setup.go`, and `procedure.go`; rename/update their tests with `apply_patch` rather than retaining obsolete test files.

**Exact constructor tree:**

```go
newRootCommand(stdout, stderr)
newGetCommand(options, stdout)       // vaults, concepts, pages, procedures
newSearchCommand(options, stdout)    // knowledge
newGraphCommand(options, stdout)     // neighbors URI; path FROM TO
newCreateCommand(options, stdout)    // vault
newApplyCommand(options, input, stdout) // workspace; page URI
newIndexCommand(options, stdout)     // vault; knowledge
newValidateCommand(options, stdout, stderr) // vault
newServeCommand(options)             // http; mcp
```

- Root `RunE` renders the live home instead of returning missing-command usage. Register command groups for basic (`create`, `get`, `apply`), knowledge (`search`, `graph`, `index`), workspace (`validate`, `serve`), and other (`version`, `completion`, `help`) operations.
- Make `--vault` the only root persistent data-context flag. Keep operation-specific flags local and reject every unknown flag before any vault/API call.
- `get pages` calls `vault.ListPages`; `get pages URI` calls `vault.ReadPage`; `get procedures` calls `vault.DiscoverProcesses`; `get procedures URI` calls `vault.InvokeProcess`.
- Page/procedure detail commands reject list-only flags in detail mode and list commands reject detail-only flags where their use has no effect.
- `apply page` reads stdin or `--filename`; `apply workspace` retains existing local/GitHub-wiki semantics; `create vault` retains scaffold options; outputs acknowledge `changed` or `no-op` explicitly.
- Keep search backend defaults and query validation unchanged, but render candidates/count/path/should-read as ordered TOON and add list-only `--fields`.
- Render graph, index, validation, version, and mutation results through `writeTOON`. HTTP JSON, MCP structured results/transports, and completion scripts remain unchanged.
- Format, then run `go test ./cmd/gnosis -count=1` and `go test ./internal/vault -count=1`; expect green.

## Task 4: Remove active obsolete references and verify the integrated result

**Load:** all non-README active matches for removed commands and flags; do not edit completed directive evidence solely to modernize history.

**Files:** update active `docs/procedures/**/*.md`, `plugins/gnosis/skills/**/*.md`, relevant integration fixtures, and `mise.toml`; create or update the current directive evidence. Never modify README.md.

**Exact replacements:**

```text
procedure discovery --tags X       -> get procedures --tags X
procedure invoke --uri URI          -> get procedures URI --full
read URI [--json]                   -> get pages URI --full
write URI --filename FILE           -> apply page URI --filename FILE
graph neighbors --uri URI           -> graph neighbors URI
graph path --from A --to B           -> graph path A B
scaffold ...                         -> create vault ...
setup ...                            -> apply workspace ...
validate [--vault ROOT]              -> validate vault [--vault ROOT]
get concepts --json                 -> get concepts
```

- Run an active-path `rg` assertion for every removed form; expect no matches outside historical completed directives.
- Run focused acceptance commands against a temporary fixture and this repository, decode every ordinary stdout payload with the pinned TOON decoder, and check channel/exit behavior.
- Review the complete diff for scope, command consistency, schema size, self-correcting errors, idempotency, protocol isolation, stale references, and README preservation.
- Run the complete quality gate from the requirements packet with fresh output; set only this directive to `done` after every criterion is evidenced.

# Review bindings

- Repository root/HEAD: `/home/llin/Source/gnosis` @ `1c3f9af` on `main`; normal shared checkout, ahead of `origin/main` by five commits, clean at planning start.
- Requirements: this directive revision plus the exact governing source bindings above.
- Allowed paths: `cmd/gnosis/**`, CLI-facing tags in `internal/vault/**`, `go.mod`, `go.sum`, active `docs/procedures/**`, `plugins/gnosis/skills/**`, relevant `integration/**`, `mise.toml`, and this directive. README.md is forbidden.

# Planning review

## Purpose/decision pass

Reviewed: requirements packet; repository `AGENTS.md`; `gnosis://local/directives/reshape-cli-around-resource-verbs.md` @ `sha256:430e7ff899caa8593997a5796410c82e94af7d7883e7c47d035aacd898169658`; TOON v3.3; kubectl v1.36 reference; repository root/HEAD above.

Verdict: APPROVE

No Critical or Important findings. The delivery advances the existing resource-verb direction, preserves protocol contracts, honors the repository's breaking-removal policy, and does not imply a purpose or durable knowledge decision beyond the author-requested CLI architecture.

## Engineering pass

Reviewed: same bindings; current command constructors and tests; pinned official/community TOON sources; all active command references returned by repository search.

Verdict: APPROVE

No Critical or Important findings. The plan isolates the unreleased encoder behind one tested boundary, defines exact final paths and removals, covers channels and exit codes with subprocess evidence, preserves HTTP/MCP contracts, and keeps the change in one coupled directive because command paths, output schemas, help, errors, and active procedure invocations cannot ship independently.

# Finding dispositions

- No Critical or Important findings were produced by either planning pass.

# Implementation evidence

- Replaced the mixed surface with the kubectl-shaped verb/resource tree in this directive, one persistent `--vault`, positional resource identities, and no compatibility aliases.
- Added one ordered TOON output boundary for ordinary success, empty, help, and error payloads; preserved HTTP JSON, MCP frames, server diagnostics, and completion scripts in their native protocols.
- Added list counts, `--fields`, definitive empty messages, 1,000-character page/procedure previews with `--full`, structured mutation/no-op confirmations, and exit 0/1/2 handling.
- Process tests build the binary and verify TOON stdout, empty stderr, exit 2 usage failures with valid flags, exit 1 runtime failures, and native completion output.
- Removed active obsolete command references from procedures, plugin skills, the draft directive, and `mise.toml`; searches found no obsolete forms outside historical completed directives and intentional removal tests.
- Both modified plugin skills pass `quick_validate.py`; `README.md` has no diff.
- `go test ./cmd/gnosis ./internal/vault -count=1` passed.
- `mise run checks` passed gofmt, vet, fresh tests, race tests, build, and `gnosis validate vault`; validation reported 42 files and zero warnings.

# Purpose/Decision Changes

None. This delivery implements the approved command-interface directive without changing gnosis purpose or adding a durable product decision beyond the bound requirements.
