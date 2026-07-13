package vault

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadEffectiveVaultUsesDefaultsWithoutConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vault.config, DefaultConfig()) {
		t.Fatalf("config = %#v, want defaults %#v", vault.config, DefaultConfig())
	}
	if got, want := vault.sources, []vaultSource(nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v, want none", got)
	}
}

func TestDefaultConfigEnablesVaultProcesses(t *testing.T) {
	if got, want := DefaultConfig().Gnosis.Processes, []string{"vault"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("processes = %v, want %v", got, want)
	}
}

func TestProcessEnabledNormalizesConfiguredFamilies(t *testing.T) {
	config := DefaultConfig()
	config.Gnosis.Processes = []string{" gnosis-vault "}
	if !config.ProcessEnabled([]string{"vault"}) {
		t.Fatal("legacy configured process family was not enabled")
	}
	config.Gnosis.Processes = []string{"vault"}
	if !config.ProcessEnabled([]string{"gnosis-vault"}) {
		t.Fatal("legacy authored process family was not enabled")
	}
}

func TestLoadEffectiveVaultLoadsProcessTags(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "."

[gnosis]
processes = ["gnosis-vault", "gnosis-planning"]
`)

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := vault.config.Gnosis.Processes, []string{"gnosis-vault", "gnosis-planning"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("processes = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultWithEmptyFileHasNoSources(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeConfig(t, root, "")

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(vault.sources) != 0 {
		t.Fatalf("sources = %v, want none", vault.sources)
	}
}

func TestLoadEffectiveVaultLoadsMultipleDeclaredVaultsInOrder(t *testing.T) {
	root := t.TempDir()
	obsidian := filepath.Join(root, "obsidian")
	writeConfig(t, obsidian, `[vault]
vault_name = "obsidian"
vault_root = "."
`)
	writeConfig(t, root, `[[vaults]]
vault_name = "obsidian"
vault_root = "obsidian"
`)

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourcePaths(vault), []string{obsidian}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultLoadsImportedVaultsDepthFirstWithTheirOwnSettings(t *testing.T) {
	workspace := t.TempDir()
	first := filepath.Join(workspace, "first")
	nested := filepath.Join(workspace, "nested")
	second := filepath.Join(workspace, "second")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"

[[vaults]]
vault_name = "declared-first"
vault_root = "first"

[[vaults]]
vault_name = "declared-second"
vault_root = "second"
`)
	writeConfig(t, first, `[vault]
vault_name = "first"
vault_root = "pages"
link_format = "absolute"
vault_index = false
vault_log = false

[[vaults]]
vault_name = "nested"
vault_root = "../nested"
`)
	writeConfig(t, nested, `[vault]
vault_name = "nested"
vault_root = "."
`)
	writeConfig(t, second, `[vault]
vault_name = "second"
vault_root = "knowledge"
`)
	for _, dir := range []string{filepath.Join(workspace, "local"), filepath.Join(first, "pages"), filepath.Join(second, "knowledge")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	vault, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	wantRoots := []string{
		filepath.Join(workspace, "local"),
		filepath.Join(first, "pages"),
		nested,
		filepath.Join(second, "knowledge"),
	}
	if got := sourcePaths(vault); !reflect.DeepEqual(got, wantRoots) {
		t.Fatalf("vault roots = %v, want %v", got, wantRoots)
	}
	if got, want := vault.sources[0].vaultRoot, workspace; got != want {
		t.Fatalf("local source root = %v, want %v", got, want)
	}
	if got, want := vault.sources[1].config.Vault.Name, "first"; got != want {
		t.Fatalf("first source vault name = %q, want imported config name %q", got, want)
	}
	if got := vault.sources[1].config.LinkFormatValue(); got != LinkFormatAbsolute {
		t.Fatalf("first source link format = %q, want %q", got, LinkFormatAbsolute)
	}
	if vault.sources[1].config.IndexEnabled() || vault.sources[1].config.LogEnabled() {
		t.Fatalf("first source settings = %+v, want imported settings", vault.sources[1].config.Vault)
	}
}

