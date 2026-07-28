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
}

// VaultConfig holds local vault settings.
type VaultConfig struct {
	Name             string   `toml:"vault_name"`
	Root             string   `toml:"vault_root"`
	Backend          string   `toml:"backend"`
	Repository       string   `toml:"repository"`
	EntryPoints      []string `toml:"entry_points"`
	LinkFormat       string   `toml:"link_format"`
	LinkFormatStrict bool     `toml:"link_format_strict"`
	VaultIndex       bool     `toml:"vault_index"`
	VaultLog         bool     `toml:"vault_log"`
}

// DeclaredVaultConfig identifies one additional vault root in a workspace.
type DeclaredVaultConfig struct {
	Name string `toml:"vault_name"`
	Root string `toml:"vault_root"`
}

func defaultConfig(root string) Config {
	config := Config{
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
			VaultIndex:       true,
			VaultLog:         true,
		},
	}
	if withinGitRepository(root) {
		config.Vault.Name = "local"
		config.Vault.Root = "docs"
		config.Vault.LinkFormatStrict = true
		config.Vault.VaultIndex = false
		config.Vault.VaultLog = false
	}
	return config
}

func withinGitRepository(root string) bool {
	_, found := gitRepositoryRoot(root)
	return found
}

func gitRepositoryRoot(root string) (string, bool) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	for {
		if isGitWorkTree(root) {
			return filepath.Clean(root), true
		}
		parent := filepath.Dir(root)
		if parent == root {
			return "", false
		}
		root = parent
	}
}

