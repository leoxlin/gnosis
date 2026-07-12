package vault

import (
	"fmt"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

// SearchSource reads configured gnosis vault roots into search documents.
type SearchSource struct {
	resolution ConfigResolution
}

// NewSearchSource resolves root and validates each configured vault root.
func NewSearchSource(root string) (*SearchSource, error) {
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

	resolution, err := ResolveConfig(absolute)
	if err != nil {
		return nil, err
	}
	for _, source := range resolution.Sources {
		info, err := os.Stat(source.Path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", source.Path)
		}
	}
	return &SearchSource{resolution: resolution}, nil
}

type searchPage struct {
	root     string
	path     string
	document Document
	data     []byte
}

// Documents reads live concept files from every configured vault root.
func (s *SearchSource) Documents() ([]Document, error) {
	pages, err := s.resolvedPages()
	if err != nil {
		return nil, err
	}

	documents := make([]Document, 0, len(pages))
	for _, page := range pages {
		documents = append(documents, page.document)
	}
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].ID < documents[j].ID
	})
	return documents, nil
}

func (s *SearchSource) resolvedPages() ([]*searchPage, error) {
	pages, err := s.pages()
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
	source, err := NewSearchSource(root)
	if err != nil {
		return nil, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return nil, err
	}

	matches := make([]*searchPage, 0, 1)
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
			paths = append(paths, page.document.ID)
		}
		sort.Strings(paths)
		return nil, fmt.Errorf("multiple documents found with type %q and title %q: %s", conceptType, title, strings.Join(paths, ", "))
	}
}

