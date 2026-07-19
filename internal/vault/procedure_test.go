package vault

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadAndInvokeProcedureRoundTrip(t *testing.T) {
	root := apiTestVault(t)

	processURI := "gnosis://agent-test/processes/query-vault.md"
	invocation, err := InvokeProcess(root, processURI)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Process.URI != processURI || invocation.Process.Type != "Procedure" {
		t.Fatalf("invoked process = %+v", invocation.Process)
	}
	if invocation.Process.Origin.Kind != OriginLocal || invocation.Process.Origin.Vault != "agent-test" || invocation.Process.Revision == "" {
		t.Fatalf("process origin = %+v", invocation.Process)
	}
	if invocation.Process.Invocation != "model" || strings.Join(invocation.Process.Tags, ",") != "test-vault" {
		t.Fatalf("process metadata = %+v", invocation.Process)
	}
	if !strings.Contains(invocation.Sections.Process, "Read only the selected pages") {
		t.Fatalf("process section = %q", invocation.Sections.Process)
	}
	if !strings.Contains(invocation.Sections.Completion, "grounded answer") {
		t.Fatalf("completion section = %q", invocation.Sections.Completion)
	}
	encoded, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["sections"]; !ok {
		t.Fatalf("invocation = %s, want sections", encoded)
	}
	if _, ok := fields["steps"]; ok {
		t.Fatalf("invocation = %s, want no steps", encoded)
	}

	page, err := ReadPage(root, processURI)
	if err != nil {
		t.Fatal(err)
	}
	if page.Document.URI != invocation.Process.URI || !strings.Contains(page.Markdown, "# query-vault") {
		t.Fatalf("page = %+v", page)
	}
	if _, err := ReadPage(root, "processes/query-vault.md"); err == nil {
		t.Fatal("ReadPage accepted a relative path")
	}
}

func TestInvokeMultiStepProcess(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/planning.md", `---
type: Procedure
title: planning
description: Plan a delivery in ordered steps.
tags: [test-planning]
---

# planning

## STEP 1 - refining-requirements

### Inputs

- The request.

### Process

1. Refine the requirements.

### Completion

The requirements are exact.

## STEP 2 - creating-records

### Inputs

- The exact requirements.

### Process

1. Create the record.

### Completion

The record is ready.
`)

	invocation, err := InvokeProcess(root, "gnosis://agent-test/processes/planning.md")
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Sections != (ProcessSections{}) {
		t.Fatalf("sections = %+v, want empty", invocation.Sections)
	}
	if len(invocation.Steps) != 2 {
		t.Fatalf("steps = %+v", invocation.Steps)
	}
	first, second := invocation.Steps[0], invocation.Steps[1]
	if first.Number != 1 || first.Name != "refining-requirements" || !strings.Contains(first.Sections.Process, "Refine the requirements") {
		t.Fatalf("first step = %+v", first)
	}
	if second.Number != 2 || second.Name != "creating-records" || !strings.Contains(second.Sections.Completion, "record is ready") {
		t.Fatalf("second step = %+v", second)
	}
	encoded, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["steps"]; !ok {
		t.Fatalf("invocation = %s, want steps", encoded)
	}
	if _, ok := fields["sections"]; ok {
		t.Fatalf("invocation = %s, want no top-level sections", encoded)
	}
	validation, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(validation.Errors) != 0 {
		t.Fatalf("validation errors = %v", validation.Errors)
	}
}

func TestDiscoverProcessesFiltersByAllTags(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/planning.md", `---
type: Procedure
title: planning
description: A planning-only process.
tags: [test-vault, test-planning]
---

# planning

## Inputs

- Requirements.

## Process

1. Plan.

## Completion

The plan is complete.
`)

	discovery, err := DiscoverProcesses(root, []string{"test-vault", "test-planning"})
	if err != nil {
		t.Fatal(err)
	}
	procedures := discovery["procedures"]
	if len(procedures) != 1 || procedures[0]["uri"] != "gnosis://agent-test/processes/planning.md" {
		t.Fatalf("procedures = %+v", procedures)
	}
}

func TestDiscoverProcessesIncludesAllModelInvocableProceduresByDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	discovery, err := DiscoverProcesses(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	procedures := discovery["procedures"]
	foundQuery := false
	foundRefining := false
	for _, procedure := range procedures {
		if procedure["uri"] == "gnosis://core/procedures/query-vault.md" {
			foundQuery = true
		}
		if procedure["uri"] == "gnosis://core/procedures/refining-procedure.md" {
			foundRefining = true
		}
	}
	if !foundQuery || !foundRefining {
		t.Fatalf("default discovery omitted model-invocable procedures: %+v", procedures)
	}
}

func TestDiscoverProcessesIncludesAllFamiliesAndOmitsExplicit(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/enabled.md", `---
type: Procedure
title: enabled
description: A model-invocable procedure in an enabled family.
tags: [enabled-family]
---

# enabled

## Inputs

- Facts.

## Process

1. Work.

## Completion

The work is complete.
`)
	write(t, root, "docs/processes/disabled.md", `---
type: Procedure
title: disabled
description: A procedure in another family.
tags: [disabled-family]
---

# disabled

## Inputs

- Facts.

## Process

1. Work.

## Completion

The work is complete.
`)
	write(t, root, "docs/processes/explicit.md", `---
type: Procedure
title: explicit
description: An explicit procedure in an enabled family.
tags: [enabled-family]
invocation: explicit
---

# explicit

## Inputs

- Facts.

## Process

1. Work.

## Completion

The work is complete.
`)
	write(t, root, "docs/processes/explicit-invalid.md", `---
type: Procedure
title: explicit-invalid
description: A hidden explicit procedure with an invalid body.
tags: [enabled-family]
invocation: explicit
---

# explicit-invalid

[Malformed destination](bad%ZZ.md)
`)
	write(t, root, "docs/processes/explicit-invalid-metadata.md", `---
type: Procedure
title: explicit-invalid-metadata
description: [invalid]
tags: [enabled-family]
invocation: explicit
aliases:
  nested: invalid
---

# explicit-invalid-metadata
`)
	write(t, root, "docs/unrelated-invalid.md", `---
type: Note
title: Unrelated invalid page
aliases:
  nested: invalid
---
`)

	discovery, err := DiscoverProcesses(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	procedures := discovery["procedures"]
	foundEnabled, foundDisabled, foundExplicit := false, false, false
	for _, procedure := range procedures {
		switch procedure["uri"] {
		case "gnosis://agent-test/processes/enabled.md":
			foundEnabled = true
		case "gnosis://agent-test/processes/disabled.md":
			foundDisabled = true
		case "gnosis://agent-test/processes/explicit.md":
			foundExplicit = true
		}
	}
	if !foundEnabled || !foundDisabled || foundExplicit {
		t.Fatalf("procedures = %+v", procedures)
	}

	invocation, err := InvokeProcess(root, "gnosis://agent-test/processes/explicit.md")
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Process.Invocation != "explicit" {
		t.Fatalf("invocation = %+v", invocation)
	}
}

func TestDiscoverProcessesRejectsMalformedEnabledProcedure(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/malformed.md", `---
type: Procedure
title: malformed
description: A malformed enabled procedure.
tags: [enabled-family]
aliases:
  nested: invalid
---

# malformed
`)

	_, err := DiscoverProcesses(root, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required section") {
		t.Fatalf("discovery error = %v", err)
	}
}

func TestProcedureValidationAndInvocationShareContract(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/missing-tags.md", `---
type: Procedure
title: missing-tags
description: A procedure missing its process family.
---

# missing-tags

## Inputs

- Facts.

## Process

1. Work.

## Completion

The work is complete.
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	want := `process requires non-empty "tags" frontmatter`
	if !strings.Contains(strings.Join(result.Errors, "\n"), want) {
		t.Fatalf("validation errors = %v, want %q", result.Errors, want)
	}
	if _, err := InvokeProcess(root, "gnosis://agent-test/processes/missing-tags.md"); err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("InvokeProcess error = %v, want %q", err, want)
	}
}

