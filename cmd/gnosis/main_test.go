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
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"validate", "--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Usage:") || !strings.Contains(stdout.String(), "gnosis validate") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{
		{"version", "extra"},
		{"validate", "extra"},
		{"setup", "extra"},
		{"read", "extra"},
		{"query", "search", "one", "two"},
		{"query", "graph", "one", "two"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), "unexpected argument") {
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
	if stdout.String() != string(want) {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
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
	if stdout.String() != "Type: Concept\nDescription: A reusable knowledge record.\n\nType: Pattern\nDescription: Pattern\n\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	stdout.Reset()
	if err := run([]string{"concepts", "-vault", root, "-type", "Concept"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Title: Attention Mechanism\nDescription: Weighted token lookup.\n") ||
		!strings.Contains(stdout.String(), "Title: Transformer Architecture\nDescription: Self-attention model.\n") ||
		strings.Contains(stdout.String(), "Path:") ||
		strings.Contains(stdout.String(), "Self-attention details live only in the body") {
		t.Fatalf("stdout = %q", stdout.String())
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
	writeTestFile(t, root, "gnosis.toml", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n\n[vaults.gnosis]\ninclude = []\n")
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
		"concepts/repository-purpose.md",
		"concepts/repository-decision.md",
		"concepts/repository-directive.md",
		"concepts/repository-process.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	conceptsIndex, err := os.ReadFile(filepath.Join(root, "concepts", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conceptsIndex), "Repository Purpose") ||
		!strings.Contains(string(conceptsIndex), "Repository Process") {
		t.Fatalf("concepts index should list the concept definitions:\n%s", conceptsIndex)
	}
}

func TestRunScaffoldWithConceptsPreservesExistingFilesUnlessForced(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	custom := "---\ntype: Concept Type\ntitle: Custom Purpose\ndescription: Local custom concept.\n---\n\n# Custom Purpose\n"
	writeTestFile(t, root, "concepts/repository-purpose.md", custom)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"scaffold", "-vault", root, "-concepts"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	purposePath := filepath.Join(root, "concepts", "repository-purpose.md")
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
	if !strings.Contains(string(config), "[vaults.gnosis]\ninclude = [\"forge\"]") {
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
