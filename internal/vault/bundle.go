package vault

import (
	"io/fs"
	"sort"

	"gnosis/docs"
)

// BundledDocument is one Markdown document included in the gnosis binary.
type BundledDocument struct {
	Path string
	Data []byte
}

// bundledDocuments returns the built-in documentation. Paths are relative to
// a vault root, so local vault pages with the same path take precedence.
func bundledDocuments() ([]BundledDocument, error) {
	return readBundledDocuments([]string{
		"concepts/*.md",
		"gnosis/processes/*/*.md",
	})
}

// BundledConcepts returns the built-in gnosis concept definitions.
func BundledConcepts() ([]BundledDocument, error) {
	return readBundledDocuments([]string{"concepts/*.md"})
}

func readBundledDocuments(patterns []string) ([]BundledDocument, error) {
	content := docs.Content()
	paths := []string{}
	for _, pattern := range patterns {
		matches, err := fs.Glob(content, pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	sort.Strings(paths)

	documents := make([]BundledDocument, 0, len(paths))
	for _, path := range paths {
		data, err := fs.ReadFile(content, path)
		if err != nil {
			return nil, err
		}
		documents = append(documents, BundledDocument{
			Path: path,
			Data: data,
		})
	}
	return documents, nil
}