func TestInvokeProcessReportsSelectedFrontmatterError(t *testing.T) {
	root := apiTestVault(t)
	write(t, root, "docs/processes/broken.md", "---\ntype: [\n---\n")

	_, err := InvokeProcess(root, "gnosis://agent-test/processes/broken.md")
	if err == nil || !strings.Contains(err.Error(), "invalid YAML frontmatter") {
		t.Fatalf("invocation error = %v", err)
	}
}

func TestValidateRequiresProcessShapeAndMetadata(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "invalid-process"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "process.md", `---
type: Procedure
title: invalid
description: Invalid process.
invocation: surprise
---

# invalid

## Process

1. Run.
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.Errors, "\n")
	for _, want := range []string{"missing required section \"Inputs\"", "missing required section \"Completion\"", "invocation"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("errors = %v, want %q", result.Errors, want)
		}
	}
}

func TestValidateRequiresProcessSelectionMetadata(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "missing-selection-metadata"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "process.md", `---
type: Procedure
title: vague-process
---

# vague-process

## Inputs

- Current facts.

## Process

1. Work.

## Completion

The work is complete.
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.Errors, "\n")
	for _, want := range []string{"description"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("errors = %v, want %q", result.Errors, want)
		}
	}
}

func TestValidateRejectsInvalidMultiStepProcess(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "invalid-multi-step"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "process.md", `---
type: Procedure
title: invalid-multi-step
description: An invalid multi-step process.
---

# invalid-multi-step

## STEP 1 - first

### Inputs

- Facts.

### Process

1. Work.

### Completion

The first step is complete.

## STEP 3 - second

### Inputs

- More facts.

### Process

1. Work again.
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(result.Errors, "\n")
	for _, want := range []string{"expected STEP 2", `STEP 3 - second missing required section "Completion"`} {
		if !strings.Contains(joined, want) {
			t.Fatalf("errors = %v, want %q", result.Errors, want)
		}
	}
}

func TestValidateRejectsUnresolvedTypedRelationship(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "broken-relationship"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "source.md", `---
type: Note
title: Source
description: A typed relationship source.
relationships:
  - type: uses
    target: missing.md
---

# Source
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "unresolved relationships[0] target missing.md") {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func apiTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "agent-test"
vault_root = "docs"
vault_index = false
vault_log = false
`)
	write(t, root, "docs/concepts/provenance.md", `---
type: Concept
title: Provenance
description: Source identity and history.
---

# Provenance
`)
	write(t, root, "docs/processes/query-vault.md", `---
type: Procedure
title: query-vault
description: Use when answering a question from recorded vault knowledge.
tags: [test-vault]
invocation: model
---

# query-vault

## Inputs

- [Provenance](../concepts/provenance.md)

## Process

1. Read only the selected pages.

## Completion

The grounded answer is complete.

## STEP notes

Optional legacy section.
`)
	return root
}
