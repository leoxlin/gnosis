package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"
)

func TestGetVaultsListsEffectiveVaultsAsTOON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	registerCommandTarget(t, "workspace", workspace)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", "workspace", "get", "vaults"}, &stdout, &stderr); err != nil {
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
	t.Setenv("HOME", t.TempDir())
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "core"
vault_root = "."
`)
	registerCommandTarget(t, "core", workspace)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", "core", "get", "vaults"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "count: 1") ||
		!strings.Contains(stdout.String(), "core,local") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestGetConceptsAcceptsOnePositionalTypeAndFields(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "note.md", `---
type: Note
title: Keep it small
description: Prefer the smallest adequate design.
---
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", "test", "get", "concepts", "Note", "--fields", "title,uri",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "concepts[1]{title,uri}") ||
		!strings.Contains(stdout.String(), "Keep it small") {
		t.Fatalf("output = %q", stdout.String())
	}

	for _, args := range [][]string{
		{"get", "concepts", "Note", "Procedure", "--vault", "test"},
		{"get", "concepts", "--type", "Note", "--vault", "test"},
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
		"type: Note\n"+
		"title: Long note\n"+
		"description: Exercises bounded output.\n"+
		"---\n\n"+body)
	uri := "gnosis://test/long.md"

	var preview, stderr bytes.Buffer
	if err := run([]string{"--vault", "test", "get", "pages", uri}, &preview, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), "truncated: true") ||
		!strings.Contains(preview.String(), "--full") {
		t.Fatalf("preview = %q", preview.String())
	}

	var full bytes.Buffer
	if err := run([]string{"--vault", "test", "get", "pages", uri, "--full"}, &full, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.String(), "truncated: false") ||
		strings.Contains(full.String(), "help[") {
		t.Fatalf("full = %q", full.String())
	}
}

func TestGetPageProjectsTrustAndResolvesCurrent(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "current.md", "---\ntype: Note\ntitle: Current\nstatus: verified\n---\n")
	writeCommandFile(t, workspace, "old.md", "---\ntype: Note\ntitle: Old\nstatus: archived\nsuperseded_by: current.md\n---\n")

	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", "test", "get", "pages", "gnosis://test/old.md", "--resolve-current",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"trust:",
		"status: archived",
		"current: false",
		"current_resolution:",
		"status: current",
		`current: "gnosis://test/current.md"`,
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestGetPageReadsNamedRemoteTargetAndConfiguredRemoteImport(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/read.git")
	uri := "gnosis://remote/notes/remote.md"

	var stdout, stderr bytes.Buffer
	if err := run(
		[]string{"--vault", "remote", "get", "pages", uri, "--full"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Remote note") {
		t.Fatalf("direct remote page = %q", stdout.String())
	}

	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[[vaults]]
vault_name = "remote"
vault_root = "`+fixture.url+`"
`)
	registerCommandTarget(t, "composition", workspace)
	stdout.Reset()
	if err := run(
		[]string{"--vault", "composition", "get", "pages", uri, "--full"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Remote note") {
		t.Fatalf("configured remote import page = %q", stdout.String())
	}
}

func TestGetProceduresListsAndBoundsExecutionContract(t *testing.T) {
	commandVault(t)
	var listed, stderr bytes.Buffer
	if err := run([]string{
		"--vault", "test", "get", "procedures", "--tags", "gnosis,vault",
	}, &listed, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(listed.String(), "procedures[") ||
		!strings.Contains(listed.String(), "refining-procedure") {
		t.Fatalf("list = %q", listed.String())
	}

	uri := "gnosis://core/procedures/refining-procedure.md"
	var preview bytes.Buffer
	if err := run([]string{"--vault", "test", "get", "procedures", uri}, &preview, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.String(), "truncated: true") ||
		!strings.Contains(preview.String(), "--full") {
		t.Fatalf("preview = %q", preview.String())
	}

	var full bytes.Buffer
	if err := run([]string{"--vault", "test", "get", "procedures", uri, "--full"}, &full, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.String(), "truncated: false") ||
		strings.Contains(full.String(), "help[") {
		t.Fatalf("full = %q", full.String())
	}
}

func commandVault(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	registerCommandTarget(t, "test", workspace)
	return workspace
}

func registerCommandTarget(t *testing.T, name, target string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".config", "gnosis.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("\n[[vaults]]\nvault_name = " + strconv.Quote(name) + "\nvault_root = " + strconv.Quote(target) + "\n"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
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
