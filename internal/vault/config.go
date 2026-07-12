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

// Config is the gnosis configuration for local and declared vaults.
type Config struct {
	Vault  VaultConfig           `toml:"vault"`
	Vaults []DeclaredVaultConfig `toml:"vaults"`
	Gnosis GnosisConfig          `toml:"gnosis"`
}

// GnosisConfig controls which tagged process families are discoverable.
type GnosisConfig struct {
	Processes []string `toml:"processes"`
}

// VaultConfig holds local vault settings.
type VaultConfig struct {
	Name             string `toml:"vault_name"`
	Root             string `toml:"vault_root"`
	LinkFormat       string `toml:"link_format"`
	LinkFormatStrict bool   `toml:"link_format_strict"`
	VaultIndex       bool   `toml:"vault_index"`
	VaultLog         bool   `toml:"vault_log"`
}

// DeclaredVaultConfig identifies one additional vault root in a workspace.
type DeclaredVaultConfig struct {
	Name string `toml:"vault_name"`
	Root string `toml:"vault_root"`
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

// DefaultConfig returns the default gnosis configuration.
func DefaultConfig() Config {
	return Config{
		Gnosis: GnosisConfig{
			Processes: []string{"gnosis-vault"},
		},
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
			VaultIndex:       true,
			VaultLog:         true,
		},
	}
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

// ProcessEnabled reports whether at least one process tag is enabled.
func (c Config) ProcessEnabled(tags []string) bool {
	enabled := make(map[string]struct{}, len(c.Gnosis.Processes))
	for _, tag := range c.Gnosis.Processes {
		enabled[tag] = struct{}{}
	}
	for _, tag := range tags {
		if _, ok := enabled[tag]; ok {
			return true
		}
	}
	return false
}

func (c Config) HasLocalVault() bool {
	return strings.TrimSpace(c.Vault.Name) != "" || strings.TrimSpace(c.Vault.Root) != ""
}

// LoadConfig resolves every vault root readable from root.
func LoadConfig(root string) (Config, []string, error) {
	resolution, err := ResolveConfig(root)
	if err != nil {
		return resolution.Config, nil, err
	}
	return resolution.Config, resolution.VaultRoots, nil
}

// ResolveConfig reads configuration from root in this order:
// gnosis.local.toml, gnosis.toml, then ~/.config/gnosis.toml. It never walks
// parent directories. When none of those files exists, it uses the defaults,
// including the bundled vault documentation.
//
// A selected configuration resolves its local vault root followed by declared
// vault roots in declaration order.
func ResolveConfig(root string) (ConfigResolution, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return ConfigResolution{}, err
	}
	start := filepath.Clean(absolute)
	configPath, err := findConfigPath(start)
	if err != nil {
		return ConfigResolution{Root: start}, err
	}
	if configPath == "" {
		return ConfigResolution{
			Config: DefaultConfig(),
			Root:   start,
		}, nil
	}

	configRoot := filepath.Dir(configPath)
	config, err := loadConfigPath(configPath)
	if err != nil {
		return ConfigResolution{Root: configRoot}, err
	}
	resolution := ConfigResolution{Config: config, Root: configRoot}
	if err := resolveVaultConfig(configRoot, config, &resolution); err != nil {
		return resolution, err
	}
	return resolution, nil
}

func findConfigPath(root string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	for _, path := range []string{
		filepath.Join(root, "gnosis.local.toml"),
		filepath.Join(root, "gnosis.toml"),
		filepath.Join(home, ".config", "gnosis.toml"),
	} {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("stat %s: %w", path, err)
		}
	}
	return "", nil
}

func loadConfig(root string) (Config, error) {
	return loadConfigPath(filepath.Join(root, "gnosis.toml"))
}

func loadConfigPath(path string) (Config, error) {
	config := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("read %s: %w", path, err)
	}
	decoder := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := validateConfig(config, filepath.Dir(path)); err != nil {
		return config, fmt.Errorf("validate %s: %w", path, err)
	}
	return config, nil
}

