package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveTargetSelectsLocalAndUserDeclarations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workspace := t.TempDir()
	local := filepath.Join(workspace, "local")
	user := t.TempDir()
	for _, root := range []string{local, user} {
		writeConfig(t, root, `[vault]
vault_name = "`+filepath.Base(root)+`"
vault_root = "."
`)
	}
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "."

[[vaults]]
vault_name = "local"
vault_root = "local"
`)
	writeUserConfig(t, home, `[[vaults]]
vault_name = "user"
vault_root = "`+user+`"
`)

	if got, err := ResolveTarget(workspace, ""); err != nil || got != workspace {
		t.Fatalf("omitted target = %q, %v; want %q", got, err, workspace)
	}
	if got, err := ResolveTarget(workspace, "local"); err != nil || got != local {
		t.Fatalf("local target = %q, %v; want %q", got, err, local)
	}
	if got, err := ResolveTarget(workspace, "user"); err != nil || got != user {
		t.Fatalf("user target = %q, %v; want %q", got, err, user)
	}
}

func TestResolveTargetDeduplicatesNormalizedTargets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workspace := t.TempDir()
	target := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("create symlink: %v", err)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "shared"
vault_root = "`+target+`"

[[vaults]]
vault_name = "remote"
vault_root = "HTTPS://EXAMPLE.TEST/team/../vault.git"
`)
	writeUserConfig(t, home, `[[vaults]]
vault_name = "shared"
vault_root = "`+alias+`"

[[vaults]]
vault_name = "remote"
vault_root = "https://example.test/vault.git"
`)

	if got, err := ResolveTarget(workspace, "shared"); err != nil || got != target {
		t.Fatalf("shared target = %q, %v; want %q", got, err, target)
	}
	if got, err := ResolveTarget(workspace, "remote"); err != nil || got != "https://example.test/vault.git" {
		t.Fatalf("remote target = %q, %v", got, err)
	}
}

func TestResolveTargetRejectsConflictsAndUnknownNames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	workspace := t.TempDir()
	left := t.TempDir()
	right := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "shared"
vault_root = "`+left+`"
`)
	writeUserConfig(t, home, `[[vaults]]
vault_name = "shared"
vault_root = "`+right+`"
`)

	if _, err := ResolveTarget(workspace, "shared"); err == nil ||
		!strings.Contains(err.Error(), "conflicts") ||
		!strings.Contains(err.Error(), "gnosis.toml") {
		t.Fatalf("conflict error = %v", err)
	}

	writeUserConfig(t, home, "")
	if _, err := ResolveTarget(workspace, "missing"); err == nil ||
		!strings.Contains(err.Error(), "configured vaults: shared") {
		t.Fatalf("unknown error = %v", err)
	}
	if _, err := ResolveTarget(workspace, "bad/name"); err == nil ||
		!strings.Contains(err.Error(), "canonical vault name") {
		t.Fatalf("noncanonical error = %v", err)
	}
}

func TestResolveTargetRejectsUnconfiguredInvocation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, err := ResolveTarget(t.TempDir(), ""); err == nil ||
		!strings.Contains(err.Error(), "no local vault is configured") {
		t.Fatalf("unconfigured error = %v", err)
	}
}

func writeUserConfig(t *testing.T, home, content string) {
	t.Helper()
	configDir := filepath.Join(home, ".config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, configDir, content)
}
