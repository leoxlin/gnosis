---
type: Directive
title: Adopt OpenSpec strategies in intent tools
description: Add directive delta semantics, scenario criteria, computed progress, and strict directive validation to the intent system.
status: done
---

# Goal

Implement [Adopt OpenSpec strategies for intent records](../decisions/adopt-openspec-intent-strategies.md): directive records gain delta semantics and scenario grammar, the CLI derives task progress from checkboxes, the validator enforces the Directive contract, and `maintain-intents` archives by merging declared deltas.

# Architecture

Directive contract parsing lives in one new `internal/vault/directive.go` beside `procedure.go`, reusing the existing `frontmatterScalar`, `markdownSectionsAtLevel`, and `markdownSectionBlocks` helpers. Validation hooks into the existing per-type dispatch in `validate.go:159-167`. Progress derivation is a pure function over the directive body. The CLI addition follows the existing `get.go` subcommand pattern exactly.

# Tech stack

Go 1.25, cobra, toon-go (existing dependencies only).

# Global constraints

- Follow [Adopt OpenSpec strategies for intent records](../decisions/adopt-openspec-intent-strategies.md).
- AXI output conventions: TOON on stdout, usage errors via `newUsageError`, `--fields` selector with defaults and allowlist, definitive empty state plus help hints.
- Red-green-refactor for every behavior change: write the failing test, run it red, implement, run focused then surrounding tests green.
- Do not change `QueryResult`, the page model, or any existing command's output shape.

# Scope

- New `internal/vault/directive.go`: directive contract parsing, task-progress derivation, `Directives` listing.
- `internal/vault/validate.go`: one dispatch branch calling the directive contract check.
- `cmd/gnosis/get.go`: new `get directives` subcommand.
- Tests: new `internal/vault/directive_test.go`; additions to `internal/vault/validate_test.go` and `cmd/gnosis/get_test.go`.
- Records: replace `docs/concepts/directive.md` and `docs/procedures/development/maintain-intents.md` with the complete contents below; apply the exact edits below to `docs/procedures/development/planning-directives.md` and `docs/procedures/development/implementing-directive.md`.

# Dependencies

None.

# Implementation plan

### Task 1: Directive contract parsing and progress derivation

**Load:** `internal/vault/procedure.go` (parsing helpers), `internal/vault/search.go` (`validationPages`), `docs/concepts/directive.md` (schema).
**Files:** create `internal/vault/directive.go`, create `internal/vault/directive_test.go`.
**Interfaces:** produces `Directives(root string) ([]DirectiveSummary, error)` and `parseDirective(fields frontmatterFields, body string) []string`.

- [x] Red: write `internal/vault/directive_test.go` exactly as below; run `go test ./internal/vault/ -run 'TestParseDirective|TestDirectives'`; expect compile failure (`undefined: Directives`).
- [x] Green: write `internal/vault/directive.go` exactly as below; run `go test ./internal/vault/ -run 'TestParseDirective|TestDirectives'`; expect ok.

`internal/vault/directive.go`:

