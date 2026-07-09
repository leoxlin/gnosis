package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// LinkFormat is the preferred style for internal markdown links.
type LinkFormat string

const (
	// LinkFormatRelative prefers relative links such as "path/to/file.md".
	LinkFormatRelative LinkFormat = "relative"
	// LinkFormatAbsolute prefers bundle-relative links such as "/path/to/file.md".
	LinkFormatAbsolute LinkFormat = "absolute"
)

// Config is the top-level Gnosis configuration.
type Config struct {
	Vault VaultConfig `toml:"vault"`
}

// VaultConfig holds vault-specific settings.
type VaultConfig struct {
	LinkFormat       string   `toml:"link_format"`
	LinkFormatStrict bool     `toml:"link_format_strict"`
	VaultRoots       []string `toml:"vault_roots"`
}

// DefaultConfig returns the default Gnosis configuration.
func DefaultConfig() Config {
	return Config{
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
			VaultRoots:       nil,
		},
	}
}

// LinkFormatValue returns the configured link format as a typed value.
func (c Config) LinkFormatValue() LinkFormat {
	switch c.Vault.LinkFormat {
	case string(LinkFormatAbsolute):
		return LinkFormatAbsolute
	case string(LinkFormatRelative):
		return LinkFormatRelative
	default:
		return LinkFormatRelative
	}
}

// IsStrict reports whether the configured link format is enforced as an error.
func (c Config) IsStrict() bool {
	return c.Vault.LinkFormatStrict
}

// LoadConfig loads gnosis.toml starting from root and walking up through
// parent directories until the file is found or the filesystem root is reached.
// It returns the parsed configuration and the resolved vault root directories.
// If vault.vault_roots is set in gnosis.toml, each entry is resolved relative
// to the directory containing gnosis.toml. If it is empty or no gnosis.toml is
// found, the original root argument is returned as the single vault root.
func LoadConfig(root string) (Config, []string, error) {
	config := DefaultConfig()
	root = filepath.Clean(root)
	start := root

	for {
		path := filepath.Join(root, "gnosis.toml")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return config, nil, fmt.Errorf("read %s: %w", path, err)
			}
			if err := toml.Unmarshal(data, &config); err != nil {
				return config, nil, fmt.Errorf("parse %s: %w", path, err)
			}
			vaultRoots := make([]string, len(config.Vault.VaultRoots))
			for i, rel := range config.Vault.VaultRoots {
				vaultRoots[i] = filepath.Clean(filepath.Join(root, rel))
			}
			if len(vaultRoots) == 0 {
				vaultRoots = []string{root}
			}
			return config, vaultRoots, nil
		}

		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}

	return config, []string{start}, nil
}
