package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

type commandRemoteFixture struct {
	url    string
	remote string
	seed   string
}

func newCommandRemoteFixture(t *testing.T, remoteURL string) commandRemoteFixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
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
	runCommandGit(t, "config", "--global", "user.name", "gnosis test")
	runCommandGit(t, "config", "--global", "user.email", "gnosis@example.test")
	runCommandGit(t, "config", "--global", "url.file://"+filepath.ToSlash(remote)+".insteadOf", remoteURL)
	runCommandGit(t, "init", "--initial-branch=main", seed)
	writeCommandFile(t, seed, "gnosis.toml", `[vault]
vault_name = "remote"
vault_root = "."
vault_index = false
vault_log = false
`)
	concepts, err := vault.BundledConcepts()
	if err != nil {
		t.Fatal(err)
	}
	for _, concept := range concepts {
		writeCommandFile(t, seed, concept.Path, string(concept.Data))
	}
	writeCommandFile(t, seed, "notes/remote.md", `---
type: Note
title: Remote note
description: A page stored in the remote vault.
---

# Remote note
`)
	runCommandGit(t, "-C", seed, "add", ".")
	runCommandGit(t, "-C", seed, "commit", "-m", "initial remote vault")
	runCommandGit(t, "clone", "--bare", seed, remote)
	registerCommandTarget(t, "remote", remoteURL)
	return commandRemoteFixture{url: remoteURL, remote: remote, seed: seed}
}

func commandRemoteCommitCount(t *testing.T, fixture commandRemoteFixture) int {
	t.Helper()
	count, err := strconv.Atoi(runCommandGit(t, "--git-dir", fixture.remote, "rev-list", "--count", "HEAD"))
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func rejectCommandRemotePushes(t *testing.T, fixture commandRemoteFixture) {
	t.Helper()
	hook := filepath.Join(fixture.remote, "hooks", "pre-receive")
	writeCommandFile(t, fixture.remote, "hooks/pre-receive", "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		t.Fatal(err)
	}
}

func updateCommandRemoteNote(t *testing.T, fixture commandRemoteFixture, title string) {
	t.Helper()
	writeCommandFile(t, fixture.seed, "notes/remote.md", `---
type: Note
title: `+title+`
description: A page stored in the remote vault.
---

# `+title+`
`)
	runCommandGit(t, "-C", fixture.seed, "add", "notes/remote.md")
	runCommandGit(t, "-C", fixture.seed, "commit", "-m", "update remote note")
	runCommandGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
}

func runCommandGit(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}
