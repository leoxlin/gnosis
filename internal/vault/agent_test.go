package vault

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestReadAndInvokeProcessRoundTrip(t *testing.T) {
	root := agentTestVault(t)

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
	if invocation.Process.Invocation != "model" || strings.Join(invocation.Process.Tags, ",") != "test-vault" || len(invocation.Process.UseWhen) != 1 || invocation.Process.UseWhen[0] != "Answering a question from a vault." {
		t.Fatalf("process metadata = %+v", invocation.Process)
	}
	if !strings.Contains(invocation.Sections.Process, "Read only the selected pages") {
		t.Fatalf("process section = %q", invocation.Sections.Process)
	}
	if !strings.Contains(invocation.Sections.Completion, "grounded answer") {
		t.Fatalf("completion section = %q", invocation.Sections.Completion)
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

func TestDiscoverProcessesUsesTypedConceptCatalog(t *testing.T) {
	root := agentTestVault(t)
	writeConfig(t, root, `[vault]
vault_name = "agent-test"
vault_root = "docs"
vault_index = false
vault_log = false

[gnosis]
processes = ["test-vault"]
`)
	write(t, root, "docs/processes/planning.md", `---
type: Procedure
title: planning
description: A planning-only process.
tags: [test-planning]
use_when:
  - Planning.
---

# planning

## Knowledge inputs

- Requirements.

## Process

1. Plan.

## Completion

The plan is complete.
`)

	discovery, err := DiscoverProcesses(root)
	if err != nil {
		t.Fatal(err)
	}
	procedures := discovery["procedures"]
	var queryVault ConceptRecord
	for _, procedure := range procedures {
		if procedure.URI == "gnosis://agent-test/processes/query-vault.md" {
			queryVault = procedure
		}
	}
	if queryVault.URI == "" {
		t.Fatalf("procedures missing local query-vault: %+v", procedures)
	}
	tags, ok := queryVault.Fields["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "test-vault" {
		t.Fatalf("tags = %#v", queryVault.Fields["tags"])
	}
}

func TestReadPageAcceptsOnlyCanonicalGnosisURIs(t *testing.T) {
	root := agentTestVault(t)
	canonical := "gnosis://agent-test/processes/query-vault.md"

	page, err := ReadPage(root, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if page.Document.URI != canonical {
		t.Fatalf("URI = %q, want %q", page.Document.URI, canonical)
	}

	if _, err := ReadPage(root, "gnosis://vault/agent-test/processes/query-vault.md"); err == nil {
		t.Fatal("ReadPage accepted the retired vault-path URI")
	}
}

func TestTraceLinksReturnsDirectedTypedEdges(t *testing.T) {
	root := agentTestVault(t)

	neighbors, err := TraceNeighbors(root, "gnosis://agent-test/processes/query-vault.md", DirectionOut, []string{"links_to"})
	if err != nil {
		t.Fatal(err)
	}
	if len(neighbors.Edges) != 1 {
		t.Fatalf("edges = %+v", neighbors.Edges)
	}
	edge := neighbors.Edges[0]
	if edge.From.URI != "gnosis://agent-test/processes/query-vault.md" || edge.To.URI != "gnosis://agent-test/concepts/provenance.md" || edge.Relation != "links_to" {
		t.Fatalf("edge = %+v", edge)
	}

	path, err := TracePath(root, "gnosis://agent-test/processes/query-vault.md", "gnosis://agent-test/concepts/provenance.md", DirectionOut, []string{"links_to"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if path.Status != PathFound || len(path.Edges) != 1 || len(path.Nodes) != 2 {
		t.Fatalf("path = %+v", path)
	}

	reverse, err := TracePath(root, "gnosis://agent-test/concepts/provenance.md", "gnosis://agent-test/processes/query-vault.md", DirectionOut, []string{"links_to"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if reverse.Status != PathDisconnected {
		t.Fatalf("reverse path = %+v", reverse)
	}

	incoming, err := TracePath(root, "gnosis://agent-test/concepts/provenance.md", "gnosis://agent-test/processes/query-vault.md", DirectionIn, []string{"links_to"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if incoming.Status != PathFound {
		t.Fatalf("incoming path = %+v", incoming)
	}
}

func TestTraceLinksResolvesLogicalImportedTargets(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "docs"
vault_index = false
vault_log = false

[[vaults]]
vault_name = "shared"
vault_root = "imported/docs"
`)
	writeConfig(t, imported, `[vault]
vault_name = "shared"
vault_root = "docs"
vault_index = false
vault_log = false
`)
	write(t, workspace, "docs/processes/start.md", `---
type: Procedure
title: start
description: Start with shared knowledge.
use_when:
  - Starting.
---

# start

## Knowledge inputs

- [Shared](../shared/end.md)

## Process

1. Read shared knowledge.

## Completion

Shared knowledge is read.
`)
	write(t, imported, "docs/shared/end.md", `---
type: Concept
title: Shared End
description: Imported shared knowledge.
---

# Shared End
`)

	path, err := TracePath(workspace, "gnosis://workspace/processes/start.md", "gnosis://shared/shared/end.md", DirectionOut, []string{"links_to"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if path.Status != PathFound || len(path.Nodes) != 2 {
		t.Fatalf("path = %+v", path)
	}
	if path.Nodes[1].Origin.Kind != OriginImport || path.Nodes[1].Origin.Vault != "shared" {
		t.Fatalf("imported origin = %+v", path.Nodes[1].Origin)
	}
	validation, err := Validate(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(validation.Errors) != 0 {
		t.Fatalf("validation errors = %v", validation.Errors)
	}
}

func TestTracePathDistinguishesUnknownDisconnectedAndDepthExceeded(t *testing.T) {
	root := agentTestVault(t)
	write(t, root, "docs/concepts/middle.md", `---
type: Concept
title: Middle
description: A middle node.
relationships:
  - type: links_to
    target: end.md
---

# Middle
`)
	write(t, root, "docs/concepts/end.md", `---
type: Concept
title: End
description: An end node.
---

# End
`)
	write(t, root, "docs/concepts/disconnected.md", `---
type: Concept
title: Disconnected
description: A disconnected node.
---

# Disconnected
`)
	write(t, root, "docs/concepts/provenance.md", `---
type: Concept
title: Provenance
description: Source identity and history.
relationships:
  - type: links_to
    target: middle.md
---

# Provenance
`)

	unknown, err := TracePath(root, "gnosis://agent-test/missing.md", "gnosis://agent-test/concepts/end.md", DirectionOut, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Status != PathUnknownSource {
		t.Fatalf("unknown path = %+v", unknown)
	}

	disconnected, err := TracePath(root, "gnosis://agent-test/concepts/provenance.md", "gnosis://agent-test/concepts/disconnected.md", DirectionOut, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if disconnected.Status != PathDisconnected {
		t.Fatalf("disconnected path = %+v", disconnected)
	}

	limited, err := TracePath(root, "gnosis://agent-test/concepts/provenance.md", "gnosis://agent-test/concepts/end.md", DirectionOut, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if limited.Status != PathDepthExceeded {
		t.Fatalf("limited path = %+v", limited)
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
use_when:
  - Testing invalid records.
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
	for _, want := range []string{"missing required section \"Knowledge inputs\"", "missing required section \"Completion\"", "invocation"} {
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

## Knowledge inputs

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
	for _, want := range []string{"description", "use_when"} {
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

func agentTestVault(t *testing.T) string {
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
use_when:
  - Answering a question from a vault.
---

# query-vault

## Knowledge inputs

- [Provenance](../concepts/provenance.md)

## Process

1. Read only the selected pages.

## Completion

The grounded answer is complete.
`)
	return root
}