```go
package vault

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// DirectiveType is the intent handoff concept type.
const DirectiveType = "Directive"

var directiveStatuses = []string{"draft", "open", "blocked", "done"}

var directiveTaskHeading = regexp.MustCompile(`^Task ([1-9][0-9]*): \S`)

// DirectiveSummary is one Directive record with its derived task progress.
type DirectiveSummary struct {
	DocumentRef
	Status     string `json:"status"`
	TasksTotal int     `json:"tasks_total"`
	TasksDone  int     `json:"tasks_done"`
}

// Directives lists every effective Directive record with derived progress.
func Directives(root string) ([]DirectiveSummary, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.validationPages()
	if err != nil {
		return nil, err
	}
	summaries := make([]DirectiveSummary, 0)
	for _, page := range pages {
		if page.document.Type != DirectiveType {
			continue
		}
		status, _ := frontmatterScalar(page.fields, "status")
		done, total := directiveTaskProgress(page.document.Body)
		summaries = append(summaries, DirectiveSummary{
			DocumentRef: page.document.Ref(),
			Status:      strings.TrimSpace(status),
			TasksTotal:  total,
			TasksDone:   done,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].URI < summaries[j].URI
	})
	return summaries, nil
}

// directiveTaskProgress counts checkbox tasks outside code fences.
func directiveTaskProgress(body string) (done, total int) {
	inFence := false
	fence := ""
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fence = marker
			} else if marker == fence {
				inFence = false
				fence = ""
			}
			continue
		}
		if inFence {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "- [ ]"):
			total++
		case strings.HasPrefix(trimmed, "- [x]"), strings.HasPrefix(trimmed, "- [X]"):
			total++
			done++
		}
	}
	return done, total
}

// parseDirective checks the Directive contract and returns every problem.
func parseDirective(fields frontmatterFields, body string) []string {
	problems := []string{}
	status, scalar := frontmatterScalar(fields, "status")
	status = strings.TrimSpace(status)
	if !scalar || status == "" {
		problems = append(problems, "directive requires non-empty \"status\" frontmatter")
	} else if !slices.Contains(directiveStatuses, status) {
		problems = append(problems, fmt.Sprintf("directive status %q must be one of: %s", status, strings.Join(directiveStatuses, ", ")))
	}

	sections, duplicates := markdownSectionsAtLevel(body, 1)
	for _, name := range duplicates {
		problems = append(problems, fmt.Sprintf("duplicate section %q", name))
	}
	for _, name := range []string{"Goal", "Scope", "Acceptance criteria"} {
		if strings.TrimSpace(sections[name]) == "" {
			problems = append(problems, fmt.Sprintf("missing required section %q", name))
		}
	}
	problems = append(problems, directivePlanProblems(sections["Implementation plan"])...)
	problems = append(problems, directiveDeltaProblems(sections["Purpose/Decision Changes"])...)
	problems = append(problems, directiveScenarioProblems(sections["Acceptance criteria"])...)
	return problems
}

func directivePlanProblems(plan string) []string {
	if strings.TrimSpace(plan) == "" {
		return nil
	}
	blocks := markdownSectionBlocks(plan, 3)
	if len(blocks) == 0 {
		return []string{"implementation plan requires at least one \"### Task <N>:\" section"}
	}
	problems := []string{}
	for index, block := range blocks {
		match := directiveTaskHeading.FindStringSubmatch(block.Title)
		if match == nil {
			problems = append(problems, fmt.Sprintf("invalid task heading %q; want \"Task <N>: <deliverable>\"", block.Title))
			continue
		}
		if match[1] != strconv.Itoa(index+1) {
			problems = append(problems, fmt.Sprintf("task %q is out of order; expected Task %d", block.Title, index+1))
		}
		_, total := directiveTaskProgress(block.Body)
		if total == 0 {
			problems = append(problems, fmt.Sprintf("task %q requires at least one checkbox step", block.Title))
		}
	}
	return problems
}

func directiveDeltaProblems(changes string) []string {
	if strings.TrimSpace(changes) == "" {
		return nil
	}
	problems := []string{}
	for _, block := range markdownSectionBlocks(changes, 2) {
		if !slices.Contains([]string{"Added", "Modified", "Removed"}, block.Title) {
			problems = append(problems, fmt.Sprintf("invalid delta section %q; want Added, Modified, or Removed", block.Title))
			continue
		}
		if strings.TrimSpace(block.Body) == "" {
			problems = append(problems, fmt.Sprintf("delta section %q requires at least one entry", block.Title))
		}
	}
	return problems
}

func directiveScenarioProblems(criteria string) []string {
	problems := []string{}
	for _, block := range markdownSectionBlocks(criteria, 4) {
		name, ok := strings.CutPrefix(block.Title, "Scenario: ")
		if !ok || strings.TrimSpace(name) == "" {
			problems = append(problems, fmt.Sprintf("invalid scenario heading %q; want \"Scenario: <name>\"", block.Title))
			continue
		}
		if !strings.Contains(block.Body, "**WHEN**") {
			problems = append(problems, fmt.Sprintf("scenario %q requires a **WHEN** bullet", name))
		}
		if !strings.Contains(block.Body, "**THEN**") {
			problems = append(problems, fmt.Sprintf("scenario %q requires a **THEN** bullet", name))
		}
	}
	return problems
}
```

`internal/vault/directive_test.go`:

