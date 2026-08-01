package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gnosis/internal/s3store"
)

// LinkFormat is the preferred style for internal markdown links.
type LinkFormat string

const (
	LinkFormatRelative LinkFormat = "relative"
	LinkFormatAbsolute LinkFormat = "absolute"
)

// Config is the gnosis configuration for local and declared vaults.
type Config struct {
	Vault      VaultConfig           `toml:"vault"`
	Vaults     []DeclaredVaultConfig `toml:"vaults"`
	Hooks      []HookConfig          `toml:"hooks"`
	GitHub     []GitHubConfig        `toml:"github"`
	CodeScopes []CodeScopeConfig     `toml:"code_scopes"`
}

// CodeScopeConfig binds one named code index to a local Git repository.
type CodeScopeConfig struct {
	Name           string   `toml:"name"`
	Root           string   `toml:"root"`
	Languages      []string `toml:"languages"`
	Live           bool     `toml:"live"`
	FreshnessWait  string   `toml:"freshness_wait"`
	MaxFiles       int      `toml:"max_files"`
	MaxFileBytes   int64    `toml:"max_file_bytes"`
	MaxRecords     int      `toml:"max_records"`
	MaxDiagnostics int      `toml:"max_diagnostics"`
	MaxResults     int      `toml:"max_results"`
	MaxTraversal   int      `toml:"max_traversal"`
}

const (
	DefaultCodeMaxFiles       = 10_000
	DefaultCodeMaxFileBytes   = 2 << 20
	DefaultCodeMaxRecords     = 500_000
	DefaultCodeMaxDiagnostics = 10_000
	DefaultCodeMaxResults     = 100
	DefaultCodeMaxTraversal   = 1_000
	DefaultCodeFreshnessWait  = 2 * time.Second
	MaxCodeFreshnessWait      = 30 * time.Second
)

// VaultConfig holds local vault settings.
type VaultConfig struct {
	Name             string   `toml:"vault_name"`
	Root             string   `toml:"vault_root"`
	Backend          string   `toml:"backend"`
	Repository       string   `toml:"repository"`
	S3Bucket         string   `toml:"s3_bucket"`
	S3Region         string   `toml:"s3_region"`
	S3Prefix         string   `toml:"s3_prefix"`
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

// GitHubConfig binds one allowed repository to a named vault evidence store.
type GitHubConfig struct {
	Repository       string `toml:"repository"`
	EvidenceDir      string `toml:"evidence_dir"`
	EvidenceBackend  string `toml:"evidence_backend"`
	S3Bucket         string `toml:"s3_bucket"`
	S3Region         string `toml:"s3_region"`
	S3Prefix         string `toml:"s3_prefix"`
	TokenEnv         string `toml:"token_env"`
	WebhookSecretEnv string `toml:"webhook_secret_env"`
	PerPage          int    `toml:"per_page"`
	MaxPages         int    `toml:"max_pages"`
	MaxBodyBytes     int64  `toml:"max_body_bytes"`
}

const (
	DefaultGitHubPerPage      = 100
	DefaultGitHubMaxPages     = 20
	DefaultGitHubMaxBodyBytes = 2 << 20
)

func defaultConfig(_ string) Config {
	return Config{
		Vault: VaultConfig{
			LinkFormat:       string(LinkFormatRelative),
			LinkFormatStrict: false,
			VaultIndex:       true,
			VaultLog:         true,
		},
	}
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
	return strings.TrimSpace(c.Vault.Name) != "" || strings.TrimSpace(c.Vault.Root) != "" || strings.TrimSpace(c.Vault.Backend) != "" || strings.TrimSpace(c.Vault.Repository) != "" || strings.TrimSpace(c.Vault.S3Bucket) != "" || strings.TrimSpace(c.Vault.S3Region) != "" || strings.TrimSpace(c.Vault.S3Prefix) != ""
}

func findConfigPath(root string) (string, error) {
	root, err := filepath.Abs(root)
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
	return "", nil
}

func userConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "gnosis.toml"), nil
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
			if config.Vault.S3Bucket != "" || config.Vault.S3Region != "" || config.Vault.S3Prefix != "" {
				return fmt.Errorf("vault.s3_* requires backend %q", s3BackendName)
			}
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
			if config.Vault.S3Bucket != "" || config.Vault.S3Region != "" || config.Vault.S3Prefix != "" {
				return fmt.Errorf("vault.s3_* must be empty for backend %q", githubWikiBackend)
			}
		case s3BackendName:
			if strings.TrimSpace(config.Vault.Root) != "" || strings.TrimSpace(config.Vault.Repository) != "" {
				return fmt.Errorf("vault.vault_root and vault.repository must be empty for backend %q", s3BackendName)
			}
			if _, err := (s3store.Config{Bucket: config.Vault.S3Bucket, Region: config.Vault.S3Region, Prefix: config.Vault.S3Prefix}).Validate(); err != nil {
				return fmt.Errorf("vault: %w", err)
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
		if !isCanonicalVaultName(declared.Name) {
			return fmt.Errorf("vaults[%d].vault_name %q must be a canonical gnosis URI authority", i, declared.Name)
		}
		if err := validateDeclaredVaultRoot(declared, root); err != nil {
			return fmt.Errorf("vaults[%d]: %w", i, err)
		}
	}
	if err := validateHooks(config.Hooks, config.Vault.Name); err != nil {
		return err
	}
	if err := validateGitHubConfigs(config.GitHub); err != nil {
		return err
	}
	if err := validateCodeScopes(config.CodeScopes, root); err != nil {
		return err
	}
	return nil
}

