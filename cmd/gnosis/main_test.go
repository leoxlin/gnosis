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
	if !strings.Contains(stdout.String(), "Usage: gnosis validate") {
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
		{"query", "one", "two"},
		{"graph-query", "one", "two"},
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
		{name: "compact", args: []string{"graph-query", "-vault", root, "transformer"}},
		{name: "pretty", args: []string{"graph-query", "-vault", root, "-pretty", "transformer"}},
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
	if err := run([]string{"query", "-vault", root, "transformer"}, &stdout, &stderr); err != nil {
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
	if err := run([]string{"query", "-vault", root, "-pretty", "Transformer Architecture"}, &stdout, &stderr); err != nil {
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
		{args: []string{"query", "-vault", root}, want: "missing question"},
		{args: []string{"query", "-vault", root, "-top", "0", "query"}, want: "-top"},
		{args: []string{"query", "-vault", root, "-max-read", "-1", "query"}, want: "-max-read"},
		{args: []string{"graph-query", "-vault", root, "-depth", "0", "query"}, want: "-depth"},
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
	if err := run([]string{"graph-query", "-vault", root, "transformer"}, &stdout, &stderr); err != nil {
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

func queryTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
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
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		err := run([]string{"validate", "-vault", t.TempDir()}, &stdout, &stderr)
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

func TestRunSetupAndIndex(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"setup", "-vault", root}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "ok: vault setup") || stderr.Len() != 0 {
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

func TestRunSetupWithConceptsIndexesThem(t *testing.T) {
	root := filepath.Join(t.TempDir(), "vault")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"setup", "-vault", root, "-concepts"}, &stdout, &stderr); err != nil {
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

func TestRunSetupAndIndexHonorDisabledNavigation(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "gnosis.toml", `[vault]
vault_index = false
vault_log = false
`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"setup", "-vault", root}, &stdout, &stderr); err != nil {
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