```go
package vault

import (
	"strings"
	"testing"
)

func parseDirectiveProblems(t *testing.T, document string) []string {
	t.Helper()
	parsed, err := parsePage([]byte(document))
	if err != nil {
		t.Fatal(err)
	}
	return parseDirective(parsed.fields, parsed.body)
}

func TestParseDirectiveAcceptsMinimalValidDirective(t *testing.T) {
	problems := parseDirectiveProblems(t, `---
type: Directive
title: Valid
status: open
---

# Goal

Ship it.

# Scope

- This.

# Acceptance criteria

- It works — run `make test`; expect ok.
`)
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
}

func TestParseDirectiveRejectsMissingOrInvalidStatus(t *testing.T) {
	for _, status := range []string{"", "pending", "Open"} {
		statusLine := ""
		if status != "" {
			statusLine = "status: " + status + "\n"
		}
		document := "---\ntype: Directive\ntitle: Bad\n" + statusLine + "---\n\n# Goal\n\nG.\n\n# Scope\n\nS.\n\n# Acceptance criteria\n\nA.\n"
		problems := parseDirectiveProblems(t, document)
		if len(problems) == 0 || !strings.Contains(problems[0], "status") {
			t.Fatalf("status %q: problems = %v", status, problems)
		}
	}
}

func TestParseDirectiveRejectsMissingRequiredSections(t *testing.T) {
	problems := parseDirectiveProblems(t, `---
type: Directive
title: Sparse
status: draft
---

# Goal

G.
`)
	joined := strings.Join(problems, "; ")
	for _, want := range []string{"Scope", "Acceptance criteria"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems = %v, want missing %q", problems, want)
		}
	}
}

func TestParseDirectiveChecksPlanTasks(t *testing.T) {
	problems := parseDirectiveProblems(t, `---
type: Directive
title: Plan
status: draft
---

# Goal

G.

# Scope

S.

# Implementation plan

### Task 1: Do it

No checkboxes here.

### Wrong heading

- [ ] step

# Acceptance criteria

A.
`)
	joined := strings.Join(problems, "; ")
	for _, want := range []string{"requires at least one checkbox step", "invalid task heading"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems = %v, want %q", problems, want)
		}
	}
}

func TestParseDirectiveChecksDeltaSections(t *testing.T) {
	problems := parseDirectiveProblems(t, `---
type: Directive
title: Delta
status: draft
---

# Goal

G.

# Scope

S.

# Purpose/Decision Changes

## Changed

- something

## Added

# Acceptance criteria

A.
`)
	joined := strings.Join(problems, "; ")
	for _, want := range []string{"invalid delta section", "requires at least one entry"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems = %v, want %q", problems, want)
		}
	}
}

func TestParseDirectiveChecksScenarioGrammar(t *testing.T) {
	problems := parseDirectiveProblems(t, `---
type: Directive
title: Scenarios
status: draft
---

# Goal

G.

# Scope

S.

# Acceptance criteria

#### Scenario: missing then

- **WHEN** asked

#### Scenario:

- **WHEN** asked
- **THEN** answered
`)
	joined := strings.Join(problems, "; ")
	for _, want := range []string{"requires a **THEN** bullet", "invalid scenario heading"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems = %v, want %q", problems, want)
		}
	}
}

func TestDirectiveTaskProgressSkipsCodeFences(t *testing.T) {
	done, total := directiveTaskProgress("# Plan\n\n- [ ] one\n- [x] two\n\n```\n- [ ] not a task\n```\n")
	if done != 1 || total != 2 {
		t.Fatalf("done,total = %d,%d", done, total)
	}
}

func TestDirectivesListsSummariesWithProgress(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "directives/alpha.md", `---
type: Directive
title: Alpha
status: open
---

# Goal

G.

# Scope

S.

# Implementation plan

### Task 1: Work

- [x] done step
- [ ] open step

# Acceptance criteria

A.
`)
	summaries, err := Directives(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %+v", summaries)
	}
	summary := summaries[0]
	if summary.Status != "open" || summary.TasksDone != 1 || summary.TasksTotal != 2 {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.URI != "gnosis://test/directives/alpha.md" {
		t.Fatalf("uri = %q", summary.URI)
	}
}
```

