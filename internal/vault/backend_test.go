package vault

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseRemoteLocator(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		want       string
		wantRemote bool
		wantErr    string
	}{
		{name: "local", value: "../vault", wantRemote: false},
		{
			name: "https", value: "https://GitHub.COM/example/knowledge-vault.git",
			want: "https://github.com/example/knowledge-vault.git", wantRemote: true,
		},
		{
			name: "ssh", value: "ssh://git@GitHub.COM/example/knowledge-vault.git",
			want: "ssh://git@github.com/example/knowledge-vault.git", wantRemote: true,
		},
		{name: "unsupported", value: "ftp://example.com/vault.git", wantRemote: true, wantErr: "https or ssh"},
		{name: "https user", value: "https://token@example.com/vault.git", wantRemote: true, wantErr: "user information"},
		{name: "ssh password", value: "ssh://git:secret@example.com/vault.git", wantRemote: true, wantErr: "password"},
		{name: "query", value: "https://example.com/vault.git?branch=main", wantRemote: true, wantErr: "query or fragment"},
		{name: "fragment", value: "https://example.com/vault.git#main", wantRemote: true, wantErr: "query or fragment"},
		{name: "missing path", value: "https://example.com/", wantRemote: true, wantErr: "repository path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, remote, err := parseRemoteLocator(test.value)
			if remote != test.wantRemote {
				t.Fatalf("remote = %t, want %t", remote, test.wantRemote)
			}
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("locator = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRemoteCacheRootsAreDeterministicAndIsolated(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	first, err := remoteCacheRoot("https://one.example/vault.git")
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := remoteCacheRoot("https://one.example/vault.git")
	if err != nil {
		t.Fatal(err)
	}
	second, err := remoteCacheRoot("https://two.example/vault.git")
	if err != nil {
		t.Fatal(err)
	}
	if first != repeated {
		t.Fatalf("repeated cache root = %q, want %q", repeated, first)
	}
	if first == second {
		t.Fatalf("distinct remotes share cache root %q", first)
	}
	if len(filepath.Base(first)) != 64 {
		t.Fatalf("cache digest = %q, want 64 hex characters", filepath.Base(first))
	}
}

func TestRemoteGitBackendClonesReusesAndPulls(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/vault.git")

	first, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, filepath.Join(first.root, "note.md")); !strings.Contains(got, "first") {
		t.Fatalf("initial note = %q", got)
	}
	repeated, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.root != first.root {
		t.Fatalf("repeated root = %q, want %q", repeated.root, first.root)
	}

	writeTestFile(t, filepath.Join(fixture.seed, "note.md"), "second\n")
	runGit(t, "-C", fixture.seed, "add", "note.md")
	runGit(t, "-C", fixture.seed, "commit", "-m", "advance remote")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
	refreshed, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, filepath.Join(refreshed.root, "note.md")); got != "second\n" {
		t.Fatalf("refreshed note = %q", got)
	}
}

func TestRemoteGitBackendRejectsChangedOrigin(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/vault.git")
	target, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", target.root, "remote", "set-url", "origin", "https://other.example/vault.git")
	if _, err := resolveVaultTarget(fixture.url); err == nil || !strings.Contains(err.Error(), "cache origin") {
		t.Fatalf("origin error = %v", err)
	}
}

