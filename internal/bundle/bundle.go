// Package bundle exposes the documentation staged into the gnosis CLI build.
package bundle

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// Document is one Markdown document included in the CLI binary.
type Document struct {
	Path string
	Data []byte
}

//go:embed content/concepts/repository-*.md content/concepts/vault-*.md content/documentation/*.md content/repository/processes/*.md content/vault/processes/*.md
var stagedFS embed.FS

// Documents returns the enabled built-in documentation. Paths are relative to
// a vault root, so local vault pages with the same path take precedence.
func Documents(includeForge, includeVault bool) ([]Document, error) {
	patterns := []string{}
	if includeForge {
		patterns = append(patterns, "content/concepts/repository-*.md", "content/repository/processes/*.md")
	}
	if includeVault {
		patterns = append(patterns, "content/concepts/vault-*.md", "content/documentation/*.md", "content/vault/processes/*.md")
	}

	paths := []string{}
	for _, pattern := range patterns {
		matches, err := fs.Glob(stagedFS, pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)

	documents := make([]Document, 0, len(paths))
	for _, path := range paths {
		data, err := stagedFS.ReadFile(path)
		if err != nil {
			return nil, err
		}
		documents = append(documents, Document{
			Path: strings.TrimPrefix(path, "content/"),
			Data: data,
		})
	}
	return documents, nil
}