func isGitWorkTree(root string) bool {
	gitPath := filepath.Join(root, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	if info.IsDir() {
		_, err = os.Stat(filepath.Join(gitPath, "HEAD"))
		return err == nil
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return false
	}
	gitDir, found := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir: ")
	if !found {
		return false
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	_, err = os.Stat(filepath.Join(gitDir, "HEAD"))
	return err == nil
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
	return strings.TrimSpace(c.Vault.Name) != "" || strings.TrimSpace(c.Vault.Root) != "" || strings.TrimSpace(c.Vault.Backend) != "" || strings.TrimSpace(c.Vault.Repository) != ""
}

func findConfigPath(root string) (string, error) {
	globalPath, err := userConfigPath()
	if err != nil {
		return "", err
	}

	searchRoots := []string{filepath.Clean(root)}
	if repositoryRoot, found := gitRepositoryRoot(root); found {
		for current := filepath.Clean(root); current != repositoryRoot; {
			current = filepath.Dir(current)
			searchRoots = append(searchRoots, current)
		}
	}
	for _, searchRoot := range searchRoots {
		for _, name := range []string{"gnosis.local.toml", "gnosis.toml"} {
			path := filepath.Join(searchRoot, name)
			info, err := os.Stat(path)
			if err == nil && !info.IsDir() {
				return path, nil
			}
			if err != nil && !os.IsNotExist(err) {
				return "", fmt.Errorf("stat %s: %w", path, err)
			}
		}
	}

	info, err := os.Stat(globalPath)
	if err == nil && !info.IsDir() {
		return globalPath, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("stat %s: %w", globalPath, err)
	}
	if err == nil && info.IsDir() {
		return "", nil
	}
	return "", nil
}

func userConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gnosis.toml"), nil
}

func implicitVaultRoot(start string) string {
	if repositoryRoot, found := gitRepositoryRoot(start); found {
		return repositoryRoot
	}
	return start
}

func loadConfig(root string) (Config, error) {
	return loadConfigPath(filepath.Join(root, "gnosis.toml"))
}

func loadConfigPath(path string) (Config, error) {
	config := defaultConfig(filepath.Dir(path))
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

func validateConfig(config Config, root string) error {
	if config.HasLocalVault() {
		if strings.TrimSpace(config.Vault.Name) == "" {
			return fmt.Errorf("vault.vault_name must not be empty")
		}
		if !isCanonicalVaultName(config.Vault.Name) {
			return fmt.Errorf("vault.vault_name %q must be a canonical gnosis URI authority", config.Vault.Name)
		}
		switch config.Vault.Backend {
		case "":
			if strings.TrimSpace(config.Vault.Repository) != "" {
				return fmt.Errorf("vault.repository requires a backend")
			}
			if strings.TrimSpace(config.Vault.Root) == "" {
				return fmt.Errorf("vault.vault_root must not be empty")
			}
			if _, err := resolveVaultRoot(config, root); err != nil {
				return err
			}
		case githubWikiBackend:
			if strings.TrimSpace(config.Vault.Root) != "" {
				return fmt.Errorf("vault.vault_root must be empty for backend %q", githubWikiBackend)
			}
			if err := validateGitHubRepository(config.Vault.Repository); err != nil {
				return fmt.Errorf("vault.repository: %w", err)
			}
		default:
			return fmt.Errorf("vault.backend %q is not supported", config.Vault.Backend)
		}
		switch config.Vault.LinkFormat {
		case string(LinkFormatRelative), string(LinkFormatAbsolute):
		default:
			return fmt.Errorf("vault.link_format must be %q or %q, got %q", LinkFormatRelative, LinkFormatAbsolute, config.Vault.LinkFormat)
		}
		for index, uri := range config.Vault.EntryPoints {
			if !IsCanonicalURI(strings.TrimSpace(uri)) {
				return fmt.Errorf("vault.entry_points[%d] must be a canonical gnosis URI", index)
			}
		}
	}
	for i, declared := range config.Vaults {
		if strings.TrimSpace(declared.Name) == "" {
			return fmt.Errorf("vaults[%d].vault_name must not be empty", i)
		}
		if err := validateDeclaredVaultRoot(declared, root); err != nil {
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

func isCanonicalVaultName(name string) bool {
	if name == "" || name == anyVaultAuthority || name != strings.TrimSpace(name) {
		return false
	}
	probe := documentURI(name, "probe.md")
	vaultName, path, ok := canonicalGnosisParts(probe)
	return ok && vaultName == name && path == "probe.md"
}

func resolveDeclaredVaultRoot(config DeclaredVaultConfig, root string) (string, error) {
	path := strings.TrimSpace(config.Root)
	if path == "" {
		return "", fmt.Errorf("vault_root must not be empty")
	}
	if _, remote, err := parseRemoteLocator(path); err != nil {
		return "", err
	} else if remote {
		target, err := resolveVaultTarget(path)
		if err != nil {
			return "", err
		}
		return target.root, nil
	}
	return resolveLocalDeclaredVaultRoot(path, root)
}

func validateDeclaredVaultRoot(config DeclaredVaultConfig, root string) error {
	path := strings.TrimSpace(config.Root)
	if path == "" {
		return fmt.Errorf("vault_root must not be empty")
	}
	if _, remote, err := parseRemoteLocator(path); err != nil || remote {
		return err
	}
	_, err := resolveLocalDeclaredVaultRoot(path, root)
	return err
}

func resolveLocalDeclaredVaultRoot(path, root string) (string, error) {
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
func WriteWorkspaceConfig(root string, imports []string, force bool) (bool, string, error) {
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
	return writeTargetFile(root, "gnosis.toml", []byte(contents.String()), force, "gnosis: configure workspace")
}

// WriteGitHubWikiConfig configures a GitHub Wiki as the primary vault.
func WriteGitHubWikiConfig(root, name, repository string, force bool) (bool, string, error) {
	if !isCanonicalVaultName(name) {
		return false, "", fmt.Errorf("vault name %q must be a canonical gnosis URI authority", name)
	}
	if err := validateGitHubRepository(repository); err != nil {
		return false, "", fmt.Errorf("GitHub Wiki repository: %w", err)
	}
	contents := fmt.Sprintf(`[vault]
vault_name = %s
backend = %q
repository = %s
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
`, strconv.Quote(name), githubWikiBackend, strconv.Quote(repository))
	return writeTargetFile(root, "gnosis.toml", []byte(contents), force, "gnosis: configure workspace")
}

func writeVaultConfig(root, name string, disableIndex, disableLog, force bool) (bool, error) {
	configPath := filepath.Join(root, "gnosis.toml")
	if !force {
		if _, err := os.Stat(configPath); err == nil {
			return false, nil
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	if strings.TrimSpace(name) == "" {
		absolute, err := filepath.Abs(root)
		if err != nil {
			return false, err
		}
		name = filepath.Base(absolute)
	}
	if !isCanonicalVaultName(name) {
		return false, fmt.Errorf("vault name %q must be a canonical gnosis URI authority", name)
	}
	contents := fmt.Sprintf(`[vault]
vault_name = %s
vault_root = "."
link_format = "relative"
link_format_strict = false
vault_index = %t
vault_log = %t
`, strconv.Quote(name), !disableIndex, !disableLog)
	return WriteGeneratedFile(configPath, []byte(contents), force)
}
