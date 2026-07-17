package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"
)

func TestGetVaultsListsEffectiveVaultsAsTOON(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.Mkdir(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "workspace"
vault_root = "."

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeCommandFile(t, imported, "gnosis.toml", `[vault]
vault_name = "imported"
vault_root = "."
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "vaults"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if _, err := toon.Decode(stdout.Bytes()); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	for _, value := range []string{
		"count: 3",
		"vaults[3]{vault,kind,root}",
		"workspace,local",
		"imported,import",
		"core,bundle",
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestGetVaultsDoesNotRepeatConfiguredCore(t *testing.T) {
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "core"
vault_root = "."
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "vaults"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "count: 1") ||
		!strings.Contains(stdout.String(), "core,local") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestGetConceptsAcceptsOnePositionalTypeAndFields(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "decision.md", `---
type: Decision
title: Keep it small
description: Prefer the smallest adequate design.
---
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", workspace, "get", "concepts", "Decision", "--fields", "title,uri",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "concepts[1]{title,uri}") ||
		!strings.Contains(stdout.String(), "Keep it small") {
		t.Fatalf("output = %q", stdout.String())
	}

	for _, args := range [][]string{
		{"get", "concepts", "Decision", "Procedure", "--vault", workspace},
		{"get", "concepts", "--type", "Decision", "--vault", workspace},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := run(args, &stdout, &stderr); err == nil {
			t.Fatalf("run(%q) succeeded", args)
		}
	}
}

func TestGetPagePreviewAndFullContent(t *testing.T) {
	workspace := commandVault(t)
	body := strings.Repeat("界", detailPreviewLimit+1)
	writeCommandFile(t, workspace, "long.md", "---\n"+
		"type: Decision\n"+
		"title: Long decision\n"+
		"description: Exercises bounded output.\n"+
		"---\n\n"+body)
	uri := "gnosis://test/long.md"

	var preview, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "pages", uri}, &preview, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), "truncated: true") ||
		!strings.Contains(preview.String(), "--full") {
		t.Fatalf("preview = %q", preview.String())
	}

	var full bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "pages", uri, "--full"}, &full, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.String(), "truncated: false") ||
		strings.Contains(full.String(), "help[") {
		t.Fatalf("full = %q", full.String())
	}
}

func TestGetProceduresListsAndBoundsExecutionContract(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	workspace := t.TempDir()
	var listed, stderr bytes.Buffer
	if err := run([]string{
		"--vault", workspace, "get", "procedures", "--tags", "gnosis,development",
	}, &listed, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(listed.String(), "procedures[") ||
		!strings.Contains(listed.String(), "implementing-directive") {
		t.Fatalf("list = %q", listed.String())
	}

	uri := "gnosis://core/procedures/development/implementing-directive.md"
	var preview bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "procedures", uri}, &preview, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), "truncated: true") ||
		!strings.Contains(preview.String(), "--full") {
		t.Fatalf("preview = %q", preview.String())
	}

	var full bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "procedures", uri, "--full"}, &full, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.String(), "truncated: false") ||
		strings.Contains(full.String(), "help[") {
		t.Fatalf("full = %q", full.String())
	}
}

func commandVault(t *testing.T) string {
	t.Helper()
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	return workspace
}

func writeCommandFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGetDirectivesListsStatusAndDerivedProgress(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "directives/alpha.md", `---
type: Directive
title: Alpha
description: First.
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
- [ ] another open step

# Acceptance criteria

A.
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "get", "directives"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"directives[1]{uri,title,status,tasks_done,tasks_total}",
		"Alpha",
		"open,1,3",
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}
}
