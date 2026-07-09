package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefault(t *testing.T) {
	root := t.TempDir()
	config, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatRelative {
		t.Fatalf("link format = %q, want relative", config.Vault.LinkFormat)
	}
	if config.IsStrict() {
		t.Fatal("expected strict to default to false")
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

	config, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatAbsolute {
		t.Fatalf("link format = %q, want absolute", config.Vault.LinkFormat)
	}
	if !config.IsStrict() {
		t.Fatal("expected strict to be true")
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

	config, err := LoadConfig(vault)
	if err != nil {
		t.Fatal(err)
	}
	if config.LinkFormatValue() != LinkFormatAbsolute {
		t.Fatalf("link format = %q, want absolute", config.Vault.LinkFormat)
	}
}
