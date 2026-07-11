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

//go:embed content/concepts/*.md content/gnosis/processes/*/*.md
var stagedFS embed.FS

// Documents returns the built-in documentation. Paths are relative to a vault
// root, so local vault pages with the same path take precedence.
func Documents() ([]Document, error) {
	return documents([]string{
		"content/concepts/*.md",
		"content/gnosis/processes/*/*.md",
	})
}

// Concepts returns the built-in gnosis concept definitions.
func Concepts() ([]Document, error) {
	return documents([]string{"content/concepts/*.md"})
}

func documents(patterns []string) ([]Document, error) {
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
