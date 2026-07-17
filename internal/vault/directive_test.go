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

- It works — run `+"`make test`"+`; expect ok.
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

func TestDirectiveTaskProgressSkipsNestedCodeFences(t *testing.T) {
	body := "# Plan\n\n- [ ] one\n\n````markdown\n```yaml\n- [ ] not a task\n```\n````\n\n- [x] two\n"
	done, total := directiveTaskProgress(body)
	if done != 1 || total != 2 {
		t.Fatalf("done,total = %d,%d", done, total)
	}
}

func TestParseDirectiveIgnoresRecordsEmbeddedInNestedFences(t *testing.T) {
	problems := parseDirectiveProblems(t, "---\ntype: Directive\ntitle: Nested\nstatus: draft\n---\n\n# Goal\n\nG.\n\n# Scope\n\nS.\n\n# Implementation plan\n\n### Task 1: Embed a record\n\n- [ ] Apply this file:\n\n````markdown\n# Goal\n\nEmbedded.\n\n# Acceptance criteria\n\n#### Scenario: embedded\n\n- **WHEN** x\n- **THEN** y\n````\n\n# Acceptance criteria\n\nA.\n")
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
}
