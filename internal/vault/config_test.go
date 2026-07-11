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

func TestResolveConfigAllowsOnlyBundledDocumentation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeConfig(t, root, "[vaults.gnosis]\ninclude = [\"vault\"]\n")

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Config.VaultEnabled() || resolution.Config.ForgeEnabled() {
		t.Fatalf("forge = %t vault = %t, want vault only", resolution.Config.ForgeEnabled(), resolution.Config.VaultEnabled())
	}
	if len(resolution.Sources) != 0 {
		t.Fatalf("sources = %v, want none", resolution.Sources)
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

func TestDefaultBundledVaultDocumentationCanBeDisabled(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "."

[vaults.gnosis]
include = []
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Config.VaultEnabled() {
		t.Fatal("vault documentation is enabled, want disabled")
	}
	if resolution.Config.ForgeEnabled() {
		t.Fatal("forge documentation is enabled by default")
	}
}

func TestGnosisBundleIncludesSelectBundlesExplicitly(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "."

[vaults.gnosis]
include = ["forge"]
`)

	resolution, err := ResolveConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Config.ForgeEnabled() || resolution.Config.VaultEnabled() {
		t.Fatalf("forge = %t vault = %t, want forge only", resolution.Config.ForgeEnabled(), resolution.Config.VaultEnabled())
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
vault_root = "."
`)
	}
	writeConfig(t, first, `[vault]
vault_name = "first"
vault_root = "."

[vaults]
include = ["../third"]
`)
	writeConfig(t, workspace, `[vaults]
include = ["first", "second"]
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

func TestResolveConfigRejectsRemoteImportsAndCycles(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vaults]
include = ["https://github.com/leoxlin/gnosis.git"]
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "remote vault imports") {
		t.Fatalf("remote error = %v", err)
	}

	other := filepath.Join(root, "other")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `[vaults]
include = ["other"]
`)
	writeConfig(t, other, `[vaults]
include = [".."]
`)
	if _, err := ResolveConfig(root); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle error = %v", err)
	}
}
