package vault

import (
	"os"
	"path/filepath"
	"strings"
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
	if !config.IndexEnabled() {
		t.Fatal("expected vault index to default to true")
	}
	if !config.LogEnabled() {
		t.Fatal("expected vault log to default to true")
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != root {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, root)
	}
}

func TestLoadConfigVaultNavigationFlags(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_index = false
vault_log = false
`)

	config, _, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.IndexEnabled() {
		t.Fatal("expected vault index to be disabled")
	}
	if config.LogEnabled() {
		t.Fatal("expected vault log to be disabled")
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
	if !config.IndexEnabled() || !config.LogEnabled() {
		t.Fatal("expected omitted vault navigation settings to retain true defaults")
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

func TestLoadConfigRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nunknown = true\n")

	_, _, err := LoadConfig(root)
	if err == nil || !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("error = %v, want unknown field error", err)
	}
}

func TestLoadConfigRejectsInvalidLinkFormat(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nlink_format = \"wiki\"\n")

	_, _, err := LoadConfig(root)
	if err == nil || !strings.Contains(err.Error(), "link_format") {
		t.Fatalf("error = %v, want link format error", err)
	}
}

func TestLoadConfigRejectsUnsafeVaultRoots(t *testing.T) {
	tests := []struct {
		name    string
		roots   string
		message string
	}{
		{name: "empty", roots: `[""]`, message: "must not be empty"},
		{name: "absolute quoted", roots: `["/tmp"]`, message: "must be relative"},
		{name: "parent", roots: `["../notes"]`, message: "escapes"},
		{name: "duplicate", roots: `["docs", "./docs"]`, message: "duplicates"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, "[vault]\nvault_roots = "+test.roots+"\n")

			_, _, err := LoadConfig(root)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want error containing %q", err, test.message)
			}
		})
	}
}

func TestLoadConfigAllowsCurrentDirectoryVaultRoot(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_roots = [\".\"]\n")

	_, vaultRoots, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(vaultRoots) != 1 || vaultRoots[0] != root {
		t.Fatalf("vault roots = %v, want [%s]", vaultRoots, root)
	}
}
