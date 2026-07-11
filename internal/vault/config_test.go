package vault

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveConfigUsesDefaultsWithoutConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolution.Config, DefaultConfig()) {
		t.Fatalf("config = %#v, want defaults %#v", resolution.Config, DefaultConfig())
	}
	if got, want := resolution.VaultRoots, []string(nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
	if got, want := resolution.Sources, []VaultSource(nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v, want none", got)
	}
}

func TestResolveConfigAlwaysIncludesBundledDocumentation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeConfig(t, root, "")

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Config.VaultEnabled() || !resolution.Config.ForgeEnabled() {
		t.Fatalf("forge = %t vault = %t, want both bundles", resolution.Config.ForgeEnabled(), resolution.Config.VaultEnabled())
	}
	if len(resolution.Sources) != 0 {
		t.Fatalf("sources = %v, want none", resolution.Sources)
	}
}

func TestResolveConfigLoadsMultipleDeclaredVaultsInOrder(t *testing.T) {
	root := t.TempDir()
	obsidian := filepath.Join(root, "obsidian")
	if err := os.Mkdir(obsidian, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `[[vaults]]
vault_name = "obsidian"
vault_root = "obsidian"
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolution.VaultRoots, []string{obsidian}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
}

func TestResolveConfigPrefersLocalConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Project"
vault_root = "."
`)
	if err := os.WriteFile(filepath.Join(root, "gnosis.local.toml"), []byte(`[vault]
vault_name = "Local"
vault_root = "."
`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, configDir, `[vault]
vault_name = "Global"
vault_root = "."
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolution.Config.Vault.Name, "Local"; got != want {
		t.Fatalf("vault name = %q, want %q", got, want)
	}
	if got, want := resolution.Sources[0].Config.Vault.Name, "Local"; got != want {
		t.Fatalf("source vault name = %q, want %q", got, want)
	}
}

func TestResolveConfigDoesNotReadParentConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	parent := t.TempDir()
	writeConfig(t, parent, `[vault]
vault_name = "Parent"
vault_root = "."
`)
	root := filepath.Join(parent, "child")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resolution.Config, DefaultConfig()) {
		t.Fatalf("config = %#v, want defaults %#v", resolution.Config, DefaultConfig())
	}
}

func TestResolveConfigUsesGlobalConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, configDir, `[vault]
vault_name = "Global"
vault_root = "."
`)

	resolution, err := ResolveConfig(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolution.Config.Vault.Name, "Global"; got != want {
		t.Fatalf("vault name = %q, want %q", got, want)
	}
}

func TestResolveConfigReadsLocalRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "docs"
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "docs")}
	if strings.Join(resolution.VaultRoots, ",") != strings.Join(want, ",") {
		t.Fatalf("vault roots = %v, want %v", resolution.VaultRoots, want)
	}
}

func TestConfigAlwaysIncludesAllBundledDocumentation(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "."
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Config.VaultEnabled() || !resolution.Config.ForgeEnabled() {
		t.Fatalf("forge = %t vault = %t, want both bundles", resolution.Config.ForgeEnabled(), resolution.Config.VaultEnabled())
	}
}

func TestResolveConfigRejectsNestedVaultImports(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"Local\"\nvault_root = \".\"\n\n[vault.imports]\nvaults = [\"other\"]\n")

	_, err := ResolveConfig(root)
	if err == nil {
		t.Fatalf("error = %v, want nested vault imports to be rejected", err)
	}
}

func TestResolveConfigLoadsDeclaredVaultsInOrder(t *testing.T) {
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
vault_root = "."
`)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "first"
vault_root = "first"

[[vaults]]
vault_name = "third"
vault_root = "third"

[[vaults]]
vault_name = "second"
vault_root = "second"
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

func TestResolveConfigRejectsDeprecatedVaultDirs(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_dirs = ["docs"]
`)

	_, err := ResolveConfig(root)
	if err == nil {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveConfigRejectsInvalidDeclaredVaults(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[[vaults]]
vault_name = "missing"
vault_root = "missing"
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing vault error = %v", err)
	}

	writeConfig(t, root, `[[vaults]]
vault_name = ""
vault_root = "."
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "vault_name") {
		t.Fatalf("empty name error = %v", err)
	}
}
