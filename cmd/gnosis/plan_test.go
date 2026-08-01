package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPlanAndApplyKnowledgeChangeCLI(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "concepts/note.md", `---
type: Concept
title: Note
description: A short general-purpose record.
path: notes
---
`)
	candidate := filepath.Join(t.TempDir(), "candidate.md")
	if err := os.WriteFile(candidate, []byte(`---
type: Note
title: Planned
description: A two-phase change.
---

# Planned
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", "test",
		"plan", "knowledge-change", "gnosis://test/notes/planned.md",
		"--expected-absent", "--filename", candidate,
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	for _, expected := range []string{
		"resource: knowledge-change",
		"operation: create",
		"applicable: true",
		"diff:",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("plan output = %q, missing %q", output, expected)
		}
	}
	digest := regexp.MustCompile(`digest: "(sha256:[a-f0-9]+)"`).FindStringSubmatch(output)
	if len(digest) != 2 {
		t.Fatalf("plan output = %q, missing digest", output)
	}
	if _, err := os.Stat(filepath.Join(workspace, "notes", "planned.md")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote page: %v", err)
	}

	stdout.Reset()
	if err := run([]string{
		"--vault", "test",
		"apply", "knowledge-change", "gnosis://test/notes/planned.md",
		"--expected-absent", "--digest", digest[1], "--filename", candidate,
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "status: applied") ||
		!strings.Contains(stdout.String(), "changed: true") {
		t.Fatalf("apply output = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(workspace, "notes", "planned.md")); err != nil {
		t.Fatal(err)
	}
}

func TestKnowledgeChangeCLIRequiresOneExpectedState(t *testing.T) {
	commandVault(t)
	for _, args := range [][]string{
		{"--vault", "test", "plan", "knowledge-change", "gnosis://test/notes/page.md"},
		{
			"--vault", "test", "plan", "knowledge-change", "gnosis://test/notes/page.md",
			"--expected-absent", "--expected-revision", "sha256:test",
		},
	} {
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		if err == nil || exitCode(err) != 2 ||
			!strings.Contains(err.Error(), "provide exactly one") {
			t.Fatalf("error = %v", err)
		}
	}
}
