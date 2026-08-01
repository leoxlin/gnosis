package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEffectiveVaultPreparesNormalizedRemoteOncePerLoad(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/prepared-once.git")
	configureTestRemoteVault(t, fixture)
	if _, err := resolveVaultTarget(fixture.url); err != nil {
		t.Fatal(err)
	}

	workspace := t.TempDir()
	writeConfig(t, workspace, `[[vaults]]
vault_name = "first"
vault_root = "https://EXAMPLE.TEST/team/prepared-once.git"

[[vaults]]
vault_name = "second"
vault_root = "https://example.test/team/prepared-once.git"
`)
	trace := filepath.Join(t.TempDir(), "git-trace.json")
	t.Setenv("GIT_TRACE2_EVENT", trace)

	first, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.sources) != 1 {
		t.Fatalf("first load sources = %d, want 1", len(first.sources))
	}
	if got := gitPullCount(t, trace); got != 1 {
		t.Fatalf("first load pulls = %d, want 1", got)
	}

	writeTestFile(t, filepath.Join(fixture.seed, "note.md"), "refreshed\n")
	runGit(t, "-C", fixture.seed, "add", "note.md")
	runGit(t, "-C", fixture.seed, "commit", "-m", "refresh remote")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	second, err := loadEffectiveVault(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.sources) != 1 {
		t.Fatalf("second load sources = %d, want 1", len(second.sources))
	}
	if got := gitPullCount(t, trace); got != 2 {
		t.Fatalf("two loads pull count = %d, want 2", got)
	}
	if got := readTestFile(t, filepath.Join(second.sources[0].path, "note.md")); got != "refreshed\n" {
		t.Fatalf("refreshed note = %q", got)
	}
}

func gitPullCount(t *testing.T, trace string) int {
	t.Helper()
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, `"event":"start"`) && strings.Contains(line, `"pull","--ff-only"`) {
			count++
		}
	}
	return count
}