- [x] Run `go test ./internal/vault/ -run 'TestParseDirective|TestDirectives|TestDirectiveTaskProgress' -v`; expect every test ok.
- [x] Refactor: confirm no duplication with `procedure.go` helpers (reuse, do not copy); run `gofmt -l internal/vault`; expect no output.
- [x] Commit: `feat: add directive contract parsing and progress derivation`.

### Task 2: Validation dispatch for Directive records

**Load:** `internal/vault/validate.go:159-167`, `internal/vault/validate_test.go`.
**Files:** modify `internal/vault/validate.go`, modify `internal/vault/validate_test.go`.
**Interfaces:** consumes `parseDirective`; validator reports one error per contract problem.

- [x] Red: append the two tests below to `internal/vault/validate_test.go`; run `go test ./internal/vault/ -run TestValidateDirective`; expect the invalid-fixture test to fail (no errors reported yet).
- [x] Green: in `internal/vault/validate.go`, inside `validateFile` after the `isProcedureType` branch (current line 164-166), add:

```go
			if conceptType == DirectiveType {
				for _, problem := range parseDirective(fields, body) {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", path, problem))
				}
			}
```

- [x] Run `go test ./internal/vault/ -run TestValidate`; expect ok. Then `go test ./internal/vault/`; expect ok (existing directive fixtures elsewhere in tests must still pass — fix fixture, not code, if a pre-existing test fixture violates the contract).

Tests to append to `internal/vault/validate_test.go`:

```go
func TestValidateDirectiveRejectsContractViolations(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "directives/bad.md", `---
type: Directive
title: Bad
status: pending
---

# Goal

G.
`)
	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.Errors, "; ")
	for _, want := range []string{"status", "Scope", "Acceptance criteria"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("errors = %v, want %q", result.Errors, want)
		}
	}
}

func TestValidateDirectiveAcceptsValidDirective(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "directives/good.md", `---
type: Directive
title: Good
description: A valid directive.
status: draft
---

# Goal

G.

# Scope

S.

# Acceptance criteria

- It works — run `make test`; expect ok.
`)
	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %v", result.Errors)
	}
}
```

- [x] Run `mise run build`, then `./dist/gnosis validate vault` in the repository; expect `status: valid` — every record under `docs/directives/` already satisfies the contract.
- [x] Commit: `feat: validate directive contract`.

### Task 3: `gnosis get directives` command

**Load:** `cmd/gnosis/get.go`, `cmd/gnosis/get_test.go` (`commandVault`, `writeCommandFile`, `run` helpers).
**Files:** modify `cmd/gnosis/get.go`, modify `cmd/gnosis/get_test.go`.
**Interfaces:** consumes `vault.Directives`; produces TOON list output.

- [x] Red: append the test below to `cmd/gnosis/get_test.go`; run `go test ./cmd/gnosis/ -run TestGetDirectives`; expect "unknown command" failure.
- [x] Green: in `cmd/gnosis/get.go`, add `newGetDirectivesCommand(options, stdout),` to the `command.AddCommand(...)` call in `newGetCommand` (current lines 28-33), and append this function:

```go
func newGetDirectivesCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "directives [flags]",
		Short: "List directives with derived task progress",
		Args:  cobra.NoArgs,
		Example: "gnosis get directives\n" +
			"gnosis get directives --fields uri,status,tasks_done,tasks_total",
		RunE: func(_ *cobra.Command, _ []string) error {
			selector, err := parseFields(
				fields,
				[]string{"uri", "title", "status", "tasks_done", "tasks_total"},
				[]string{"uri", "title", "status", "tasks_done", "tasks_total", "revision"},
			)
			if err != nil {
				return newUsageError(err)
			}
			directives, err := vault.Directives(options.vaultPath)
			if err != nil {
				return fmt.Errorf("get directives: %w", err)
			}
			rows := make([]toon.Object, 0, len(directives))
			for _, directive := range directives {
				rows = append(rows, selector.object(func(name string) (any, bool) {
					switch name {
					case "uri":
						return directive.URI, true
					case "title":
						return directive.Title, true
					case "status":
						return directive.Status, true
					case "tasks_done":
						return directive.TasksDone, true
					case "tasks_total":
						return directive.TasksTotal, true
					case "revision":
						return directive.Revision, true
					default:
						return nil, false
					}
				}))
			}
			return writeTOON(stdout, listOutput(
				"directives",
				len(rows),
				rows,
				"0 directives found in the current vault",
				[]string{"Run `gnosis get pages <uri> --full` to read one directive"},
			))
		},
	}
	command.Flags().StringVar(
		&fields,
		"fields",
		"",
		"comma-separated fields: uri, title, status, tasks_done, tasks_total, revision",
	)
	return command
}
```

