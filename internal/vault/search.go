package vault

import (
	"fmt"
	"os"
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
	for _, vaultRoot := range resolution.VaultRoots {
		info, err := os.Stat(vaultRoot)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", vaultRoot)
		}
	}
	return &SearchSource{resolution: resolution}, nil
}

type searchPage struct {
	root     string
	path     string
	document Document
}

// Documents reads live concept files from every configured vault root.
func (s *SearchSource) Documents() ([]Document, error) {
	pages, err := s.pages()
	if err != nil {
		return nil, err
	}

	pathIDs := make(map[string]string, len(pages))
	for _, page := range pages {
		pathIDs[page.path] = page.document.ID
	}
	for _, page := range pages {
		links, err := internalLinks(page.document.Body)
		if err != nil {
			return nil, fmt.Errorf("parse links in %s: %w", page.path, err)
		}
		targets := make(map[string]struct{})
		for _, link := range links {
			extension := strings.ToLower(filepath.Ext(link.Path))
			if extension != "" && extension != ".md" {
				continue
			}
			target, err := linkTarget(page.root, page.path, link)
			if err != nil {
				continue
			}
			candidates := []string{target}
			if extension == "" {
				candidates = append(candidates, target+".md")
			}
			for _, candidate := range candidates {
				id, exists := pathIDs[filepath.Clean(candidate)]
				if !exists || id == page.document.ID {
					continue
				}
				targets[id] = struct{}{}
				break
			}
		}
		for target := range targets {
			page.document.Links = append(page.document.Links, target)
		}
		sort.Strings(page.document.Links)
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

// Read returns the complete Markdown document with an exact type and title.
func Read(root, conceptType, title string) ([]byte, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return nil, err
	}
	pages, err := source.pages()
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
		return os.ReadFile(matches[0].path)
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

	for _, vaultRoot := range s.resolution.VaultRoots {
		err := filepath.WalkDir(vaultRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != vaultRoot && ignoredVaultDir(entry.Name()) {
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

			page, err := s.readSearchPage(vaultRoot, path)
			if err != nil {
				return err
			}
			seenPaths[path] = struct{}{}
			pages = append(pages, page)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return pages, nil
}

func (s *SearchSource) readSearchPage(vaultRoot, path string) (*searchPage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
	relative, err := filepath.Rel(s.resolution.Root, path)
	if err != nil {
		return nil, err
	}

	return &searchPage{
		root: vaultRoot,
		path: filepath.Clean(path),
		document: Document{
			ID:          filepath.ToSlash(relative),
			Title:       strings.TrimSpace(title),
			Description: strings.TrimSpace(description),
			Type:        strings.TrimSpace(conceptType),
			Aliases:     aliases,
			Tags:        tags,
			Body:        body,
			Links:       []string{},
		},
	}, nil
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
