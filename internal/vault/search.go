package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// vaultSource is one directory in the effective vault's ordered composition.
// vaultRoot identifies the configuration directory that owns the source.
type vaultSource struct {
	path      string
	vaultRoot string
	config    Config
}

// effectiveVault owns the ordered, effective view of configured vault pages.
type effectiveVault struct {
	root    string
	config  Config
	sources []vaultSource
	backend *gitBackend
}

func loadEffectiveVault(root string) (*effectiveVault, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", filepath.Clean(root))
	}

	start := filepath.Clean(absolute)
	configPath, err := findConfigPath(start)
	if err != nil {
		return nil, err
	}
	vault := &effectiveVault{root: start, config: defaultConfig(start)}
	if configPath != "" {
		vault.root = filepath.Dir(configPath)
		vault.config, err = loadConfigPath(configPath)
		if err != nil {
			return nil, err
		}
		composer := vaultComposer{
			vault:    vault,
			resolved: make(map[string]struct{}),
			active:   make(map[string]int),
		}
		if err := composer.compose(vault.root, vault.config); err != nil {
			return nil, err
		}
	}
	for _, source := range vault.sources {
		info, err := os.Stat(source.path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", source.path)
		}
	}
	return vault, nil
}

type vaultComposer struct {
	vault    *effectiveVault
	resolved map[string]struct{}
	active   map[string]int
	stack    []string
}

func (c *vaultComposer) compose(root string, config Config) error {
	root = filepath.Clean(root)
	identity, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve vault %s: %w", root, err)
	}
	identity, err = filepath.Abs(identity)
	if err != nil {
		return err
	}
	identity = filepath.Clean(identity)
	if first, exists := c.active[identity]; exists {
		cycle := append(append([]string{}, c.stack[first:]...), root)
		return fmt.Errorf("vault import cycle: %s", strings.Join(cycle, " -> "))
	}
	if _, exists := c.resolved[identity]; exists {
		return nil
	}

	c.active[identity] = len(c.stack)
	c.stack = append(c.stack, root)
	defer func() {
		delete(c.active, identity)
		c.stack = c.stack[:len(c.stack)-1]
	}()

	if config.HasLocalVault() {
		var vaultRoot string
		if config.Vault.Backend == githubWikiBackend {
			if root != c.vault.root {
				return fmt.Errorf("GitHub Wiki backends are supported only for the primary vault")
			}
			backend, err := prepareGitHubWikiBackend(config.Vault.Repository)
			if err != nil {
				return err
			}
			c.vault.backend = backend
			vaultRoot = backend.root
		} else {
			var err error
			vaultRoot, err = resolveVaultRoot(config, root)
			if err != nil {
				return fmt.Errorf("validate %s: %w", filepath.Join(root, "gnosis.toml"), err)
			}
		}
		c.vault.sources = append(c.vault.sources, vaultSource{path: vaultRoot, vaultRoot: root, config: config})
	}

	for i, declared := range config.Vaults {
		importRoot, err := resolveDeclaredVaultRoot(declared, root)
		if err != nil {
			return fmt.Errorf("vaults[%d]: %w", i, err)
		}
		configPath := filepath.Join(importRoot, "gnosis.toml")
		info, err := os.Stat(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("vaults[%d] import target %s must contain %s", i, importRoot, configPath)
			}
			return fmt.Errorf("vaults[%d]: stat %s: %w", i, configPath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("vaults[%d] import configuration %s is not a file", i, configPath)
		}
		importConfig, err := loadConfigPath(configPath)
		if err != nil {
			return fmt.Errorf("vaults[%d]: %w", i, err)
		}
		if err := c.compose(importRoot, importConfig); err != nil {
			return fmt.Errorf("vaults[%d] %q: %w", i, declared.Name, err)
		}
	}
	c.resolved[identity] = struct{}{}
	return nil
}

func (v *effectiveVault) publish(message string) error {
	if v.backend == nil {
		return nil
	}
	return v.backend.publish(message)
}

func (v *effectiveVault) localRoot() (string, bool) {
	for _, source := range v.sources {
		if source.vaultRoot == v.root {
			return source.path, true
		}
	}
	return "", false
}

// SearchSource adapts the effective vault view into retrieval documents.
type SearchSource struct {
	vault *effectiveVault
}

// NewSearchSource resolves root and validates each configured vault root.
func NewSearchSource(root string) (*SearchSource, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	return &SearchSource{vault: vault}, nil
}

// Documents reads live concept files from every configured vault root.
func (s *SearchSource) Documents() ([]Document, error) {
	pages, err := s.vault.resolvedPages()
	if err != nil {
		return nil, err
	}

	documents := make([]Document, 0, len(pages))
	for _, page := range pages {
		documents = append(documents, page.document)
	}
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].URI < documents[j].URI
	})
	return documents, nil
}

func (v *effectiveVault) resolvedPages() ([]*effectivePage, error) {
	pages, err := v.pages()
	if err != nil {
		return nil, err
	}
	if err := resolveDocumentEdges(pages); err != nil {
		return nil, err
	}
	return pages, nil
}

