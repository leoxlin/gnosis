package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gnosis/internal/vault"
)

func TestVaultsListsEffectiveVaultsAsJSON(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.Mkdir(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "gnosis.toml"), []byte(`[vault]
vault_name = "workspace"
vault_root = "."

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imported, "gnosis.toml"), []byte(`[vault]
vault_name = "imported"
vault_root = "."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run(
		[]string{"vaults", "--vault", workspace, "--json"},
		&stdout,
		&stderr,
	); err != nil {
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

func TestVaultsDoesNotRepeatConfiguredCore(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "gnosis.toml"), []byte(`[vault]
vault_name = "core"
vault_root = "."
`), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run(
		[]string{"vaults", "--vault", workspace, "--json"},
		&stdout,
		&stderr,
	); err != nil {
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
