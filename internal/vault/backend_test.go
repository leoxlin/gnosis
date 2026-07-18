package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubWikiBackendPullsAndPublishes(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	home := filepath.Join(root, "home")
	cache := filepath.Join(root, "cache")
	remote := filepath.Join(root, "wiki.git")
	seed := filepath.Join(root, "seed")
	workspace := filepath.Join(root, "workspace")
	for _, path := range []string{home, seed, workspace} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", cache)
	runGit(t, "config", "--global", "user.name", "gnosis test")
	runGit(t, "config", "--global", "user.email", "gnosis@example.test")
	runGit(t, "config", "--global", "url.file://"+filepath.ToSlash(remote)+".insteadOf", "https://github.com/OWNER/REPOSITORY.wiki.git")

	runGit(t, "init", "--initial-branch=main", seed)
	writeTestFile(t, filepath.Join(seed, "Home.md"), "---\ntype: Reference\ntitle: Home\ndescription: first\n---\n\n# Home\n\nfirst\n")
	writeTestFile(t, filepath.Join(seed, "concepts", "note.md"), "---\ntype: ConceptType\ntitle: Note\ndescription: A short general-purpose record.\npath: notes\n---\n\n# Note\n")
	runGit(t, "-C", seed, "add", ".")
	runGit(t, "-C", seed, "commit", "-m", "initial wiki")
	runGit(t, "clone", "--bare", seed, remote)

	writeConfig(t, workspace, `[vault]
vault_name = "wiki"
backend = "github-wiki"
repository = "OWNER/REPOSITORY"
link_format = "relative"
vault_index = true
vault_log = false
`)
	page, err := ReadPage(workspace, "gnosis://wiki/Home.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Markdown, "first") {
		t.Fatalf("initial page = %q, want cloned wiki content", page.Markdown)
	}

	writeTestFile(t, filepath.Join(seed, "Home.md"), strings.ReplaceAll(page.Markdown, "first", "second"))
	runGit(t, "-C", seed, "add", "Home.md")
	runGit(t, "-C", seed, "commit", "-m", "update wiki")
	runGit(t, "-C", seed, "push", remote, "main")
	page, err = ReadPage(workspace, "gnosis://wiki/Home.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Markdown, "second") {
		t.Fatalf("updated page = %q, want pulled wiki content", page.Markdown)
	}

	content := []byte("---\ntype: Note\ntitle: Added\ndescription: written through gnosis\n---\n\n# Added\n\nTest.\n")
	if _, err := WriteDocument(workspace, "gnosis://wiki/notes/added.md", content, false); err != nil {
		t.Fatal(err)
	}
	checkout := filepath.Join(root, "checkout")
	runGit(t, "clone", remote, checkout)
	got, err := os.ReadFile(filepath.Join(checkout, "notes", "added.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("published content = %q, want %q", got, content)
	}

	if _, _, err := GenerateWorkspaceIndexes(workspace, IndexOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	indexed := filepath.Join(root, "indexed")
	runGit(t, "clone", remote, indexed)
	if _, err := os.Stat(filepath.Join(indexed, "index.md")); err != nil {
		t.Fatalf("published index: %v", err)
	}
	before := runGitOutput(t, "--git-dir", remote, "rev-list", "--count", "HEAD")
	if _, _, err := GenerateWorkspaceIndexes(workspace, IndexOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	after := runGitOutput(t, "--git-dir", remote, "rev-list", "--count", "HEAD")
	if before != after {
		t.Fatalf("no-op index changed remote history from %s to %s", before, after)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

func runGit(t *testing.T, args ...string) {
	t.Helper()
	runGitOutput(t, args...)
}

func runGitOutput(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
