package vault

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const gitVaultCache = "git-vaults"

type vaultTarget struct {
	root    string
	backend *gitBackend
}

func resolveVaultTarget(value string) (vaultTarget, error) {
	remote, ok, err := parseRemoteLocator(value)
	if err != nil {
		return vaultTarget{}, err
	}
	if !ok {
		return vaultTarget{root: value}, nil
	}
	root, err := remoteCacheRoot(remote)
	if err != nil {
		return vaultTarget{}, err
	}
	backend, err := prepareGitBackend(remote, root)
	if err != nil {
		return vaultTarget{}, err
	}
	return vaultTarget{root: backend.root, backend: backend}, nil
}

func writeTargetFile(root, relative string, content []byte, force bool, message string) (bool, string, error) {
	target, err := resolveVaultTarget(root)
	if err != nil {
		return false, "", err
	}
	if err := os.MkdirAll(target.root, 0o755); err != nil {
		return false, "", err
	}
	destination := filepath.Join(target.root, relative)
	changed, err := WriteGeneratedFile(destination, content, force)
	if err != nil {
		return false, destination, err
	}
	if changed && target.backend != nil {
		if err := target.backend.publish(message); err != nil {
			return true, destination, err
		}
	}
	return changed, destination, nil
}

func parseRemoteLocator(value string) (string, bool, error) {
	if !strings.Contains(value, "://") {
		return "", false, nil
	}
	if value != strings.TrimSpace(value) {
		return "", true, fmt.Errorf("vault target: remote URL must not have surrounding whitespace")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", true, fmt.Errorf("vault target: invalid remote URL: %w", err)
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	switch parsed.Scheme {
	case "https":
		if parsed.User != nil {
			return "", true, fmt.Errorf("vault target: HTTPS URL must not contain user information")
		}
	case "ssh":
		if parsed.User != nil {
			if _, hasPassword := parsed.User.Password(); hasPassword {
				return "", true, fmt.Errorf("vault target: SSH URL must not contain a password")
			}
		}
	default:
		return "", true, fmt.Errorf("vault target: remote URL scheme must be https or ssh")
	}
	if parsed.Host == "" {
		return "", true, fmt.Errorf("vault target: remote URL must contain a host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", true, fmt.Errorf("vault target: remote URL must not contain a query or fragment")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = path.Clean(parsed.Path)
	parsed.RawPath = ""
	if parsed.Path == "." || parsed.Path == "/" {
		return "", true, fmt.Errorf("vault target: remote URL must contain a repository path")
	}
	return parsed.String(), true, nil
}

func remoteCacheRoot(remote string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("remote vault cache: %w", err)
	}
	digest := sha256.Sum256([]byte(remote))
	return filepath.Join(cache, "gnosis", gitVaultCache, hex.EncodeToString(digest[:])), nil
}
