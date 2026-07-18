package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyWorkspaceGitHubWiki(t *testing.T) {
	workspace := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", workspace, "apply", "workspace",
		"--github-wiki", "OWNER/REPOSITORY", "--name", "wiki",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "gnosis.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[vault]
vault_name = "wiki"
backend = "github-wiki"
repository = "OWNER/REPOSITORY"
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
`
	if string(content) != want {
		t.Fatalf("gnosis.toml = %q, want %q", content, want)
	}
	if !strings.Contains(stdout.String(), "resource: workspace") ||
		!strings.Contains(stdout.String(), "changed: true") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestApplyWorkspaceRejectsInvalidFlagCombinations(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"missing name", []string{"apply", "workspace", "--github-wiki", "OWNER/REPOSITORY"}, "--name is required"},
		{"mixed import", []string{"apply", "workspace", "--github-wiki", "OWNER/REPOSITORY", "--name", "wiki", "--import", "local"}, "cannot be combined"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(test.args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.want) || exitCode(err) != 2 {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestApplyPageAcknowledgesRepeatAsNoOp(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "concepts/note.md", `---
type: ConceptType
title: Note
description: A short general-purpose record.
path: notes
---

# Note
`)
	input := filepath.Join(t.TempDir(), "note.md")
	content := `---
type: Note
title: Repeat safely
description: Repeating the same apply changes nothing.
---
`
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"--vault", workspace, "apply", "page",
		"gnosis://test/notes/repeat-safely.md", "--filename", input,
	}

	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "changed: true") {
		t.Fatalf("first apply = %q", stdout.String())
	}

	stdout.Reset()
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "status: no-op") ||
		!strings.Contains(stdout.String(), "changed: false") {
		t.Fatalf("second apply = %q", stdout.String())
	}
}

func TestCreateVaultAcknowledgesRepeatAsNoOp(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "vault")
	args := []string{"--vault", workspace, "create", "vault", "--name", "repeat"}

	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "changed: true") {
		t.Fatalf("first create = %q", stdout.String())
	}

	stdout.Reset()
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "status: no-op") ||
		!strings.Contains(stdout.String(), "changed: false") {
		t.Fatalf("second create = %q", stdout.String())
	}
}