func TestLoadEffectiveVaultDeduplicatesVaultReachedByMultiplePaths(t *testing.T) {
	workspace := t.TempDir()
	left := filepath.Join(workspace, "left")
	right := filepath.Join(workspace, "right")
	shared := filepath.Join(workspace, "shared")
	writeConfig(t, workspace, `[[vaults]]
vault_name = "left"
vault_root = "left"

[[vaults]]
vault_name = "right"
vault_root = "right"
`)
	for _, root := range []string{left, right} {
		writeConfig(t, root, `[vault]
vault_name = "`+filepath.Base(root)+`"
vault_root = "."

[[vaults]]
vault_name = "shared"
vault_root = "../shared"
`)
	}
	writeConfig(t, shared, `[vault]
vault_name = "shared"
vault_root = "."
`)

	vault, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourcePaths(vault), []string{left, shared, right}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultDeduplicatesSymlinkedImport(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	alias := filepath.Join(workspace, "imported-alias")
	writeConfig(t, imported, "[vault]\nvault_name = \"imported\"\nvault_root = \".\"\n")
	if err := os.Symlink(imported, alias); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "imported"
vault_root = "imported"

[[vaults]]
vault_name = "imported-alias"
vault_root = "imported-alias"
`)

	vault, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourcePaths(vault), []string{imported}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want canonical import once as %v", got, want)
	}
}

func TestLoadEffectiveVaultRejectsImportCycles(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, `[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "imported"
vault_root = "."

[[vaults]]
vault_name = "workspace"
vault_root = ".."
`)

	_, err := loadEffectiveVault(workspace)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle error = %v", err)
	}
}

func TestLoadEffectiveVaultRequiresImportedVaultConfiguration(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.Mkdir(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)

	_, err := loadEffectiveVault(workspace)
	if err == nil || !strings.Contains(err.Error(), filepath.Join(imported, "gnosis.toml")) {
		t.Fatalf("missing imported config error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsRemoteVaultImports(t *testing.T) {
	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "remote"
vault_root = "https://example.com/remote-vault.git"
`)

	_, err := loadEffectiveVault(workspace)
	if err == nil || !strings.Contains(err.Error(), "remote vault imports are not supported") {
		t.Fatalf("remote import error = %v", err)
	}
}

func TestLoadEffectiveVaultPrefersLocalConfiguration(t *testing.T) {
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

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := vault.config.Vault.Name, "Local"; got != want {
		t.Fatalf("vault name = %q, want %q", got, want)
	}
	if got, want := vault.sources[0].config.Vault.Name, "Local"; got != want {
		t.Fatalf("source vault name = %q, want %q", got, want)
	}
}

func TestLoadEffectiveVaultDoesNotReadParentConfiguration(t *testing.T) {
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

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vault.config, DefaultConfig()) {
		t.Fatalf("config = %#v, want defaults %#v", vault.config, DefaultConfig())
	}
}

func TestLoadEffectiveVaultUsesGlobalConfiguration(t *testing.T) {
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

	vault, err := loadEffectiveVault(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := vault.config.Vault.Name, "Global"; got != want {
		t.Fatalf("vault name = %q, want %q", got, want)
	}
}

func TestLoadEffectiveVaultReadsLocalRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "docs"
`)

	vault, err := loadEffectiveVault(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "docs")}
	if got := sourcePaths(vault); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultRejectsNestedVaultImports(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"Local\"\nvault_root = \".\"\n\n[vault.imports]\nvaults = [\"other\"]\n")

	_, err := loadEffectiveVault(root)
	if err == nil {
		t.Fatalf("error = %v, want nested vault imports to be rejected", err)
	}
}

func TestLoadEffectiveVaultLoadsDeclaredVaultsInOrder(t *testing.T) {
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

	vault, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{first, third, second}
	if got := sourcePaths(vault); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultRejectsDeprecatedVaultDirs(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_dirs = ["docs"]
`)

	_, err := loadEffectiveVault(root)
	if err == nil {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsInvalidDeclaredVaults(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[[vaults]]
vault_name = "missing"
vault_root = "missing"
`)
	if _, err := loadEffectiveVault(root); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing vault error = %v", err)
	}

	writeConfig(t, root, `[[vaults]]
vault_name = ""
vault_root = "."
`)
	if _, err := loadEffectiveVault(root); err == nil || !strings.Contains(err.Error(), "vault_name") {
		t.Fatalf("empty name error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsNoncanonicalVaultName(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "bad name"
vault_root = "."
`)

	_, err := loadEffectiveVault(root)
	if err == nil || !strings.Contains(err.Error(), "canonical gnosis URI authority") {
		t.Fatalf("invalid vault name error = %v", err)
	}
}

func sourcePaths(vault *effectiveVault) []string {
	paths := make([]string, 0, len(vault.sources))
	for _, source := range vault.sources {
		paths = append(paths, source.path)
	}
	return paths
}
