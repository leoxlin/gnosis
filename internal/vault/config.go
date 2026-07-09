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
	LinkFormat       string `toml:"link_format"`
	LinkFormatStrict bool   `toml:"link_format_strict"`
}

// DefaultConfig returns the default Gnosis configuration.
func DefaultConfig() Config {
	return Config{
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
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

// LoadConfig loads gnosis.toml starting from the vault root and walking up
// through parent directories until the file is found or the filesystem root
// is reached. If no gnosis.toml is found, the default configuration is returned.
func LoadConfig(root string) (Config, error) {
	config := DefaultConfig()
	root = filepath.Clean(root)

	for {
		path := filepath.Join(root, "gnosis.toml")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return config, fmt.Errorf("read %s: %w", path, err)
			}
			if err := toml.Unmarshal(data, &config); err != nil {
				return config, fmt.Errorf("parse %s: %w", path, err)
			}
			return config, nil
		}

		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}

	return config, nil
}