func resolveVaultConfig(root string, config Config, resolution *ConfigResolution) error {
	root = filepath.Clean(root)

	if config.HasLocalVault() {
		vaultRoot, err := resolveVaultRoot(config, root)
		if err != nil {
			return fmt.Errorf("validate %s: %w", filepath.Join(root, "gnosis.toml"), err)
		}
		if root == resolution.Root {
			resolution.LocalVaultRoots = append(resolution.LocalVaultRoots, vaultRoot)
		}
		resolution.VaultRoots = append(resolution.VaultRoots, vaultRoot)
		resolution.Sources = append(resolution.Sources, VaultSource{Path: vaultRoot, VaultRoot: root, Config: config})
	}

	for i, declared := range config.Vaults {
		vaultRoot, err := resolveDeclaredVaultRoot(declared, root)
		if err != nil {
			return fmt.Errorf("vaults[%d]: %w", i, err)
		}
		sourceConfig := config
		sourceConfig.Vault.Name = declared.Name
		sourceConfig.Vault.Root = declared.Root
		sourceConfig.Vaults = nil
		resolution.VaultRoots = append(resolution.VaultRoots, vaultRoot)
		resolution.Sources = append(resolution.Sources, VaultSource{Path: vaultRoot, VaultRoot: vaultRoot, Config: sourceConfig})
	}
	return nil
}

func validateConfig(config Config, root string) error {
	seenProcessTags := make(map[string]struct{}, len(config.Gnosis.Processes))
	for i, tag := range config.Gnosis.Processes {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			return fmt.Errorf("gnosis.processes[%d] must not be empty", i)
		}
		if _, exists := seenProcessTags[tag]; exists {
			return fmt.Errorf("gnosis.processes[%d] duplicates %q", i, tag)
		}
		seenProcessTags[tag] = struct{}{}
	}
	if config.HasLocalVault() {
		if strings.TrimSpace(config.Vault.Name) == "" {
			return fmt.Errorf("vault.vault_name must not be empty")
		}
		if strings.TrimSpace(config.Vault.Root) == "" {
			return fmt.Errorf("vault.vault_root must not be empty")
		}
		if _, err := resolveVaultRoot(config, root); err != nil {
			return err
		}
		switch config.Vault.LinkFormat {
		case string(LinkFormatRelative), string(LinkFormatAbsolute):
		default:
			return fmt.Errorf("vault.link_format must be %q or %q, got %q", LinkFormatRelative, LinkFormatAbsolute, config.Vault.LinkFormat)
		}
	}
	for i, declared := range config.Vaults {
		if strings.TrimSpace(declared.Name) == "" {
			return fmt.Errorf("vaults[%d].vault_name must not be empty", i)
		}
		if _, err := resolveDeclaredVaultRoot(declared, root); err != nil {
			return fmt.Errorf("vaults[%d]: %w", i, err)
		}
	}
	return nil
}

func resolveVaultRoot(config Config, root string) (string, error) {
	rel := config.Vault.Root
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("vault.vault_root must not be empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("vault.vault_root must be relative: %q", rel)
	}
	resolved := filepath.Clean(filepath.Join(root, rel))
	fromRoot, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", fmt.Errorf("resolve vault.vault_root %q: %w", rel, err)
	}
	if fromRoot == ".." || strings.HasPrefix(fromRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("vault.vault_root escapes the configuration directory: %q", rel)
	}
	return resolved, nil
}

func resolveDeclaredVaultRoot(config DeclaredVaultConfig, root string) (string, error) {
	path := strings.TrimSpace(config.Root)
	if path == "" {
		return "", fmt.Errorf("vault_root must not be empty")
	}
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
	return filepath.Clean(path), nil
}

// WriteWorkspaceConfig creates a workspace configuration with declared vaults.
func WriteWorkspaceConfig(root string, imports []string, force bool) (bool, error) {
	var contents strings.Builder
	for _, value := range imports {
		contents.WriteString("[[vaults]]\n")
		contents.WriteString("vault_name = ")
		contents.WriteString(strconv.Quote(filepath.Base(filepath.Clean(value))))
		contents.WriteString("\n")
		contents.WriteString("vault_root = ")
		contents.WriteString(strconv.Quote(value))
		contents.WriteString("\n\n")
	}
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
vault_root = "."
link_format = "relative"
link_format_strict = false
vault_index = %t
vault_log = %t

[gnosis]
processes = ["gnosis-vault"]
`, strconv.Quote(name), !disableIndex, !disableLog)
	return WriteGeneratedFile(filepath.Join(root, "gnosis.toml"), []byte(contents), force)
}
