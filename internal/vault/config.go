package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// Config is the top-level gnosis configuration.
type Config struct {
	Vault VaultConfig `toml:"vault"`
}

// ConfigResolution records both the configuration directory and the vault
// roots resolved from it. Root is the requested directory when no gnosis.toml
// is found.
type ConfigResolution struct {
	Config     Config
	Root       string
	VaultRoots []string
}

// VaultConfig holds vault-specific settings.
type VaultConfig struct {
	LinkFormat       string   `toml:"link_format"`
	LinkFormatStrict bool     `toml:"link_format_strict"`
	VaultRoots       []string `toml:"vault_roots"`
	VaultIndex       bool     `toml:"vault_index"`
	VaultLog         bool     `toml:"vault_log"`
}

// DefaultConfig returns the default gnosis configuration.
func DefaultConfig() Config {
	return Config{
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
			VaultRoots:       nil,
			VaultIndex:       true,
			VaultLog:         true,
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

// IndexEnabled reports whether directory indexes are maintained for the vault.
func (c Config) IndexEnabled() bool {
	return c.Vault.VaultIndex
}

// LogEnabled reports whether the root activity log is maintained for the vault.
func (c Config) LogEnabled() bool {
	return c.Vault.VaultLog
}

// LoadConfig loads gnosis.toml starting from root and walking up through
// parent directories until the file is found or the filesystem root is reached.
// It returns the parsed configuration and the resolved vault root directories.
// If vault.vault_roots is set in gnosis.toml, each entry is resolved relative
// to the directory containing gnosis.toml. If it is empty or no gnosis.toml is
// found, the original root argument is returned as the single vault root.
func LoadConfig(root string) (Config, []string, error) {
	resolution, err := ResolveConfig(root)
	if err != nil {
		return resolution.Config, nil, err
	}
	return resolution.Config, resolution.VaultRoots, nil
}

// ResolveConfig loads gnosis.toml using the same walk-up behavior as
// LoadConfig while also retaining the directory paths are relative to.
func ResolveConfig(root string) (ConfigResolution, error) {
	config := DefaultConfig()
	root = filepath.Clean(root)
	start := root

	for {
		path := filepath.Join(root, "gnosis.toml")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return ConfigResolution{Config: config, Root: root}, fmt.Errorf("read %s: %w", path, err)
			}
			decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
			if err := decoder.Decode(&config); err != nil {
				return ConfigResolution{Config: config, Root: root}, fmt.Errorf("parse %s: %w", path, err)
			}
			vaultRoots, err := validateConfig(config, root)
			if err != nil {
				return ConfigResolution{Config: config, Root: root}, fmt.Errorf("validate %s: %w", path, err)
			}
			if len(vaultRoots) == 0 {
				vaultRoots = []string{root}
			}
			return ConfigResolution{
				Config:     config,
				Root:       root,
				VaultRoots: vaultRoots,
			}, nil
		}

		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}

	return ConfigResolution{
		Config:     config,
		Root:       start,
		VaultRoots: []string{start},
	}, nil
}

func validateConfig(config Config, configRoot string) ([]string, error) {
	switch config.Vault.LinkFormat {
	case string(LinkFormatRelative), string(LinkFormatAbsolute):
	default:
		return nil, fmt.Errorf("vault.link_format must be %q or %q, got %q", LinkFormatRelative, LinkFormatAbsolute, config.Vault.LinkFormat)
	}

	vaultRoots := make([]string, 0, len(config.Vault.VaultRoots))
	seen := make(map[string]struct{}, len(config.Vault.VaultRoots))
	for i, rel := range config.Vault.VaultRoots {
		if strings.TrimSpace(rel) == "" {
			return nil, fmt.Errorf("vault.vault_roots[%d] must not be empty", i)
		}
		if filepath.IsAbs(rel) {
			return nil, fmt.Errorf("vault.vault_roots[%d] must be relative: %q", i, rel)
		}

		resolved := filepath.Clean(filepath.Join(configRoot, rel))
		fromRoot, err := filepath.Rel(configRoot, resolved)
		if err != nil {
			return nil, fmt.Errorf("resolve vault.vault_roots[%d] %q: %w", i, rel, err)
		}
		if fromRoot == ".." || strings.HasPrefix(fromRoot, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("vault.vault_roots[%d] escapes the configuration directory: %q", i, rel)
		}
		if _, exists := seen[resolved]; exists {
			return nil, fmt.Errorf("vault.vault_roots[%d] duplicates another root: %q", i, rel)
		}

		seen[resolved] = struct{}{}
		vaultRoots = append(vaultRoots, resolved)
	}
	return vaultRoots, nil
}
