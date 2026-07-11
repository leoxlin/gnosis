package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// LinkFormat is the preferred style for internal markdown links.
type LinkFormat string

const (
	LinkFormatRelative LinkFormat = "relative"
	LinkFormatAbsolute LinkFormat = "absolute"
)

// Config is the gnosis configuration for one vault or import-only workspace.
type Config struct {
	Vault VaultConfig `toml:"vault"`
}

// VaultConfig holds the local vault settings and its imports. A configuration
// with no name or directories is an import-only workspace.
type VaultConfig struct {
	Name             string       `toml:"vault_name"`
	Dirs             []string     `toml:"vault_dirs"`
	Imports          VaultImports `toml:"imports"`
	LinkFormat       string       `toml:"link_format"`
	LinkFormatStrict bool         `toml:"link_format_strict"`
	VaultIndex       bool         `toml:"vault_index"`
	VaultLog         bool         `toml:"vault_log"`
}

// VaultImports lists local paths or future remote URLs for other vaults.
type VaultImports struct {
	Vaults []string `toml:"vaults"`
}

// VaultSource is one directory read as part of an ordered composed vault.
// VaultRoot is the configuration directory used to derive its document IDs.
type VaultSource struct {
	Path      string
	VaultRoot string
	Config    Config
}

// ConfigResolution records the workspace configuration and its ordered sources.
type ConfigResolution struct {
	Config          Config
	Root            string
	VaultRoots      []string
	LocalVaultRoots []string
	Sources         []VaultSource
}

// DefaultConfig returns the defaults applied to a local vault configuration.
func DefaultConfig() Config {
	return Config{Vault: VaultConfig{
		LinkFormat:       string(LinkFormatRelative),
		LinkFormatStrict: false,
		VaultIndex:       true,
		VaultLog:         true,
	}}
}

func (c Config) LinkFormatValue() LinkFormat {
	if c.Vault.LinkFormat == string(LinkFormatAbsolute) {
		return LinkFormatAbsolute
	}
	return LinkFormatRelative
}

func (c Config) IsStrict() bool     { return c.Vault.LinkFormatStrict }
func (c Config) IndexEnabled() bool { return c.Vault.VaultIndex }
func (c Config) LogEnabled() bool   { return c.Vault.VaultLog }
func (c Config) HasLocalVault() bool {
	return strings.TrimSpace(c.Vault.Name) != "" || len(c.Vault.Dirs) > 0
}

// LoadConfig resolves every local directory that is readable from root.
func LoadConfig(root string) (Config, []string, error) {
	resolution, err := ResolveConfig(root)
	if err != nil {
		return resolution.Config, nil, err
	}
	return resolution.Config, resolution.VaultRoots, nil
}

// ResolveConfig finds the nearest gnosis.toml and resolves its local vault
// directories followed by recursive imports in declared order.
func ResolveConfig(root string) (ConfigResolution, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return ConfigResolution{}, err
	}
	start := filepath.Clean(absolute)
	configRoot, err := findConfigRoot(start)
	if err != nil {
		return ConfigResolution{Root: start}, err
	}

	config, err := loadConfig(configRoot)
	if err != nil {
		return ConfigResolution{Root: configRoot}, err
	}
	resolution := ConfigResolution{Config: config, Root: configRoot}
	seen := make(map[string]struct{})
	stack := make(map[string]struct{})
	if err := resolveVault(configRoot, &resolution, seen, stack); err != nil {
		return resolution, err
	}
	if len(resolution.Sources) == 0 {
		return resolution, fmt.Errorf("%s does not define local vault directories or imports", filepath.Join(configRoot, "gnosis.toml"))
	}
	return resolution, nil
}

func findConfigRoot(root string) (string, error) {
	for {
		path := filepath.Join(root, "gnosis.toml")
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return root, nil
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", fmt.Errorf("no gnosis.toml found from %s", root)
		}
		root = parent
	}
}

func loadConfig(root string) (Config, error) {
	config := DefaultConfig()
	path := filepath.Join(root, "gnosis.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("read %s: %w", path, err)
	}
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := validateConfig(config, root); err != nil {
		return config, fmt.Errorf("validate %s: %w", path, err)
	}
	return config, nil
}