Test to append to `cmd/gnosis/get_test.go`:

```go
func TestGetDirectivesListsStatusAndDerivedProgress(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "directives/alpha.md", `---
type: Directive
title: Alpha
description: First.
status: open
---

# Goal

G.

# Scope

S.

# Implementation plan

### Task 1: Work

- [x] done step
- [ ] open step
- [ ] another open step

# Acceptance criteria

A.
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "directives"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"directives[1]{uri,title,status,tasks_done,tasks_total}",
		"Alpha",
		"open,1,3",
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}
}
```

(`commandVault` sets `vault_root = "."`, so the fixture lands at `directives/alpha.md` relative to the workspace root.)

- [x] Run `go test ./cmd/gnosis/ -run TestGetDirectives`; expect ok. Run `go test ./cmd/gnosis/`; expect ok.
- [x] Run `mise run build` (the PATH `gnosis` predates this command), then `./dist/gnosis get directives` in the repository; expect one row per record in `docs/directives/` with correct `status` and progress.
- [x] Commit: `feat: add get directives command`.

### Task 4: Update the intent records to the new contract

**Load:** `docs/concepts/directive.md`, `docs/procedures/development/maintain-intents.md`, `docs/procedures/development/planning-directives.md`, `docs/procedures/development/implementing-directive.md`, `docs/decisions/adopt-openspec-intent-strategies.md`.
**Files:** modify `docs/concepts/directive.md`, `docs/procedures/development/maintain-intents.md`, `docs/procedures/development/planning-directives.md`, `docs/procedures/development/implementing-directive.md`.
**Interfaces:** produces the updated records, applied with `gnosis apply page` and read back.

- [x] Replace `docs/concepts/directive.md` with exactly this content, then apply: `gnosis apply page gnosis://local/concepts/directive.md --filename docs/concepts/directive.md`:

````markdown
---
type: ConceptType
title: Directive
description: An explicitly requested durable implementation handoff.
path: directives
---

# Directive

A **Directive** is a bounded handoff for later automated or unattended execution.

`draft` is planning-only; finalization alone changes it to executable `open`.

By convention, the Directive records lives at `gnosis://<vault>/directives/`.

## Use this for

- Explicitly requested work that needs durable scope and observable acceptance criteria.

Do not create one implicitly for ordinary implementation, task tracking, or completed work.

## Minimum record

- `status`, `# Goal`, `# Scope`, and evidence-bearing `# Acceptance criteria`.
- Multi-step work adds `# Implementation plan` with `### Task N:` sections; every task carries checkbox steps (`- [ ]`) so progress is derived, never restated. Prerequisites add `# Dependencies`. Directive dependencies bind links, revisions, and supplied contracts.
- Behavior acceptance criteria use `#### Scenario: <name>` blocks with bold `**WHEN**` and `**THEN**` bullets (optional `**GIVEN**`/`**AND**`); when any scenario is present, every scenario must follow the grammar.
- When the work changes Purpose or Decision records, add `# Purpose/Decision Changes` declaring the deltas as `## Added`, `## Modified`, or `## Removed` subsections naming the exact target records; `## Modified` carries the full replacement text or exact section edits.
- Complex work adds only execution-relevant architecture, stack, and global constraints; name affected components and justify new libraries.
- Each task names exact files, interfaces, required paths/process URIs, atomic steps with complete code or patches, commands, expected results, and a commit.
- Behavior tasks use red-green-refactor plus focused and surrounding green; other tasks use exact validation.

Omit empty optional sections. Plans contain no placeholders.

## Lifecycle

