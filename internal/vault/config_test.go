package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigRequiresGnosisToml(t *testing.T) {
	_, err := ResolveConfig(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no gnosis.toml") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveConfigReadsOrderedLocalDirectories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"docs", "notes"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_dirs = ["docs", "notes"]
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "docs"), filepath.Join(root, "notes")}
	if strings.Join(resolution.VaultRoots, ",") != strings.Join(want, ",") {
		t.Fatalf("vault roots = %v, want %v", resolution.VaultRoots, want)
	}
}

func TestResolveConfigRecursivelyImportsVaultsInOrder(t *testing.T) {
	workspace := t.TempDir()
	first := filepath.Join(workspace, "first")
	second := filepath.Join(workspace, "second")
	third := filepath.Join(workspace, "third")
	for _, root := range []string{first, second, third} {
		if err := os.Mkdir(root, 0o755); err != nil {
			t.Fatal(err)
		}
		writeConfig(t, root, `[vault]
vault_name = "`+filepath.Base(root)+`"
vault_dirs = ["."]
`)
	}
	writeConfig(t, first, `[vault]
vault_name = "first"
vault_dirs = ["."]

[vault.imports]
vaults = ["../third"]
`)
	writeConfig(t, workspace, `[vault.imports]
vaults = ["first", "second"]
`)

	resolution, err := ResolveConfig(workspace)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{first, third, second}
	if strings.Join(resolution.VaultRoots, ",") != strings.Join(want, ",") {
		t.Fatalf("vault roots = %v, want %v", resolution.VaultRoots, want)
	}
}

func TestResolveConfigRejectsRemoteImportsAndCycles(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault.imports]
vaults = ["https://github.com/leoxlin/gnosis.git"]
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "remote vault imports") {
		t.Fatalf("remote error = %v", err)
	}

	other := filepath.Join(root, "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `[vault.imports]
vaults = ["other"]
`)
	writeConfig(t, other, `[vault.imports]
vaults = [".."]
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle error = %v", err)
	}
}
