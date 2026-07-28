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

func TestCreateVaultPublishesRemoteChangesAndSkipsRemoteNoOp(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/create.git")
	args := []string{"--vault", fixture.url, "create", "vault", "--name", "remote", "--concepts"}
	before := commandRemoteCommitCount(t, fixture)

	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := commandRemoteCommitCount(t, fixture); got != before+1 {
		t.Fatalf("create commit count = %d, want %d", got, before+1)
	}
	if !strings.Contains(stdout.String(), "changed: true") {
		t.Fatalf("create output = %q", stdout.String())
	}

	rejectCommandRemotePushes(t, fixture)
	stdout.Reset()
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("no-op create attempted publication: %v", err)
	}
	if !strings.Contains(stdout.String(), "status: no-op") ||
		!strings.Contains(stdout.String(), "changed: false") {
		t.Fatalf("second create = %q", stdout.String())
	}
}

func TestCreateVaultReportsRemotePublicationFailure(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/create-rejected.git")
	before := commandRemoteCommitCount(t, fixture)
	rejectCommandRemotePushes(t, fixture)

	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--vault", fixture.url, "create", "vault", "--name", "remote"},
		&stdout,
		&stderr,
	)
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("create publication error = %v", err)
	}
	if got := commandRemoteCommitCount(t, fixture); got != before {
		t.Fatalf("rejected create commit count = %d, want %d", got, before)
	}
}

func TestApplyWorkspacePublishesRemoteChangesAndSkipsRemoteNoOp(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/workspace.git")
	args := []string{
		"--vault", fixture.url, "apply", "workspace",
		"--import", "/tmp/imported", "--force",
	}
	before := commandRemoteCommitCount(t, fixture)

	var stdout, stderr bytes.Buffer
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := commandRemoteCommitCount(t, fixture); got != before+1 {
		t.Fatalf("workspace commit count = %d, want %d", got, before+1)
	}

	rejectCommandRemotePushes(t, fixture)
	stdout.Reset()
	if err := run(args, &stdout, &stderr); err != nil {
		t.Fatalf("no-op workspace apply attempted publication: %v", err)
	}
	if !strings.Contains(stdout.String(), "status: no-op") ||
		!strings.Contains(stdout.String(), "changed: false") {
		t.Fatalf("second workspace apply = %q", stdout.String())
	}
}

func TestApplyWorkspaceReportsRemotePublicationFailure(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/workspace-rejected.git")
	before := commandRemoteCommitCount(t, fixture)
	rejectCommandRemotePushes(t, fixture)

	var stdout, stderr bytes.Buffer
	err := run(
		[]string{
			"--vault", fixture.url, "apply", "workspace",
			"--import", "/tmp/imported", "--force",
		},
		&stdout,
		&stderr,
	)
	if err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("workspace publication error = %v", err)
	}
	if got := commandRemoteCommitCount(t, fixture); got != before {
		t.Fatalf("rejected workspace commit count = %d, want %d", got, before)
	}
}

func TestApplyPageAcknowledgesRepeatWithBodyLinksAsNoOp(t *testing.T) {
	workspace := commandVault(t)
	dir := t.TempDir()
	target := filepath.Join(dir, "target.md")
	targetContent := `---
type: Concept
title: Link target
description: The linked record.
---
`
	if err := os.WriteFile(target, []byte(targetContent), 0o644); err != nil {
		t.Fatal(err)
	}
	var targetOut, targetErr bytes.Buffer
	if err := run([]string{
		"--vault", workspace, "apply", "page",
		"gnosis://test/concepts/repeat-safely.md", "--filename", target,
	}, &targetOut, &targetErr); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "concept.md")
	content := `---
type: Concept
title: Link repeat
description: A repeat apply with body links is a no-op.
---

# Concept

See [Repeat safely](repeat-safely.md).
`
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"--vault", workspace, "apply", "page",
		"gnosis://test/concepts/link-repeat.md", "--filename", input,
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