func resolveVault(root string, resolution *ConfigResolution, seen, stack map[string]struct{}) error {
	root = filepath.Clean(root)
	if _, exists := stack[root]; exists {
		return fmt.Errorf("vault import cycle includes %s", root)
	}
	if _, exists := seen[root]; exists {
		return nil
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	stack[root] = struct{}{}
	defer delete(stack, root)
	seen[root] = struct{}{}

	dirs, err := resolveVaultDirs(config, root)
	if err != nil {
		return fmt.Errorf("validate %s: %w", filepath.Join(root, "gnosis.toml"), err)
	}
	if root == resolution.Root {
		resolution.LocalVaultRoots = append(resolution.LocalVaultRoots, dirs...)
	}
	for _, dir := range dirs {
		resolution.VaultRoots = append(resolution.VaultRoots, dir)
		resolution.Sources = append(resolution.Sources, VaultSource{Path: dir, VaultRoot: root, Config: config})
	}

	for i, importRef := range config.Vault.Imports.Vaults {
		importRoot, err := resolveImport(root, importRef)
		if err != nil {
			return fmt.Errorf("vault.imports.vaults[%d]: %w", i, err)
		}
		if err := resolveVault(importRoot, resolution, seen, stack); err != nil {
			return err
		}
	}
	return nil
}

func validateConfig(config Config, root string) error {
	if !config.HasLocalVault() && len(config.Vault.Imports.Vaults) == 0 {
		return fmt.Errorf("[vault] must define vault_name and vault_dirs, or vault.imports must list a vault")
	}
	if config.HasLocalVault() {
		if strings.TrimSpace(config.Vault.Name) == "" {
			return fmt.Errorf("vault.vault_name must not be empty")
		}
		if len(config.Vault.Dirs) == 0 {
			return fmt.Errorf("vault.vault_dirs must not be empty")
		}
		if _, err := resolveVaultDirs(config, root); err != nil {
			return err
		}
		switch config.Vault.LinkFormat {
		case string(LinkFormatRelative), string(LinkFormatAbsolute):
		default:
			return fmt.Errorf("vault.link_format must be %q or %q, got %q", LinkFormatRelative, LinkFormatAbsolute, config.Vault.LinkFormat)
		}
	}
	for i, value := range config.Vault.Imports.Vaults {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("vault.imports.vaults[%d] must not be empty", i)
		}
	}
	return nil
}

func resolveVaultDirs(config Config, root string) ([]string, error) {
	dirs := make([]string, 0, len(config.Vault.Dirs))
	seen := make(map[string]struct{}, len(config.Vault.Dirs))
	for i, rel := range config.Vault.Dirs {
		if strings.TrimSpace(rel) == "" {
			return nil, fmt.Errorf("vault.vault_dirs[%d] must not be empty", i)
		}
		if filepath.IsAbs(rel) {
			return nil, fmt.Errorf("vault.vault_dirs[%d] must be relative: %q", i, rel)
		}
		resolved := filepath.Clean(filepath.Join(root, rel))
		fromRoot, err := filepath.Rel(root, resolved)
		if err != nil {
			return nil, fmt.Errorf("resolve vault.vault_dirs[%d] %q: %w", i, rel, err)
		}
		if fromRoot == ".." || strings.HasPrefix(fromRoot, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("vault.vault_dirs[%d] escapes the configuration directory: %q", i, rel)
		}
		if _, exists := seen[resolved]; exists {
			return nil, fmt.Errorf("vault.vault_dirs[%d] duplicates another directory: %q", i, rel)
		}
		seen[resolved] = struct{}{}
		dirs = append(dirs, resolved)
	}
	return dirs, nil
}

func resolveImport(root, value string) (string, error) {
	if strings.Contains(value, "://") {
		return "", fmt.Errorf("remote vault imports are not supported yet: %q", value)
	}
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", path)
	}
	if _, err := os.Stat(filepath.Join(path, "gnosis.toml")); err != nil {
		return "", fmt.Errorf("%s does not contain gnosis.toml", path)
	}
	return filepath.Clean(path), nil
}

// WriteWorkspaceConfig creates an import-only workspace configuration.
func WriteWorkspaceConfig(root string, imports []string, force bool) (bool, error) {
	var contents strings.Builder
	contents.WriteString("[vault.imports]\n")
	contents.WriteString("vaults = [")
	for i, value := range imports {
		if i > 0 {
			contents.WriteString(", ")
		}
		contents.WriteString(strconv.Quote(value))
	}
	contents.WriteString("]\n")
	return WriteGeneratedFile(filepath.Join(root, "gnosis.toml"), []byte(contents.String()), force)
}

func writeVaultConfig(root, name string, disableIndex, disableLog, force bool) (bool, error) {
	if strings.TrimSpace(name) == "" {
		absolute, err := filepath.Abs(root)
		if err != nil {
			return false, err
		}
		name = filepath.Base(absolute)
	}
	contents := fmt.Sprintf(`[vault]
vault_name = %s
vault_dirs = ["."]
link_format = "relative"
link_format_strict = false
vault_index = %t
vault_log = %t
`, strconv.Quote(name), !disableIndex, !disableLog)
	return WriteGeneratedFile(filepath.Join(root, "gnosis.toml"), []byte(contents), force)
}
