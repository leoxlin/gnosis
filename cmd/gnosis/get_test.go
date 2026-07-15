package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

func TestRootUsesResourceCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"get", "search", "index"} {
		if !strings.Contains(stdout.String(), "  "+command+" ") {
			t.Fatalf("help does not contain %q:\n%s", command, stdout.String())
		}
	}
	for _, command := range []string{"vaults", "concepts", "query"} {
		stdout.Reset()
		stderr.Reset()
		if err := run([]string{command}, &stdout, &stderr); err == nil {
			t.Fatalf("removed command %q succeeded", command)
		}
	}
}

func TestGetVaultsListsEffectiveVaultsAsJSON(t *testing.T) {
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
	if err := run([]string{"get", "vaults", "--vault", workspace, "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var catalog vault.VaultCatalog
	if err := json.Unmarshal(stdout.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	want := []vault.Origin{
		{Vault: "workspace", Kind: vault.OriginLocal, Root: workspace, Precedence: 0},
		{Vault: "imported", Kind: vault.OriginImport, Root: imported, Precedence: 1},
		{Vault: "core", Kind: vault.OriginBundle, Precedence: 2},
	}
	if !reflect.DeepEqual(catalog.Vaults, want) {
		t.Fatalf("vaults = %+v, want %+v", catalog.Vaults, want)
	}
}

func TestGetVaultsDoesNotRepeatConfiguredCore(t *testing.T) {
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "core"
vault_root = "."
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"get", "vaults", "--vault", workspace, "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var catalog vault.VaultCatalog
	if err := json.Unmarshal(stdout.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	want := []vault.Origin{{Vault: "core", Kind: vault.OriginLocal, Root: workspace, Precedence: 0}}
	if !reflect.DeepEqual(catalog.Vaults, want) {
		t.Fatalf("vaults = %+v, want %+v", catalog.Vaults, want)
	}
}

func TestGetConceptsAcceptsOnePositionalType(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "decision.md", `---
type: Decision
title: Keep it small
description: Prefer the smallest adequate design.
---
`)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"get", "concepts", "Decision", "--vault", workspace, "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var catalog vault.ConceptRecordCatalog
	if err := json.Unmarshal(stdout.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if got := catalog["concepts"]; len(got) != 1 || got[0]["title"] != "Keep it small" {
		t.Fatalf("catalog = %#v", catalog)
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
