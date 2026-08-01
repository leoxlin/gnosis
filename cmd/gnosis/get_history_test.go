package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

func TestGetHistoryDiffAndChanges(t *testing.T) {
	root := historyCommandVault(t)
	uri := "gnosis://test/note.md"

	var stdout, stderr bytes.Buffer
	if err := run(
		[]string{"--vault", "test", "get", "history", uri, "--limit", "1"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"current: present", "classification: updated", "next_cursor:", "help[1]:",
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("history output = %q, missing %q", stdout.String(), value)
		}
	}

	history, err := vault.ReadPageHistory(root, uri, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	from := history.Entries[len(history.Entries)-1].Revision
	to := history.Entries[0].Revision
	stdout.Reset()
	if err := run(
		[]string{
			"--vault", "test", "get", "diff", uri,
			"--from", from, "--to", to, "--limit", "1000",
		},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"from_revision:", "to_revision:", "diff:", "-first", "+second"} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("diff output = %q, missing %q", stdout.String(), value)
		}
	}

	stdout.Reset()
	if err := run(
		[]string{"--vault", "test", "get", "changes"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"count: 0", "next_cursor:", "help[1]:"} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("changes output = %q, missing %q", stdout.String(), value)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func historyCommandVault(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	runCommandGit(t, "init", "--initial-branch=main", root)
	runCommandGit(t, "-C", root, "config", "user.name", "gnosis test")
	runCommandGit(t, "-C", root, "config", "user.email", "gnosis@example.test")
	writeCommandFile(t, root, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "docs"
vault_index = false
vault_log = false
`)
	writeCommandFile(t, root, "docs/concepts/note.md", `---
type: Concept
title: Note
description: A test note.
path: .
---

# Note
`)
	writeCommandFile(t, root, "docs/note.md", `---
type: Note
title: History
---

# History

first
`)
	runCommandGit(t, "-C", root, "add", ".")
	runCommandGit(t, "-C", root, "commit", "-m", "add note")
	writeCommandFile(t, root, filepath.Join("docs", "note.md"), `---
type: Note
title: History
---

# History

second
`)
	runCommandGit(t, "-C", root, "add", ".")
	runCommandGit(t, "-C", root, "commit", "-m", "update note")
	registerCommandTarget(t, "test", root)
	return root
}
