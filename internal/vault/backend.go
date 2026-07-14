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

type gitBackend struct {
	root string
}

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
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
			return nil, err
		}
		if err := runGitCommand("clone", remote, root); err != nil {
			return nil, fmt.Errorf("clone GitHub Wiki %q: %w", repository, err)
		}
	} else if err != nil {
		return nil, err
	} else if err := runGitCommand("-C", root, "pull", "--ff-only"); err != nil {
		return nil, fmt.Errorf("pull GitHub Wiki %q: %w", repository, err)
	}
	return &gitBackend{root: root}, nil
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