func TestRemoteGitBackendCleansFailedClone(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", cache)
	url := "https://example.test/missing/vault.git"
	missing := filepath.Join(root, "missing.git")
	runGit(t, "config", "--global", "url.file://"+filepath.ToSlash(missing)+".insteadOf", url)

	if _, err := resolveVaultTarget(url); err == nil {
		t.Fatal("missing remote clone succeeded")
	}
	cacheRoot, err := remoteCacheRoot(url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheRoot); !os.IsNotExist(err) {
		t.Fatalf("failed cache stat = %v, want not found", err)
	}
	temporary, err := filepath.Glob(filepath.Join(filepath.Dir(cacheRoot), ".clone-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporary) != 0 {
		t.Fatalf("failed clone left temporary directories: %v", temporary)
	}
}

func TestRemoteGitBackendPreservesDivergedHistory(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/vault.git")
	target, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(target.root, "local.md"), "local\n")
	runGit(t, "-C", target.root, "add", "local.md")
	runGit(t, "-C", target.root, "commit", "-m", "local commit")
	writeTestFile(t, filepath.Join(fixture.seed, "remote.md"), "remote\n")
	runGit(t, "-C", fixture.seed, "add", "remote.md")
	runGit(t, "-C", fixture.seed, "commit", "-m", "remote commit")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	if _, err := resolveVaultTarget(fixture.url); err == nil || !strings.Contains(err.Error(), "pull remote vault") {
		t.Fatalf("divergence error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(target.root, "local.md")); err != nil {
		t.Fatalf("local commit was not preserved: %v", err)
	}
	if got := runGitOutput(t, "--git-dir", fixture.remote, "show", "main:remote.md"); got != "remote" {
		t.Fatalf("remote commit = %q", got)
	}
}

func TestRemoteGitBackendPublishesDirectPageAndIndexMutations(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/writable.git")
	configureTestRemoteVault(t, fixture)
	content := []byte(`---
type: Note
title: Added
description: Written through a remote target.
---

# Added
`)
	before := remoteCommitCount(t, fixture)
	if _, err := WriteDocument(fixture.url, "gnosis://remote/notes/added.md", content, false); err != nil {
		t.Fatal(err)
	}
	if got := remoteCommitCount(t, fixture); got != before+1 {
		t.Fatalf("page mutation commit count = %d, want %d", got, before+1)
	}
	if got := runGitOutput(t, "--git-dir", fixture.remote, "show", "main:notes/added.md"); got != strings.TrimSpace(string(content)) {
		t.Fatalf("published page = %q", got)
	}

	rejectTestPushes(t, fixture.remote)
	if _, err := WriteDocument(fixture.url, "gnosis://remote/notes/added.md", content, true); err != nil {
		t.Fatalf("no-op page mutation attempted publication: %v", err)
	}
	if got := remoteCommitCount(t, fixture); got != before+1 {
		t.Fatalf("no-op page commit count = %d, want %d", got, before+1)
	}

	removeTestPushRejection(t, fixture.remote)
	written, enabled, err := GenerateWorkspaceIndexes(fixture.url, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || len(written) == 0 {
		t.Fatalf("index result = (%v, %t), want generated indexes", written, enabled)
	}
	if got := remoteCommitCount(t, fixture); got != before+2 {
		t.Fatalf("index mutation commit count = %d, want %d", got, before+2)
	}

	rejectTestPushes(t, fixture.remote)
	if _, _, err := GenerateWorkspaceIndexes(fixture.url, IndexOptions{Overwrite: true}); err != nil {
		t.Fatalf("no-op index mutation attempted publication: %v", err)
	}
	if got := remoteCommitCount(t, fixture); got != before+2 {
		t.Fatalf("no-op index commit count = %d, want %d", got, before+2)
	}
}

func TestRemoteGitBackendPreservesCachedCommitAfterPushFailure(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/rejected.git")
	configureTestRemoteVault(t, fixture)
	target, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	beforeLocal := localCommitCount(t, target.root)
	beforeRemote := remoteCommitCount(t, fixture)
	rejectTestPushes(t, fixture.remote)
	content := []byte(`---
type: Note
title: Rejected
description: Preserved in the local cache.
---

# Rejected
`)
	if _, err := WriteDocument(fixture.url, "gnosis://remote/notes/rejected.md", content, false); err == nil ||
		!strings.Contains(err.Error(), "publish backend") {
		t.Fatalf("push failure = %v", err)
	}
	if got := localCommitCount(t, target.root); got != beforeLocal+1 {
		t.Fatalf("cached commit count = %d, want %d", got, beforeLocal+1)
	}
	if got := remoteCommitCount(t, fixture); got != beforeRemote {
		t.Fatalf("remote commit count = %d, want %d", got, beforeRemote)
	}
	if got := runGitOutput(t, "-C", target.root, "status", "--porcelain"); got != "" {
		t.Fatalf("cached worktree status = %q, want clean committed state", got)
	}
	if got := readTestFile(t, filepath.Join(target.root, "notes", "rejected.md")); got != string(content) {
		t.Fatalf("cached page = %q", got)
	}
}

type gitRemoteFixture struct {
	url    string
	remote string
	seed   string
}

func newGitRemoteFixture(t *testing.T, remoteURL string) gitRemoteFixture {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	home := filepath.Join(root, "home")
	cache := filepath.Join(root, "cache")
	seed := filepath.Join(root, "seed")
	remote := filepath.Join(root, "vault.git")
	for _, current := range []string{home, seed} {
		if err := os.MkdirAll(current, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", cache)
	runGit(t, "config", "--global", "user.name", "gnosis test")
	runGit(t, "config", "--global", "user.email", "gnosis@example.test")
	runGit(t, "config", "--global", "url.file://"+filepath.ToSlash(remote)+".insteadOf", remoteURL)
	runGit(t, "init", "--initial-branch=main", seed)
	writeTestFile(t, filepath.Join(seed, "note.md"), "first\n")
	runGit(t, "-C", seed, "add", ".")
	runGit(t, "-C", seed, "commit", "-m", "initial")
	runGit(t, "clone", "--bare", seed, remote)
	return gitRemoteFixture{url: remoteURL, remote: remote, seed: seed}
}

func configureTestRemoteVault(t *testing.T, fixture gitRemoteFixture) {
	t.Helper()
	writeConfig(t, fixture.seed, `[vault]
vault_name = "remote"
vault_root = "."
vault_index = true
vault_log = false
`)
	writeTestFile(t, filepath.Join(fixture.seed, "note.md"), `---
type: Note
title: Existing
description: Existing remote page.
---
`)
	writeTestFile(t, filepath.Join(fixture.seed, "concepts", "note.md"), `---
type: ConceptType
title: Note
description: A short general-purpose record.
path: notes
---

# Note
`)
	runGit(t, "-C", fixture.seed, "add", ".")
	runGit(t, "-C", fixture.seed, "commit", "-m", "configure remote vault")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
}

func localCommitCount(t *testing.T, root string) int {
	t.Helper()
	count, err := strconv.Atoi(runGitOutput(t, "-C", root, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func remoteCommitCount(t *testing.T, fixture gitRemoteFixture) int {
	t.Helper()
	count, err := strconv.Atoi(runGitOutput(t, "--git-dir", fixture.remote, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func rejectTestPushes(t *testing.T, remote string) {
	t.Helper()
	hook := filepath.Join(remote, "hooks", "pre-receive")
	writeTestFile(t, hook, "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
}

func removeTestPushRejection(t *testing.T, remote string) {
	t.Helper()
	if err := os.Remove(filepath.Join(remote, "hooks", "pre-receive")); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

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