// Read returns the complete Markdown document with an exact type and title.
func Read(root, conceptType, title string) ([]byte, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.pages()
	if err != nil {
		return nil, err
	}

	matches := make([]*effectivePage, 0, 1)
	for _, page := range pages {
		if page.document.Type == conceptType && page.document.Title == title {
			matches = append(matches, page)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no document found with type %q and title %q", conceptType, title)
	case 1:
		markdown, err := renderDocumentLinks(matches[0], pages)
		if err != nil {
			return nil, err
		}
		return []byte(markdown), nil
	default:
		paths := make([]string, 0, len(matches))
		for _, page := range matches {
			paths = append(paths, page.document.URI)
		}
		sort.Strings(paths)
		return nil, fmt.Errorf("multiple documents found with type %q and title %q: %s", conceptType, title, strings.Join(paths, ", "))
	}
}

func (v *effectiveVault) pages() ([]*effectivePage, error) {
	return v.loadPages(false)
}

func (v *effectiveVault) validationPages() ([]*effectivePage, error) {
	return v.loadPages(true)
}

func (v *effectiveVault) loadPages(tolerateInvalid bool) ([]*effectivePage, error) {
	pages := []*effectivePage{}
	seenPaths := make(map[string]struct{})
	seenRelativePaths := make(map[string]struct{})

	for precedence, source := range v.sources {
		err := filepath.WalkDir(source.path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != source.path && ignoredVaultDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".md" || reservedPageName(entry.Name()) {
				return nil
			}
			path = filepath.Clean(path)
			if _, exists := seenPaths[path]; exists {
				return nil
			}

			relativePath, err := filepath.Rel(source.path, path)
			if err != nil {
				return err
			}
			relativePath = filepath.ToSlash(relativePath)
			if _, exists := seenRelativePaths[relativePath]; exists {
				return nil
			}

			kind := OriginImport
			if source.vaultRoot == v.root {
				kind = OriginLocal
			}
			page, err := v.readSearchPage(source, path, Origin{
				Vault:      source.config.Vault.Name,
				Kind:       kind,
				Root:       source.path,
				Path:       path,
				Precedence: precedence,
			}, tolerateInvalid)
			if err != nil {
				return err
			}
			seenPaths[path] = struct{}{}
			seenRelativePaths[relativePath] = struct{}{}
			pages = append(pages, page)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if err := v.appendBundledPages(&pages, seenPaths, seenRelativePaths, tolerateInvalid); err != nil {
		return nil, err
	}
	return pages, nil
}

func (v *effectiveVault) appendBundledPages(pages *[]*effectivePage, seenPaths, seenRelativePaths map[string]struct{}, tolerateInvalid bool) error {
	documents, err := bundledDocuments()
	if err != nil {
		return err
	}

	const bundleRoot = ".gnosis-bundle"
	for _, document := range documents {
		relativePath := filepath.ToSlash(filepath.Clean(document.Path))
		if _, exists := seenRelativePaths[relativePath]; exists {
			continue
		}
		path := filepath.Join(bundleRoot, filepath.FromSlash(document.Path))
		if _, exists := seenPaths[path]; exists {
			continue
		}
		origin := Origin{
			Vault:      "core",
			Kind:       OriginBundle,
			Path:       document.Path,
			Precedence: len(v.sources),
		}
		var page *effectivePage
		if tolerateInvalid {
			page, err = newTolerantEffectivePage(bundleRoot, path, document.Data, origin)
		} else {
			page, err = newEffectivePage(bundleRoot, path, document.Data, origin)
		}
		if err != nil {
			return err
		}
		seenPaths[path] = struct{}{}
		seenRelativePaths[relativePath] = struct{}{}
		*pages = append(*pages, page)
	}
	return nil
}

func (v *effectiveVault) readSearchPage(source vaultSource, path string, origin Origin, tolerateInvalid bool) (*effectivePage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if tolerateInvalid {
		return newTolerantEffectivePage(source.path, path, data, origin)
	}
	return newEffectivePage(source.path, path, data, origin)
}

func resolveDocumentEdges(pages []*effectivePage) error {
	resolver := newDocumentResolver(pages)
	for _, page := range pages {
		page.document.Links = []string{}
		page.document.Edges = []Edge{}
	}

	for _, page := range pages {
		resolved, err := resolver.resolvePageLinks(page)
		if err != nil {
			return fmt.Errorf("resolve links in %s: %w", page.path, err)
		}

		seenEdges := make(map[string]struct{})
		explicitTargets := make(map[string]struct{})
		addEdge := func(target, relation, raw, source string) {
			if target == "" || target == page.document.URI {
				return
			}
			key := relation + "\x00" + target
			if _, exists := seenEdges[key]; exists {
				return
			}
			seenEdges[key] = struct{}{}
			page.document.Edges = append(page.document.Edges, Edge{
				To:       target,
				Relation: relation,
				Raw:      raw,
				Source:   source,
			})
		}

		for _, relationship := range resolved.relationships {
			relation := strings.TrimSpace(relationship.spec.Type)
			targetRaw := strings.TrimSpace(relationship.spec.Target)
			if relation == "" || targetRaw == "" {
				continue
			}
			if !relationship.include || !relationship.resolution.document || relationship.resolution.uri == "" {
				continue
			}
			target := relationship.resolution.uri
			explicitTargets[target] = struct{}{}
			addEdge(target, relation, targetRaw, "frontmatter.relationships")
		}

		for _, body := range resolved.body {
			if !body.resolution.document || body.resolution.uri == "" {
				continue
			}
			target := body.resolution.uri
			if _, explicit := explicitTargets[target]; explicit {
				continue
			}
			addEdge(target, "links_to", body.link.Raw, "body")
		}

		targets := make(map[string]struct{})
		for _, edge := range page.document.Edges {
			targets[edge.To] = struct{}{}
		}
		for target := range targets {
			page.document.Links = append(page.document.Links, target)
		}
		sort.Strings(page.document.Links)
		sort.Slice(page.document.Edges, func(i, j int) bool {
			if page.document.Edges[i].To != page.document.Edges[j].To {
				return page.document.Edges[i].To < page.document.Edges[j].To
			}
			return page.document.Edges[i].Relation < page.document.Edges[j].Relation
		})
	}
	return nil
}