- Require an explicitly requested durable implementation handoff. Creation invokes [planning-directives](../procedures/development/planning-directives.md), which owns drafting, review, persistence, and the `draft` to `open` transition.
- Apply only non-semantic corrections in place while preserving unknown metadata and status. A change to the goal, scope, dependencies, implementation plan, acceptance criteria, or declared deltas returns an unfinished Directive to `draft` and invokes `planning-directives` with its current URI, revision, original requirements, and proposed change.
- Status follows `draft` → `open` → `blocked|done`, with `blocked` → `draft` only after evidence shows the blocker is resolved. Never assign status from assertion alone: planning finalization owns `draft` → `open`, and [implementing-directive](../procedures/development/implementing-directive.md) owns evidence-backed `open` → `blocked|done` for exactly one directive per invocation. Checkbox progress is derived state and never a status. Replanning owns `blocked` → `draft` and must revalidate the requirements, dependencies, and plan before finalization reopens it. Reject every other transition.
- A completed Directive remains historical until [maintain-intents](../procedures/development/maintain-intents.md) archives it: its declared deltas merge into the living Purpose and Decision records, its still-durable choices are compacted into Decisions, and the Directive is removed. Create a new Directive for new or changed work instead of reopening it.
- Prefer correction or retention after a Directive has governed work. Delete only a confirmed local duplicate or invalid `draft` after tracing inbound links and dependency history, obtaining explicit approval for the exact deletion, and repairing or intentionally removing every inbound reference. Report imported or bundled records to their owning vault.

## Schema

```markdown
---
type: Directive
title: <name>
description: <result>
status: <draft | open | blocked | done>
---

# Goal
# Architecture
# Tech stack
# Global constraints
# Scope
# Dependencies

- <dependency link> @ <revision> — <required contract and evidence>

# Purpose/Decision Changes

## Added
## Modified
## Removed

# Implementation plan

### Task N: <deliverable>
**Load:** <exact paths/sections and process URIs>
**Files:** <create, modify, test: exact paths>
**Interfaces:** <consumes and produces: exact signatures>

- [ ] <one 2–5 minute action with complete code or patch>
- [ ] Run `<command>`; expect `<result>`.
- [ ] Commit: `<message>`.

# Acceptance criteria

- <observable outcome> — run/inspect <exact check>; expect <evidence>.

#### Scenario: <name>

- **WHEN** <trigger>
- **THEN** <observable result>
```
````

- [x] Replace `docs/procedures/development/maintain-intents.md` with exactly this content, then apply: `gnosis apply page gnosis://local/procedures/development/maintain-intents.md --filename docs/procedures/development/maintain-intents.md`:

````markdown
---
type: Procedure
title: maintain-intents
description: Use only when the author explicitly says `maintain-intents` or `maintain intents`; never select it for implicit intent maintenance.
tags: [gnosis, development]
---

# maintain-intents

`maintain-intents` archives completed handoffs by merging their declared deltas into living intent records and compacting their durable choices into Decisions before removing them.

## Inputs

- The resolved vault, repository instructions, and vault configuration.
- The effective Directive and Decision Concept Type definitions.
- Every effective Directive whose status is `done`, its declared `# Purpose/Decision Changes` deltas, all existing Decisions, their provenance, and inbound links.

## Process

1. Read the effective Directive and Decision Concept Type definitions, list their records, and read every Directive whose effective status is `done` plus every Decision that may overlap its choices.
2. Archive declared deltas first. For every `done` Directive with a `# Purpose/Decision Changes` section, apply each delta to the living records through [managing-intents](managing-intents.md): create every `## Added` record, apply every `## Modified` replacement in full, and retire, supersede, or remove every `## Removed` record according to its Concept Type lifecycle. Persist and read back each changed record. Surface any delta that conflicts with the current record state to the author instead of choosing silently.
3. Extract only durable, non-obvious choices that still constrain future work. Exclude status, routine implementation details, transient instructions, duplicated rationale, choices already captured by the merged deltas, and facts recoverable from the current implementation or version history. Bind each retained choice to the completed Directive that evidences it.
4. Cluster extracted choices by the settled choice and constraint they preserve, not by topic alone. Merge each cluster with any matching Decision identity. Keep only the current choice, essential rationale, and constraints; drop repetition and superseded or no-longer-relevant detail. Surface unresolved contradictions to the author instead of choosing silently.
5. Build the smallest complete Decision set allowed by the effective Decision lifecycle. Reject duplicate identities, obtain any required author confirmation, preserve unknown metadata, and use correction or supersession rather than rewriting decision history. Persist and read back every created or corrected Decision before deleting a source Directive.
6. Trace inbound links to each local completed Directive and repair or intentionally remove them. After all of its declared deltas are merged and its retained decisions are durable, delete the Directive's exact local origin file. Do not mutate imported or bundled origins; report them to their owning vault. Leave unfinished Directives unchanged.
7. When `vault_index` is enabled, run `gnosis index vault --vault <root>`. Run `gnosis validate vault --vault <root>` after all writes and deletions.