// CodeScope resolves one explicitly configured code scope from the workspace.
func CodeScope(start, name string) (CodeScopeConfig, error) {
	path, err := findConfigPath(start)
	if err != nil {
		return CodeScopeConfig{}, err
	}
	if path == "" {
		return CodeScopeConfig{}, fmt.Errorf("no gnosis configuration found")
	}
	config, err := loadConfigPath(path)
	if err != nil {
		return CodeScopeConfig{}, err
	}
	for _, scope := range config.CodeScopes {
		if scope.Name == name {
			scope = scope.withDefaults()
			if !filepath.IsAbs(scope.Root) {
				scope.Root = filepath.Join(filepath.Dir(path), scope.Root)
			}
			resolved, err := filepath.EvalSymlinks(scope.Root)
			if err != nil {
				return CodeScopeConfig{}, fmt.Errorf("resolve code scope %q: %w", name, err)
			}
			scope.Root, err = filepath.Abs(resolved)
			return scope, err
		}
	}
	return CodeScopeConfig{}, fmt.Errorf("code scope %q is not configured", name)
}

// CodeScopes resolves every configured code scope from the workspace.
func CodeScopes(start string) ([]CodeScopeConfig, error) {
	path, err := findConfigPath(start)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("no gnosis configuration found")
	}
	config, err := loadConfigPath(path)
	if err != nil {
		return nil, err
	}
	result := make([]CodeScopeConfig, 0, len(config.CodeScopes))
	for _, configured := range config.CodeScopes {
		scope, err := CodeScope(filepath.Dir(path), configured.Name)
		if err != nil {
			return nil, err
		}
		result = append(result, scope)
	}
	return result, nil
}

// FreshnessWaitDuration returns the validated caller-visible freshness bound.
func (scope CodeScopeConfig) FreshnessWaitDuration() time.Duration {
	if scope.FreshnessWait == "" {
		return DefaultCodeFreshnessWait
	}
	duration, _ := time.ParseDuration(scope.FreshnessWait)
	return duration
}

