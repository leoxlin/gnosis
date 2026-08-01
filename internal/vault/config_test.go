package vault

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestLoadEffectiveVaultRejectsMissingConfiguration(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()

	if _, err := loadEffectiveVault(root); err == nil || !strings.Contains(err.Error(), "no gnosis configuration") {
		t.Fatalf("missing configuration error = %v", err)
	}
}

func TestGitHubRepositoryConfigIsScopedAndBounded(t *testing.T) {
	root := t.TempDir()
	evidenceDir := filepath.Join(root, "evidence")
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."

[[github]]
repository = "Owner/Repo"
evidence_dir = "`+evidenceDir+`"
token_env = "GITHUB_TOKEN"
webhook_secret_env = "GITHUB_WEBHOOK_SECRET"
`)
	name, config, err := GitHubRepositoryConfig(root, "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	if name != "test" || config.Repository != "owner/repo" ||
		config.PerPage != DefaultGitHubPerPage ||
		config.MaxPages != DefaultGitHubMaxPages ||
		config.MaxBodyBytes != DefaultGitHubMaxBodyBytes {
		t.Fatalf("config = %q, %+v", name, config)
	}
	if _, _, err := GitHubRepositoryConfig(root, "other/repo"); err == nil ||
		!strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unknown repository error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsRemovedProcessesConfig(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Local"
vault_root = "."

[gnosis]
processes = ["vault", "planning"]
`)

	_, err := loadEffectiveVault(root)
	if err == nil || !strings.Contains(err.Error(), "gnosis") {
		t.Fatalf("removed processes config error = %v", err)
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

func TestLoadEffectiveVaultFindsRepositoryConfigurationFromDescendant(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	repository := t.TempDir()
	write(t, repository, ".git/HEAD", "ref: refs/heads/main\n")
	writeConfig(t, repository, `[vault]
vault_name = "repository"
vault_root = "knowledge"
vault_index = false
vault_log = false
`)
	write(t, repository, "knowledge/note.md", "---\ntype: Note\ntitle: Repository note\n---\n")
	descendant := filepath.Join(repository, "docs", "knowledge", "records")
	if err := os.MkdirAll(descendant, 0o755); err != nil {
		t.Fatal(err)
	}

	vault, err := loadEffectiveVault(descendant)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := vault.root, repository; got != want {
		t.Fatalf("vault root = %q, want %q", got, want)
	}
	if got, want := sourcePaths(vault), []string{filepath.Join(repository, "knowledge")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
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

func TestLoadConfigValidatesRemoteImportWithoutCloning(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "remote"
vault_root = "https://example.com/remote-vault.git"
`)

	if _, err := loadConfig(workspace); err != nil {
		t.Fatal(err)
	}
	cache, err := remoteCacheRoot("https://example.com/remote-vault.git")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cache); !os.IsNotExist(err) {
		t.Fatalf("configuration validation created cache: %v", err)
	}
}

func TestLoadEffectiveVaultLoadsRemoteImportOnce(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/imported.git")
	writeConfig(t, fixture.seed, `[vault]
vault_name = "remote"
vault_root = "."
vault_index = false
vault_log = false
`)
	runGit(t, "-C", fixture.seed, "add", "gnosis.toml")
	runGit(t, "-C", fixture.seed, "commit", "-m", "configure vault")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "first"
vault_root = "`+fixture.url+`"

[[vaults]]
vault_name = "second"
vault_root = "`+fixture.url+`"
`)
	effective, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(effective.sources) != 1 {
		t.Fatalf("sources = %v, want one de-duplicated remote", sourcePaths(effective))
	}
	if effective.sources[0].config.Vault.Name != "remote" {
		t.Fatalf("remote source = %+v", effective.sources[0])
	}
	if effective.backend != nil {
		t.Fatal("remote import unexpectedly became the writable publisher")
	}
	before := remoteCommitCount(t, fixture)
	content := []byte(`---
type: Note
title: Imported write
description: Must not be published through an import.
---
`)
	if _, err := WriteDocument(context.Background(), workspace, "gnosis://remote/notes/imported.md", content, false); err == nil ||
		!strings.Contains(err.Error(), "local vault") {
		t.Fatalf("remote import write error = %v", err)
	}
	if got := remoteCommitCount(t, fixture); got != before {
		t.Fatalf("remote import commit count = %d, want %d", got, before)
	}
}

func TestLoadEffectiveVaultKeepsLocalPrecedenceOverRemoteImport(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/precedence.git")
	writeConfig(t, fixture.seed, `[vault]
vault_name = "remote"
vault_root = "."
vault_index = false
vault_log = false
`)
	writeTestFile(t, filepath.Join(fixture.seed, "note.md"), `---
type: Resource
title: Remote note
description: Lower-precedence remote content.
---
`)
	runGit(t, "-C", fixture.seed, "add", ".")
	runGit(t, "-C", fixture.seed, "commit", "-m", "configure remote vault")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	workspace := t.TempDir()
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"
vault_index = false
vault_log = false

[[vaults]]
vault_name = "remote"
vault_root = "`+fixture.url+`"
`)
	writeTestFile(t, filepath.Join(workspace, "local", "note.md"), `---
type: Resource
title: Local note
description: Higher-precedence local content.
---
`)

	page, err := ReadPage(workspace, "gnosis://workspace/note.md")
	if err != nil {
		t.Fatal(err)
	}
	if page.Document.Title != "Local note" || page.Document.Origin.Kind != OriginLocal {
		t.Fatalf("resolved page = %+v", page.Document)
	}
}

func TestLoadEffectiveVaultSupportsConfiguredRemoteTarget(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/configured.git")
	writeConfig(t, fixture.seed, `[vault]
vault_name = "remote"
vault_root = "."
vault_index = false
vault_log = false
`)
	runGit(t, "-C", fixture.seed, "add", "gnosis.toml")
	runGit(t, "-C", fixture.seed, "commit", "-m", "configure vault")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	effective, err := loadEffectiveVault(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	if effective.backend == nil || effective.config.Vault.Name != "remote" {
		t.Fatalf("effective remote = %+v", effective)
	}
	if got, want := sourcePaths(effective), []string{effective.root}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sources = %v, want %v", got, want)
	}
}

func TestLoadEffectiveVaultRejectsRemoteImportWithoutConfiguration(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/unconfigured.git")
	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "remote"
vault_root = "`+fixture.url+`"
`)

	_, err := loadEffectiveVault(workspace)
	if err == nil || !strings.Contains(err.Error(), "gnosis.toml") {
		t.Fatalf("missing remote configuration error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsRemoteImportFailure(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	url := "https://example.test/missing/vault.git"
	runGit(t, "config", "--global", "url.file://"+filepath.ToSlash(filepath.Join(root, "missing.git"))+".insteadOf", url)
	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "remote"
vault_root = "`+url+`"
`)

	if _, err := loadEffectiveVault(workspace); err == nil || !strings.Contains(err.Error(), "clone remote vault") {
		t.Fatalf("remote import error = %v", err)
	}
}

func TestLoadEffectiveVaultRejectsLocalRemoteCycle(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/cycle.git")
	workspace := t.TempDir()
	writeConfig(t, fixture.seed, `[vault]
vault_name = "remote"
vault_root = "."

[[vaults]]
vault_name = "workspace"
vault_root = "`+workspace+`"
`)
	runGit(t, "-C", fixture.seed, "add", "gnosis.toml")
	runGit(t, "-C", fixture.seed, "commit", "-m", "add cycle")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
	writeConfig(t, workspace, `[[vaults]]
vault_name = "remote"
vault_root = "`+fixture.url+`"
`)

	if _, err := loadEffectiveVault(workspace); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("remote cycle error = %v", err)
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

	if _, err := loadEffectiveVault(root); err == nil || !strings.Contains(err.Error(), "no gnosis configuration") {
		t.Fatalf("parent configuration error = %v", err)
	}
}

func TestLoadEffectiveVaultInheritsGlobalVaultsFromLocalConfiguration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	configDir := filepath.Join(home, ".config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	imported := t.TempDir()
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "."
`)
	writeConfig(t, imported, `[vault]
vault_name = "imported"
vault_root = "."
`)
	writeConfig(t, configDir, `[[vaults]]
vault_name = "workspace"
vault_root = "`+workspace+`"

[[vaults]]
vault_name = "imported"
vault_root = "`+imported+`"
`)

	vault, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourcePaths(vault), []string{workspace, imported}; !reflect.DeepEqual(got, want) {
		t.Fatalf("vault roots = %v, want %v", got, want)
	}

	unregistered := t.TempDir()
	writeConfig(t, unregistered, `[vault]
vault_name = "unregistered"
vault_root = "."
`)
	vault, err = loadEffectiveVault(unregistered)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := sourcePaths(vault), []string{unregistered}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unregistered vault roots = %v, want %v", got, want)
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
	for _, name := range []string{"bad name", "_"} {
		root := t.TempDir()
		writeConfig(t, root, "[vault]\nvault_name = "+strconv.Quote(name)+"\nvault_root = \".\"\n")

		_, err := loadEffectiveVault(root)
		if err == nil || !strings.Contains(err.Error(), "canonical gnosis URI authority") {
			t.Fatalf("vault name %q error = %v", name, err)
		}
	}
}

func sourcePaths(vault *effectiveVault) []string {
	paths := make([]string, 0, len(vault.sources))
	for _, source := range vault.sources {
		paths = append(paths, source.path)
	}
	return paths
}
