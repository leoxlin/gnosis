package vault

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const githubWikiBackend = "github-wiki"

type preparedBackend interface {
	preparedRoot() string
	publish(string) error
}

type gitBackend struct {
	root string
}

func (b *gitBackend) preparedRoot() string { return b.root }

func validateGitHubRepository(repository string) error {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || !validGitHubName(parts[0]) || !validGitHubName(parts[1]) {
		return fmt.Errorf("must be OWNER/REPOSITORY")
	}
	return nil
}

func validGitHubName(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func prepareGitHubWikiBackend(repository string) (*gitBackend, error) {
	if err := validateGitHubRepository(repository); err != nil {
		return nil, fmt.Errorf("GitHub Wiki repository: %w", err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("GitHub Wiki cache: %w", err)
	}
	parts := strings.Split(repository, "/")
	// ponytail: cache access is single-process; add per-vault locking if concurrent CLI use becomes necessary.
	root := filepath.Join(cache, "gnosis", githubWikiBackend, parts[0], parts[1])
	remote := "https://github.com/" + repository + ".wiki.git"
	return prepareGitBackend(remote, root)
}

func prepareGitBackend(remote, root string) (*gitBackend, error) {
	info, err := os.Stat(root)
	if err == nil {
		if !info.IsDir() {
			return nil, fmt.Errorf("remote vault cache %s is not a directory", root)
		}
		if err := verifyGitOrigin(root, remote); err != nil {
			return nil, err
		}
		if err := runGitCommand("-C", root, "pull", "--ff-only"); err != nil {
			return nil, fmt.Errorf("pull remote vault %q: %w", remote, err)
		}
		return &gitBackend{root: root}, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	temporary, err := os.MkdirTemp(parent, ".clone-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	if err := runGitCommand("clone", "--", remote, temporary); err != nil {
		return nil, fmt.Errorf("clone remote vault %q: %w", remote, err)
	}
	if err := verifyGitOrigin(temporary, remote); err != nil {
		return nil, err
	}
	if err := os.Rename(temporary, root); err != nil {
		return nil, fmt.Errorf("install remote vault cache: %w", err)
	}
	return &gitBackend{root: root}, nil
}

func verifyGitOrigin(root, remote string) error {
	origin, err := gitCommandOutput("-C", root, "config", "--get", "remote.origin.url")
	if err != nil {
		return fmt.Errorf("read remote vault origin: %w", err)
	}
	if strings.TrimSpace(origin) != remote {
		return fmt.Errorf(
			"remote vault cache origin is %q, want %q",
			strings.TrimSpace(origin),
			remote,
		)
	}
	return nil
}

func (b *gitBackend) publish(message string) error {
	status, err := gitCommandOutput("-C", b.root, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return nil
	}
	if err := runGitCommand("-C", b.root, "add", "--all"); err != nil {
		return err
	}
	if err := runGitCommand("-C", b.root, "commit", "-m", message); err != nil {
		return err
	}
	return runGitCommand("-C", b.root, "push")
}

func runGitCommand(args ...string) error {
	_, err := gitCommandOutput(args...)
	return err
}

func gitCommandOutput(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