func validateCodeScopes(scopes []CodeScopeConfig, root string) error {
	seen := map[string]bool{}
	for index, scope := range scopes {
		prefix := fmt.Sprintf("code_scopes[%d]", index)
		if strings.TrimSpace(scope.Name) == "" || strings.ContainsAny(scope.Name, "/\\\x00") {
			return fmt.Errorf("%s.name must be a non-empty canonical name", prefix)
		}
		if seen[scope.Name] {
			return fmt.Errorf("%s.name %q is duplicated", prefix, scope.Name)
		}
		seen[scope.Name] = true
		if strings.TrimSpace(scope.Root) == "" {
			return fmt.Errorf("%s.root must not be empty", prefix)
		}
		resolved := scope.Root
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(root, resolved)
		}
		if _, err := filepath.EvalSymlinks(resolved); err != nil {
			return fmt.Errorf("%s.root: %w", prefix, err)
		}
		languages := map[string]bool{}
		for _, language := range scope.Languages {
			canonical := strings.ToLower(strings.TrimSpace(language))
			if language != canonical || canonical == "" || strings.ContainsAny(canonical, "/\\.\x00") {
				return fmt.Errorf("%s.languages contains invalid language %q", prefix, language)
			}
			if languages[canonical] {
				return fmt.Errorf("%s.languages contains duplicate %q", prefix, language)
			}
			languages[canonical] = true
		}
		if len(languages) == 0 {
			return fmt.Errorf("%s.languages must not be empty", prefix)
		}
		defaults := scope.withDefaults()
		if defaults.MaxFiles < 1 || defaults.MaxFileBytes < 1 || defaults.MaxRecords < 1 || defaults.MaxDiagnostics < 1 || defaults.MaxResults < 1 || defaults.MaxTraversal < 1 {
			return fmt.Errorf("%s bounds must be positive", prefix)
		}
		freshnessWait := defaults.FreshnessWaitDuration()
		if freshnessWait <= 0 || freshnessWait > MaxCodeFreshnessWait {
			return fmt.Errorf("%s.freshness_wait must be greater than zero and at most %s", prefix, MaxCodeFreshnessWait)
		}
	}
	return nil
}

func (scope CodeScopeConfig) withDefaults() CodeScopeConfig {
	if scope.FreshnessWait == "" {
		scope.FreshnessWait = DefaultCodeFreshnessWait.String()
	}
	if scope.MaxFiles == 0 {
		scope.MaxFiles = DefaultCodeMaxFiles
	}
	if scope.MaxFileBytes == 0 {
		scope.MaxFileBytes = DefaultCodeMaxFileBytes
	}
	if scope.MaxRecords == 0 {
		scope.MaxRecords = DefaultCodeMaxRecords
	}
	if scope.MaxDiagnostics == 0 {
		scope.MaxDiagnostics = DefaultCodeMaxDiagnostics
	}
	if scope.MaxResults == 0 {
		scope.MaxResults = DefaultCodeMaxResults
	}
	if scope.MaxTraversal == 0 {
		scope.MaxTraversal = DefaultCodeMaxTraversal
	}
	return scope
}

// GitHubRepositoryConfig loads one explicitly allowed repository for the selected vault root.
func GitHubRepositoryConfig(start, repository string) (string, GitHubConfig, error) {
	path, err := findConfigPath(start)
	if err != nil {
		return "", GitHubConfig{}, err
	}
	if path == "" {
		return "", GitHubConfig{}, fmt.Errorf("no gnosis configuration found")
	}
	config, err := loadConfigPath(path)
	if err != nil {
		return "", GitHubConfig{}, err
	}
	repository = strings.ToLower(strings.TrimSpace(repository))
	for _, github := range config.GitHub {
		if strings.ToLower(github.Repository) == repository {
			return config.Vault.Name, github.withDefaults(), nil
		}
	}
	return "", GitHubConfig{}, fmt.Errorf(
		"github repository %q is not configured for vault %q",
		repository,
		config.Vault.Name,
	)
}