func (s *SearchSource) pages() ([]*searchPage, error) {
	pages := []*searchPage{}
	seenPaths := make(map[string]struct{})
	seenIDs := make(map[string]struct{})

	for precedence, source := range s.resolution.Sources {
		err := filepath.WalkDir(source.Path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != source.Path && ignoredVaultDir(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".md" || reservedSearchFile(entry.Name()) {
				return nil
			}
			path = filepath.Clean(path)
			if _, exists := seenPaths[path]; exists {
				return nil
			}

			id, err := filepath.Rel(source.Path, path)
			if err != nil {
				return err
			}
			id = filepath.ToSlash(id)
			if _, exists := seenIDs[id]; exists {
				return nil
			}

			kind := OriginImport
			if source.VaultRoot == s.resolution.Root {
				kind = OriginLocal
			}
			page, err := s.readSearchPage(source, path, Origin{
				Vault:      source.Config.Vault.Name,
				Kind:       kind,
				Root:       source.Path,
				Path:       path,
				Precedence: precedence,
			})
			if err != nil {
				return err
			}
			seenPaths[path] = struct{}{}
			seenIDs[id] = struct{}{}
			pages = append(pages, page)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if err := s.appendBundledPages(&pages, seenPaths, seenIDs); err != nil {
		return nil, err
	}
	return pages, nil
}

func (s *SearchSource) appendBundledPages(pages *[]*searchPage, seenPaths, seenIDs map[string]struct{}) error {
	documents, err := bundledDocuments()
	if err != nil {
		return err
	}

	const bundleRoot = ".gnosis-bundle"
	for _, document := range documents {
		id := filepath.ToSlash(filepath.Clean(document.Path))
		if _, exists := seenIDs[id]; exists {
			continue
		}
		path := filepath.Join(bundleRoot, filepath.FromSlash(document.Path))
		if _, exists := seenPaths[path]; exists {
			continue
		}
		page, err := readSearchData(bundleRoot, path, document.Data, Origin{
			Vault:      "core",
			Kind:       OriginBundle,
			Path:       document.Path,
			Precedence: len(s.resolution.Sources),
		})
		if err != nil {
			return err
		}
		seenPaths[path] = struct{}{}
		seenIDs[id] = struct{}{}
		*pages = append(*pages, page)
	}
	return nil
}

func (s *SearchSource) readSearchPage(source VaultSource, path string, origin Origin) (*searchPage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return readSearchData(source.Path, path, data, origin)
}

func readSearchData(root, path string, data []byte, origin Origin) (*searchPage, error) {
	fields, body, err := parseFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	conceptType, err := requiredSearchScalar(fields, "type")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	title, err := optionalSearchScalar(fields, "title")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	description, err := optionalSearchScalar(fields, "description")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if strings.TrimSpace(description) == "" {
		description, err = optionalSearchScalar(fields, "summary")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	tags, valid := fields.scalars("tags")
	if !valid {
		return nil, fmt.Errorf("%s: frontmatter %q must be a scalar or sequence of scalars", path, "tags")
	}
	aliases, valid := fields.scalars("aliases")
	if !valid {
		return nil, fmt.Errorf("%s: frontmatter %q must be a scalar or sequence of scalars", path, "aliases")
	}

	if strings.TrimSpace(title) == "" {
		title = firstHeading(body)
	}
	if strings.TrimSpace(title) == "" {
		title = humanizeName(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}

	return &searchPage{
		root: root,
		path: filepath.Clean(path),
		data: data,
		document: Document{
			ID:          filepath.ToSlash(relative),
			URI:         documentURI(origin.Vault, filepath.ToSlash(relative)),
			Title:       strings.TrimSpace(title),
			Description: strings.TrimSpace(description),
			Type:        strings.TrimSpace(conceptType),
			Aliases:     aliases,
			Tags:        tags,
			Body:        body,
			Links:       []string{},
			Edges:       []Edge{},
			Origin:      origin,
			Revision:    documentRevision(data),
		},
	}, nil
}

func resolveDocumentEdges(pages []*searchPage) error {
	pathIDs := make(map[string]string, len(pages))
	idPages := make(map[string]*searchPage, len(pages))
	uriIDs := make(map[string]string, len(pages))
	for _, page := range pages {
		pathIDs[page.path] = page.document.ID
		idPages[page.document.ID] = page
		uriIDs[page.document.URI] = page.document.ID
		page.document.Links = []string{}
		page.document.Edges = []Edge{}
	}

	for _, page := range pages {
		fields, _, err := parseFrontmatter(string(page.data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", page.path, err)
		}
		specs, err := relationshipSpecs(fields)
		if err != nil {
			return fmt.Errorf("parse relationships in %s: %w", page.path, err)
		}

		seenEdges := make(map[string]struct{})
		explicitTargets := make(map[string]struct{})
		addEdge := func(target, relation, raw, source string) {
			if target == "" || target == page.document.ID {
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

		for _, spec := range specs {
			relation := strings.TrimSpace(spec.Type)
			targetRaw := strings.TrimSpace(spec.Target)
			if relation == "" || targetRaw == "" {
				continue
			}
			target := resolveDocumentTarget(page, targetRaw, pathIDs, idPages, uriIDs)
			if target == "" {
				continue
			}
			explicitTargets[target] = struct{}{}
			addEdge(target, relation, targetRaw, "frontmatter.relationships")
		}

		links, err := internalLinks(page.document.Body)
		if err != nil {
			return fmt.Errorf("parse links in %s: %w", page.path, err)
		}
		for _, link := range links {
			target := resolveDocumentTarget(page, link.Raw, pathIDs, idPages, uriIDs)
			if target == "" {
				continue
			}
			if _, explicit := explicitTargets[target]; explicit {
				continue
			}
			addEdge(target, "links_to", link.Raw, "body")
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

func resolveDocumentTarget(page *searchPage, raw string, pathIDs map[string]string, idPages map[string]*searchPage, uriIDs map[string]string) string {
	raw = strings.TrimSpace(raw)
	if id, exists := uriIDs[raw]; exists {
		return id
	}
	if canonical, ok := canonicalGnosisURI(raw); ok {
		if id, exists := uriIDs[canonical]; exists {
			return id
		}
	}
	link, include, err := parseLinkDestination(raw)
	if err != nil || !include {
		return ""
	}
	extension := strings.ToLower(filepath.Ext(link.Path))
	if extension != "" && extension != ".md" {
		return ""
	}

	target, err := linkTarget(page.root, page.path, link)
	if err == nil {
		candidates := []string{target}
		if extension == "" {
			candidates = append(candidates, target+".md")
		}
		for _, candidate := range candidates {
			if id, exists := pathIDs[filepath.Clean(candidate)]; exists {
				return id
			}
		}
	}

	for _, candidate := range logicalDocumentCandidates(page.document.ID, link) {
		if _, exists := idPages[candidate]; exists {
			return candidate
		}
	}
	return ""
}

func logicalDocumentCandidates(sourceID string, link Link) []string {
	logical := filepath.ToSlash(link.Path)
	if link.Absolute {
		logical = strings.TrimPrefix(logical, "/")
	} else {
		logical = pathpkg.Join(pathpkg.Dir(sourceID), logical)
	}
	logical = pathpkg.Clean(logical)
	if logical == ".." || strings.HasPrefix(logical, "../") {
		return nil
	}
	candidates := []string{logical}
	if filepath.Ext(link.Path) == "" {
		candidates = append(candidates, logical+".md")
	}
	return candidates
}

func requiredSearchScalar(fields Frontmatter, key string) (string, error) {
	value, scalar := fields.scalar(key)
	if !scalar {
		if _, exists := fields[key]; exists {
			return "", fmt.Errorf("frontmatter %q must be a scalar", key)
		}
		return "", fmt.Errorf("missing non-empty %q frontmatter", key)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing non-empty %q frontmatter", key)
	}
	return value, nil
}

func optionalSearchScalar(fields Frontmatter, key string) (string, error) {
	value, scalar := fields.scalar(key)
	if scalar {
		return strings.TrimSpace(value), nil
	}
	if _, exists := fields[key]; exists {
		return "", fmt.Errorf("frontmatter %q must be a scalar", key)
	}
	return "", nil
}

func reservedSearchFile(name string) bool {
	return name == "index.md" || name == "log.md"
}