## Completion

Every effective `done` Directive was inspected; every declared delta is merged into the living records or reported; each still-relevant durable choice appears once in the smallest lifecycle-compliant Decision set; retained Decisions contain only the essential current choice, rationale, and constraints; every deletable local `done` Directive and its inbound links are removed; non-local completed Directives are reported; unfinished Directives are unchanged; and vault validation passes.
````

- [x] In `docs/procedures/development/planning-directives.md`, in `## STEP 2 - creating-directives` → `### Inputs`, replace the line `- Required directive contract: \`draft\` status, Goal, Scope, evidence-bearing Acceptance criteria, and an Implementation plan for multi-step work.` with `- Required directive contract: \`draft\` status, Goal, Scope, evidence-bearing Acceptance criteria (\`#### Scenario:\` blocks with \`**WHEN**\`/\`**THEN**\` bullets for behavior), an Implementation plan of \`### Task N:\` sections with checkbox steps for multi-step work, and \`## Added\`/\`## Modified\`/\`## Removed\` deltas under Purpose/Decision Changes whenever the work changes Purpose or Decision records.`
- [x] In the same file, in `## STEP 4 - finalizing-directives` → `### Inputs`, replace `- Required directive contract: status, Goal, Scope, evidence-bearing Acceptance criteria, an Implementation plan for multi-step work, and linked revision-bound contracts for prerequisites.` with `- Required directive contract: status, Goal, Scope, evidence-bearing Acceptance criteria (scenario blocks for behavior), an Implementation plan with checkbox tasks for multi-step work, Added/Modified/Removed deltas for Purpose/Decision changes, and linked revision-bound contracts for prerequisites.`
- [x] In `docs/procedures/development/implementing-directive.md`, in `## STEP 3 - implementing-tasks` → `### Process` item 1, append after `...governing procedure URIs.` the sentence: `When a task completes, mark its checkbox steps \`- [x]\` in the local directive file and persist with \`gnosis apply page '<directive URI>' --filename <directive-file>\` before starting the next task; derived progress must always match completed work.`
- [x] In the same file, in `## STEP 5 - verifying-directive` → `### Process` item 1, append after `...against the current state.` the sentence: `Map each \`#### Scenario:\` block to its **WHEN** setup and **THEN** observation and prove each one.`
- [x] Apply the two edited procedures: `gnosis apply page gnosis://local/procedures/development/planning-directives.md --filename docs/procedures/development/planning-directives.md` and `gnosis apply page gnosis://local/procedures/development/implementing-directive.md --filename docs/procedures/development/implementing-directive.md`.
- [x] Read back all four records with `gnosis get pages '<URI>' --full`; expect the new text present. Run `gnosis validate vault`; expect `status: valid`.
- [x] Commit: `feat: adopt openspec strategies in intent records`.

# Acceptance criteria

- Contract violations are validation errors — create a temp vault with a directive missing `# Scope` and with `status: pending`; run `./dist/gnosis --vault <tmp> validate vault` (after `mise run build`); expect exit 1 naming both problems; run `go test ./internal/vault/ -run TestValidateDirective`; expect ok.
- Progress is derived — run `mise run build`, then `./dist/gnosis get directives` in this repository; expect every record in `docs/directives/` listed with its real `status` and `tasks_done`/`tasks_total` matching its checkboxes.
- The intent records carry the new contract — run `gnosis get pages gnosis://local/concepts/directive.md --full`; expect the delta and scenario rules; run `gnosis get procedures gnosis://local/procedures/development/maintain-intents.md --full`; expect the archive-first process.
- Vault integrity holds — run `gnosis validate vault`; expect `status: valid`, `warnings: 0`.
- No regressions — run `mise run checks`; expect gofmt, vet, tests with race detector, build, and vault validation all green.
