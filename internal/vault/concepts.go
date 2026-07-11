package vault

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// ListConcepts writes concept type previews, or concepts of an exact type,
// from the vault rooted at root.
func ListConcepts(root, conceptType string, output io.Writer) error {
	conceptType = strings.TrimSpace(conceptType)
	source, err := NewSearchSource(root)
	if err != nil {
		return fmt.Errorf("concepts: %w", err)
	}
	documents, err := source.Documents()
	if err != nil {
		return fmt.Errorf("concepts: %w", err)
	}
	if conceptType == "" {
		writeConceptTypePreviews(output, documents)
		return nil
	}

	matching := make([]Document, 0)
	for _, document := range documents {
		if document.Type == conceptType {
			matching = append(matching, document)
		}
	}
	sort.Slice(matching, func(i, j int) bool {
		if matching[i].Title == matching[j].Title {
			return matching[i].ID < matching[j].ID
		}
		return matching[i].Title < matching[j].Title
	})

	if len(matching) == 0 {
		fmt.Fprintf(output, "no concepts with type %q\n", conceptType)
		return nil
	}
	for _, document := range matching {
		fmt.Fprintf(output, "Title: %s\n", document.Title)
		fmt.Fprintf(output, "Description: %s\n\n", document.Description)
	}
	return nil
}

type conceptTypePreview struct {
	name        string
	description string
}

func writeConceptTypePreviews(output io.Writer, documents []Document) {
	descriptions := make(map[string]string)
	types := make(map[string]struct{})
	for _, document := range documents {
		if document.Type == "Concept Type" {
			types[document.Title] = struct{}{}
			if _, exists := descriptions[document.Title]; !exists {
				descriptions[document.Title] = document.Description
			}
			continue
		}
		types[document.Type] = struct{}{}
	}

	previews := make([]conceptTypePreview, 0, len(types))
	for name := range types {
		description := strings.TrimSpace(descriptions[name])
		if description == "" {
			description = name
		}
		previews = append(previews, conceptTypePreview{
			name:        name,
			description: description,
		})
	}
	sort.Slice(previews, func(i, j int) bool {
		return previews[i].name < previews[j].name
	})

	if len(previews) == 0 {
		fmt.Fprintln(output, "no concept types")
		return
	}
	for _, preview := range previews {
		fmt.Fprintf(output, "Type: %s\nDescription: %s\n\n", preview.name, preview.description)
	}
}
