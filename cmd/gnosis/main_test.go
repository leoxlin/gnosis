package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "gnosis 0.1.0\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunHelpUsesStandardOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunSubcommandHelpIsSuccessful(t *testing.T) {
	for _, test := range []struct {
		args  []string
		usage string
	}{
		{args: []string{"validate", "--help"}, usage: "gnosis validate"},
		{args: []string{"process", "discover", "--help"}, usage: "gnosis process discover"},
		{args: []string{"graph", "path", "--help"}, usage: "gnosis graph path"},
		{args: []string{"mcp", "serve", "--help"}, usage: "gnosis mcp serve"},
	} {
		t.Run(strings.Join(test.args[:len(test.args)-1], "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if err := run(test.args, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), test.usage) {
				t.Fatalf("stdout = %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestRunRejectsUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{
		{"version", "extra"},
		{"validate", "extra"},
		{"setup", "extra"},
		{"read", "first", "second"},
		{"query", "search", "one", "two"},
		{"query", "graph", "one", "two"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			if err == nil || (!strings.Contains(err.Error(), "unexpected argument") && !strings.Contains(err.Error(), "gnosis URI")) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRunGraphQueryEmitsCompactAndPrettyJSON(t *testing.T) {
	root := queryTestVault(t)
	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "compact", args: []string{"query", "graph", "-vault", root, "transformer"}},
		{name: "pretty", args: []string{"query", "graph", "-vault", root, "-pretty", "transformer"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if err := run(test.args, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
			var result struct {
				AnswerType string `json:"answer_type"`
				Candidates []struct {
					Page        string `json:"page"`
					URI         string `json:"uri"`
					Type        string `json:"type"`
					Description string `json:"description"`
				} `json:"candidates"`
				ShouldRead []string `json:"should_read"`
				IndexOnly  bool     `json:"index_only"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
			}
			if result.AnswerType != "direct" || len(result.Candidates) == 0 {
				t.Fatalf("result = %+v", result)
			}
			if result.Candidates[0].Page != "transformer.md" {
				t.Fatalf("top candidate = %+v", result.Candidates[0])
			}
			if result.Candidates[0].URI == "" || result.Candidates[0].Type != "Concept" {
				t.Fatalf("candidate identity = %+v", result.Candidates[0])
			}
			if len(result.ShouldRead) > 3 {
				t.Fatalf("should_read = %v", result.ShouldRead)
			}
			if strings.Contains(stdout.String(), "Self-attention details live only in the body") {
				t.Fatalf("JSON leaked page body: %q", stdout.String())
			}
			if test.name == "pretty" && !strings.Contains(stdout.String(), "\n  \"answer_type\"") {
				t.Fatalf("stdout is not pretty JSON: %q", stdout.String())
			}
		})
	}
}

func TestRunQueryUsesCompactTextAndOptionalJSON(t *testing.T) {
	root := queryTestVault(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"query", "search", "-vault", root, "transformer"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "answer_type: direct") ||
		!strings.Contains(stdout.String(), "Transformer Architecture (transformer.md)") ||
		strings.Contains(stdout.String(), "Self-attention details live only in the body") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stdout.Reset()
	if err := run([]string{"query", "search", "-vault", root, "-pretty", "Transformer Architecture"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var result struct {
		IndexOnly  bool     `json:"index_only"`
		ShouldRead []string `json:"should_read"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.IndexOnly || len(result.ShouldRead) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRunQueryValidatesQuestionAndBounds(t *testing.T) {
	root := queryTestVault(t)
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"query", "search", "-vault", root}, want: "missing question"},
		{args: []string{"query", "search", "-vault", root, "-top", "0", "query"}, want: "-top"},
		{args: []string{"query", "search", "-vault", root, "-max-read", "-1", "query"}, want: "-max-read"},
		{args: []string{"query", "graph", "-vault", root, "-depth", "0", "query"}, want: "-depth"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		err := run(test.args, &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("args = %v error = %v, want %q", test.args, err, test.want)
		}
	}
}

func TestRunQueryIsReadOnly(t *testing.T) {
	root := queryTestVault(t)
	path := filepath.Join(root, "transformer.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"query", "graph", "-vault", root, "transformer"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("query changed a vault page")
	}
}

func TestRunReadPrintsExactDocument(t *testing.T) {
	root := queryTestVault(t)
	path := filepath.Join(root, "transformer.md")
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"read", "-vault", root, "-type", "Concept", "-title", "Transformer Architecture"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	wantRendered := strings.ReplaceAll(string(want), "[Attention](attention.md)", "[Attention](gnosis://Test/attention.md)")
	if stdout.String() != wantRendered {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantRendered)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReadRequiresUniqueTypeAndTitle(t *testing.T) {
	root := queryTestVault(t)
	writeTestFile(t, root, "duplicate.md", `---
type: Concept
title: Transformer Architecture
---

# Duplicate
`)

	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"read", "-vault", root, "-title", "Transformer Architecture"}, want: "-type"},
		{args: []string{"read", "-vault", root, "-type", "Concept"}, want: "-title"},
		{args: []string{"read", "-vault", root, "-type", "Missing", "-title", "Transformer Architecture"}, want: "no document found"},
		{args: []string{"read", "-vault", root, "-type", "Concept", "-title", "Transformer Architecture"}, want: "multiple documents found"},
	} {
		t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(test.args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRunReadByIDAsMarkdownAndJSON(t *testing.T) {
	root := processCommandTestVault(t)
	want, err := os.ReadFile(filepath.Join(root, "processes", "query-vault.md"))
	if err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"read", "--vault", root, "--id", "processes/query-vault.md"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	wantRendered := strings.ReplaceAll(string(want), "[Provenance](../concepts/provenance.md)", "[Provenance](gnosis://Process%20Test/concepts/provenance.md)")
	if stdout.String() != wantRendered {
		t.Fatalf("stdout = %q, want %q", stdout.String(), wantRendered)
	}

	stdout.Reset()
	if err := run([]string{"read", "--vault", root, "--id", "processes/query-vault.md", "--pretty"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var page struct {
		Document struct {
			ID  string `json:"id"`
			URI string `json:"uri"`
		} `json:"document"`
		Markdown string `json:"markdown"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Document.ID != "processes/query-vault.md" || page.Document.URI == "" || page.Markdown != wantRendered {
		t.Fatalf("page = %+v", page)
	}

	if err := run([]string{"read", "--vault", root, "--id", "processes/query-vault.md", "--type", "Gnosis Process", "--title", "query-vault"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("error = %v", err)
	}
}

func TestRunReadAcceptsCanonicalURIAndRendersDocumentLinks(t *testing.T) {
	root := queryTestVault(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"read", "-vault", root, "gnosis://Test/transformer.md"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "[Attention](gnosis://Test/attention.md)") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunReadRendersOnlyResolvedMarkdownDocumentLinks(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n")
	writeTestFile(t, root, "source.md", `---
type: Concept
title: Source
---

[Inline](target.md?view=full#section)
[Reference][target]
[External](https://example.com/page)
[Missing](missing.md)
`+"`[Code](target.md)`"+`

[target]: target.md
`)
	writeTestFile(t, root, "target.md", "---\ntype: Concept\ntitle: Target\n---\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"read", "-vault", root, "gnosis://Test/source.md"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	for _, want := range []string{
		"[Inline](gnosis://Test/target.md?view=full#section)",
		"[target]: gnosis://Test/target.md",
		"[External](https://example.com/page)",
		"[Missing](missing.md)",
		"`[Code](target.md)`",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q: %q", want, got)
		}
	}
}

func TestRunProcessDiscoverAndInvoke(t *testing.T) {
	root := processCommandTestVault(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"process", "discover", "--vault", root, "--type", "Gnosis Process", "--pretty", "answer from recorded knowledge"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var discovery struct {
		Processes []struct {
			ID         string   `json:"id"`
			URI        string   `json:"uri"`
			UseWhen    []string `json:"use_when"`
			Invocation string   `json:"invocation"`
			Effects    []string `json:"effects"`
		} `json:"processes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &discovery); err != nil {
		t.Fatal(err)
	}
	var selected struct {
		ID         string   `json:"id"`
		URI        string   `json:"uri"`
		UseWhen    []string `json:"use_when"`
		Invocation string   `json:"invocation"`
		Effects    []string `json:"effects"`
	}
	for _, process := range discovery.Processes {
		if process.ID == "processes/query-vault.md" {
			selected = process
			break
		}
	}
	if selected.URI == "" || selected.Invocation != "model" {
		t.Fatalf("discovery = %+v", discovery)
	}

	stdout.Reset()
	if err := run([]string{"process", "invoke", "--vault", root, "--id", selected.URI, "--pretty"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var invocation struct {
		Process struct {
			ID string `json:"id"`
		} `json:"process"`
		Sections struct {
			Completion string `json:"completion"`
		} `json:"sections"`
		Relationships []struct {
			Relation string `json:"relation"`
		} `json:"relationships"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &invocation); err != nil {
		t.Fatal(err)
	}
	if invocation.Process.ID != "processes/query-vault.md" || invocation.Sections.Completion == "" || len(invocation.Relationships) != 1 {
		t.Fatalf("invocation = %+v", invocation)
	}
}

func TestRunGraphNeighborsAndPath(t *testing.T) {
	root := processCommandTestVault(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"graph", "neighbors", "--vault", root, "--id", "processes/query-vault.md", "--direction", "out", "--relation", "uses", "--pretty"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var neighbors struct {
		Edges []struct {
			Relation string `json:"relation"`
			To       struct {
				ID string `json:"id"`
			} `json:"to"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &neighbors); err != nil {
		t.Fatal(err)
	}
	if len(neighbors.Edges) != 1 || neighbors.Edges[0].Relation != "uses" || neighbors.Edges[0].To.ID != "concepts/provenance.md" {
		t.Fatalf("neighbors = %+v", neighbors)
	}

	stdout.Reset()
	if err := run([]string{"graph", "path", "--vault", root, "--from", "processes/query-vault.md", "--to", "concepts/provenance.md", "--direction", "out", "--depth", "2", "--pretty"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var path struct {
		Status string `json:"status"`
		Nodes  []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &path); err != nil {
		t.Fatal(err)
	}
	if path.Status != "found" || len(path.Nodes) != 2 {
		t.Fatalf("path = %+v", path)
	}
}

func TestWriteCommandReadsStandardInput(t *testing.T) {
	root := writeCommandTestVault(t)
	changeWorkingDirectory(t, root)
	content := `---
type: Note
title: Standard Input
---

# Standard Input
`
	var stdout bytes.Buffer
	command := newWriteCommand(strings.NewReader(content), &stdout)
	command.SetArgs([]string{"--type", "Note", "--title", "Standard Input"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "notes", "standard-input.md")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("content = %q, want %q", got, content)
	}
	if stdout.String() != path+"\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunWriteReadsOptionalFilename(t *testing.T) {
	root := writeCommandTestVault(t)
	changeWorkingDirectory(t, root)
	content := `---
type: Note
title: File Input
---

# File Input
`
	inputPath := filepath.Join(root, "input.md")
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"write", "-type", "Note", "-title", "File Input", "input.md"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "file-input.md")); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunConceptsPreviewsConceptTypesAndConcepts(t *testing.T) {
	root := queryTestVault(t)
	writeTestFile(t, root, "concept-type.md", `---
type: Concept Type
title: Concept
description: A reusable knowledge record.
---

# Concept
`)
	writeTestFile(t, root, "pattern-type.md", `---
type: Concept Type
title: Pattern
---

# Pattern
`)
	writeTestFile(t, root, "pattern.md", `---
type: Pattern
title: Adapter Pattern
description: Decouples an interface from its implementation.
---

# Adapter Pattern
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"concepts", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Type: Concept\nDescription: A reusable knowledge record.\n") || !strings.Contains(stdout.String(), "Type: Pattern\nDescription: Pattern\n") || !strings.Contains(stdout.String(), "Type: Gnosis Process") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Type: Vault Process") || strings.Contains(stdout.String(), "Type: Repository Process") {
		t.Fatalf("stdout contains legacy process types = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Link: gnosis://Test/concept-type.md\n") {
		t.Fatalf("stdout missing concept-type link = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stdout.Reset()
	if err := run([]string{"concepts", "-vault", root, "-type", "Concept"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Title: Attention Mechanism\nDescription: Weighted token lookup.\nLink: gnosis://Test/attention.md\n") ||
		!strings.Contains(stdout.String(), "Title: Transformer Architecture\nDescription: Self-attention model.\n") ||
		strings.Contains(stdout.String(), "Path:") ||
		strings.Contains(stdout.String(), "Self-attention details live only in the body") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunConceptsEmitsMachineReadableCatalog(t *testing.T) {
	root := queryTestVault(t)
	writeTestFile(t, root, "concept-type.md", `---
type: Concept Type
title: Concept
description: A reusable knowledge record.
---

# Concept
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"concepts", "--vault", root, "--type", "Concept", "--pretty"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Type     string `json:"type"`
		Concepts []struct {
			ID       string `json:"id"`
			URI      string `json:"uri"`
			Revision string `json:"revision"`
		} `json:"concepts"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &catalog); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout.String(), err)
	}
	if catalog.Type != "Concept" || len(catalog.Concepts) != 2 {
		t.Fatalf("catalog = %+v", catalog)
	}
	for _, concept := range catalog.Concepts {
		if concept.ID == "" || concept.URI == "" || concept.Revision == "" {
			t.Fatalf("concept identity = %+v", concept)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunConceptsValidatesArgumentsAndType(t *testing.T) {
	root := queryTestVault(t)
	for _, test := range []struct {
		args []string
		want string
	}{
		{args: []string{"concepts", "extra"}, want: "unexpected argument"},
		{args: []string{"concepts", "-vault", root, "-type", "Missing"}, want: "no concepts with type"},
	} {
		t.Run(strings.Join(test.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(test.args, &stdout, &stderr)
			if strings.Contains(test.want, "no concepts") {
				if err != nil || !strings.Contains(stdout.String(), test.want) {
					t.Fatalf("stdout = %q error = %v", stdout.String(), err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func queryTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"")
	writeTestFile(t, root, "transformer.md", `---
type: Concept
title: Transformer Architecture
description: Self-attention model.
tags: [deep-learning]
---

# Transformer Architecture

Self-attention details live only in the body.
[Attention](attention.md)
`)
	writeTestFile(t, root, "attention.md", `---
type: Concept
title: Attention Mechanism
description: Weighted token lookup.
---

# Attention Mechanism
`)
	return root
}

func processCommandTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", `[vault]
vault_name = "Process Test"
vault_root = "."
vault_index = false
vault_log = false
`)
	writeTestFile(t, root, "concepts/provenance.md", `---
type: Concept
title: Provenance
description: Source identity and history.
---

# Provenance
`)
	writeTestFile(t, root, "processes/query-vault.md", `---
type: Gnosis Process
title: query-vault
description: Use when answering a question from recorded vault knowledge.
invocation: model
effects: [read]
relationships:
  - type: uses
    target: ../concepts/provenance.md
---

# query-vault

## Use when

- Answering a question from a vault.

## Knowledge inputs

- [Provenance](../concepts/provenance.md)

## Process

1. Read selected pages.

## Completion

The answer is grounded.
`)
	return root
}

func writeCommandTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"")
	writeTestFile(t, root, "concepts/note.md", "---\ntype: Concept Type\ntitle: Note\npath: notes\n---\n")
	return root
}

func changeWorkingDirectory(t *testing.T, path string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func TestRunValidateRoutesDiagnostics(t *testing.T) {
	t.Run("warnings", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n")
		writeTestFile(t, root, "index.md", "# Index\n\n[Log](log.md)\n")
		writeTestFile(t, root, "log.md", "# Log\n\n## 2026-07-09\n")
		writeTestFile(t, root, "note.md", "---\ntype: Note\n---\n\n# Note\n")
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		if err := run([]string{"validate", "-vault", root}, &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(stdout.String(), "ok: 3 markdown file(s) validated") {
			t.Fatalf("stdout = %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "warning:") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})

	t.Run("errors", func(t *testing.T) {
		root := t.TempDir()
		writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n")
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		err := run([]string{"validate", "-vault", root}, &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "validation failed") {
			t.Fatalf("error = %v", err)
		}
		if stdout.Len() != 0 {
			t.Fatalf("stdout = %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "error:") {
			t.Fatalf("stderr = %q", stderr.String())
		}
	})
}

func TestRunScaffoldAndIndex(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"scaffold", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "ok: vault scaffolded") || stderr.Len() != 0 {
		t.Fatalf("stdout = %q stderr = %q", stdout.String(), stderr.String())
	}

	writeTestFile(t, root, "concepts/new-note.md", "---\ntype: Note\ntitle: New Note\ndescription: Test.\n---\n\n# New Note\n")
	stdout.Reset()
	if err := run([]string{"index", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), filepath.Join(root, "concepts", "index.md")) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunScaffoldWithConceptsIndexesThem(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"scaffold", "-vault", root, "-concepts"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"concepts/gnosis-purpose.md",
		"concepts/gnosis-decision.md",
		"concepts/gnosis-directive.md",
		"concepts/gnosis-process.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	conceptsIndex, err := os.ReadFile(filepath.Join(root, "concepts", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conceptsIndex), "Gnosis Purpose") ||
		!strings.Contains(string(conceptsIndex), "Gnosis Process") {
		t.Fatalf("concepts index should list the concept definitions:\n%s", conceptsIndex)
	}
}

func TestRunScaffoldWithConceptsPreservesExistingFilesUnlessForced(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	custom := "---\ntype: Concept Type\ntitle: Custom Purpose\ndescription: Local custom concept.\n---\n\n# Custom Purpose\n"
	writeTestFile(t, root, "concepts/gnosis-purpose.md", custom)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"scaffold", "-vault", root, "-concepts"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	purposePath := filepath.Join(root, "concepts", "gnosis-purpose.md")
	preserved, err := os.ReadFile(purposePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(preserved) != custom {
		t.Fatal("expected scaffold to preserve an existing concept")
	}

	if err := run([]string{"scaffold", "-vault", root, "-concepts", "-force"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(purposePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updated), "title: Custom Purpose") {
		t.Fatal("expected forced scaffold to replace the existing concept")
	}
}

func TestRunScaffoldAndIndexHonorDisabledNavigation(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", `[vault]
vault_name = "Test"
vault_root = "."
vault_index = false
vault_log = false
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"scaffold", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"index.md", "log.md", "concepts/index.md", "references/index.md"} {
		if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
			t.Fatalf("disabled navigation file exists: %s", rel)
		}
	}

	stdout.Reset()
	if err := run([]string{"index", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "ok: index disabled") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunSetupCreatesImportWorkspace(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	workspace := filepath.Join(t.TempDir(), "workspace")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"scaffold", "-vault", source}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"setup", "-vault", workspace, "-import", source}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "ok: workspace configured") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "gnosis.toml")); err != nil {
		t.Fatal(err)
	}
	config, err := os.ReadFile(filepath.Join(workspace, "gnosis.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(config), "[[vaults]]\nvault_name = \"source\"") {
		t.Fatalf("gnosis.toml = %q", config)
	}
}

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunMissingCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err == nil || err.Error() != "missing command" {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