func validateGitHubConfigs(configs []GitHubConfig) error {
	seen := map[string]bool{}
	for index, config := range configs {
		prefix := fmt.Sprintf("github[%d]", index)
		repository := strings.ToLower(strings.TrimSpace(config.Repository))
		if err := validateGitHubRepository(repository); err != nil {
			return fmt.Errorf("%s.repository: %w", prefix, err)
		}
		if seen[repository] {
			return fmt.Errorf("%s.repository %q is duplicated", prefix, repository)
		}
		seen[repository] = true
		switch config.EvidenceBackend {
		case "", "filesystem":
			if !filepath.IsAbs(config.EvidenceDir) {
				return fmt.Errorf("%s.evidence_dir must be an absolute path", prefix)
			}
			if config.S3Bucket != "" || config.S3Region != "" || config.S3Prefix != "" {
				return fmt.Errorf("%s.s3_* requires evidence_backend %q", prefix, s3BackendName)
			}
		case s3BackendName:
			if config.EvidenceDir != "" {
				return fmt.Errorf("%s.evidence_dir must be empty for evidence_backend %q", prefix, s3BackendName)
			}
			if _, err := (s3store.Config{Bucket: config.S3Bucket, Region: config.S3Region, Prefix: config.S3Prefix}).Validate(); err != nil {
				return fmt.Errorf("%s: %w", prefix, err)
			}
		default:
			return fmt.Errorf("%s.evidence_backend %q is not supported", prefix, config.EvidenceBackend)
		}
		if !validEnvironmentName(config.TokenEnv) {
			return fmt.Errorf("%s.token_env must be an environment variable name", prefix)
		}
		if config.WebhookSecretEnv != "" && !validEnvironmentName(config.WebhookSecretEnv) {
			return fmt.Errorf("%s.webhook_secret_env must be an environment variable name", prefix)
		}
		effective := config.withDefaults()
		if effective.PerPage < 1 || effective.PerPage > 100 {
			return fmt.Errorf("%s.per_page must be between 1 and 100", prefix)
		}
		if effective.MaxPages < 1 || effective.MaxPages > 1000 {
			return fmt.Errorf("%s.max_pages must be between 1 and 1000", prefix)
		}
		if effective.MaxBodyBytes < 1 || effective.MaxBodyBytes > 10<<20 {
			return fmt.Errorf("%s.max_body_bytes must be between 1 and %d", prefix, 10<<20)
		}
	}
	return nil
}

func (config GitHubConfig) withDefaults() GitHubConfig {
	config.Repository = strings.ToLower(strings.TrimSpace(config.Repository))
	if config.EvidenceBackend == "" {
		config.EvidenceBackend = "filesystem"
	}
	if config.EvidenceDir != "" {
		config.EvidenceDir = filepath.Clean(config.EvidenceDir)
	}
	config.S3Bucket = strings.TrimSpace(config.S3Bucket)
	config.S3Region = strings.TrimSpace(config.S3Region)
	config.S3Prefix, _ = s3store.NormalizePrefix(config.S3Prefix)
	if config.PerPage == 0 {
		config.PerPage = DefaultGitHubPerPage
	}
	if config.MaxPages == 0 {
		config.MaxPages = DefaultGitHubMaxPages
	}
	if config.MaxBodyBytes == 0 {
		config.MaxBodyBytes = DefaultGitHubMaxBodyBytes
	}
	return config
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

// WriteS3Config configures an Amazon S3 prefix as the primary vault.
func WriteS3Config(root, name, bucket, region, prefix string, force bool) (bool, string, error) {
	if !isCanonicalVaultName(name) {
		return false, "", fmt.Errorf("vault name %q must be a canonical gnosis URI authority", name)
	}
	config, err := (s3store.Config{Bucket: bucket, Region: region, Prefix: prefix}).Validate()
	if err != nil {
		return false, "", err
	}
	contents := fmt.Sprintf(`[vault]
vault_name = %s
backend = %q
s3_bucket = %s
s3_region = %s
`, strconv.Quote(name), s3BackendName, strconv.Quote(config.Bucket), strconv.Quote(config.Region))
	if config.Prefix != "" {
		contents += "s3_prefix = " + strconv.Quote(config.Prefix) + "\n"
	}
	contents += `link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
`
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
