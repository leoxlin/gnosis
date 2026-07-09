package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefault(t *testing.T) {
	root := t.TempDir()
	config, vaultRoots, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatRelative {
		t.Fatalf("link format = %q, want relative", config.Vault.LinkFormat)
	}
	if config.IsStrict() {
		t.Fatal("expected strict to default to false")
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != root {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, root)
	}
}

func TestLoadConfigFromVaultRoot(t *testing.T) {
	root := t.TempDir()
	content := `[vault]
link_format = "absolute"
link_format_strict = true
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	config, vaultRoots, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatAbsolute {
		t.Fatalf("link format = %q, want absolute", config.Vault.LinkFormat)
	}
	if !config.IsStrict() {
		t.Fatal("expected strict to be true")
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != root {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, root)
	}
}

func TestLoadConfigWalksUp(t *testing.T) {
	root := t.TempDir()
	vault := filepath.Join(root, "docs")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `[vault]
link_format = "absolute"
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	config, vaultRoots, err := LoadConfig(vault)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatAbsolute {
		t.Fatalf("link format = %q, want absolute", config.Vault.LinkFormat)
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != root {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, root)
	}
}

func TestLoadConfigVaultRoots(t *testing.T) {
	root := t.TempDir()
	vault := filepath.Join(root, "docs")
	if err := os.MkdirAll(vault, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `[vault]
vault_roots = ["docs"]
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, vaultRoots, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != vault {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, vault)
	}
}

func TestLoadConfigMultipleVaultRoots(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	notes := filepath.Join(root, "notes")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `[vault]
vault_roots = ["docs", "notes"]
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, vaultRoots, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{docs, notes}
	if len(vaultRoots) != len(want) {
		t.Fatalf("vault roots = %v, want %v", vaultRoots, want)
	}
	for i, got := range vaultRoots {
		if got != want[i] {
			t.Fatalf("vault roots = %v, want %v", vaultRoots, want)
		}
	}
}
