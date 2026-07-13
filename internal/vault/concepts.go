package vault

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// ConceptTypeSummary is compact metadata for one available concept type.
type ConceptTypeSummary struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	URI         string `json:"uri,omitempty"`
}

// ConceptCatalog is the machine-readable form of the concepts command.
type ConceptCatalog struct {
	Type         string               `json:"type,omitempty"`
	ConceptTypes []ConceptTypeSummary `json:"concept_types,omitempty"`
	Concepts     []DocumentRef        `json:"concepts,omitempty"`
}

// ConceptRecordCatalog is the machine-readable form of an exact-type concepts
// query. Each record contains its canonical URI and complete frontmatter.
type ConceptRecordCatalog map[string][]map[string]any

// ConceptRecords returns exact-type records with all authored frontmatter.
func ConceptRecords(root, conceptType string) (ConceptRecordCatalog, error) {
	conceptType = strings.TrimSpace(conceptType)
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, fmt.Errorf("concepts: %w", err)
	}
	pages, err := vault.pages()
	if err != nil {
		return nil, fmt.Errorf("concepts: %w", err)
	}

	records := make([]map[string]any, 0)
	for _, page := range pages {
		if page.document.Type != conceptType {
			continue
		}
		records = append(records, page.authoredRecord())
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i]["uri"].(string) < records[j]["uri"].(string)
	})
	return ConceptRecordCatalog{"concepts": records}, nil
}

// Concepts returns concept type previews or exact-type document references.
func Concepts(root, conceptType string) (ConceptCatalog, error) {
	conceptType = strings.TrimSpace(conceptType)
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return ConceptCatalog{}, fmt.Errorf("concepts: %w", err)
	}
	pages, err := vault.pages()
	if err != nil {
		return ConceptCatalog{}, fmt.Errorf("concepts: %w", err)
	}
	documents := make([]Document, 0, len(pages))
	for _, page := range pages {
		documents = append(documents, page.document)
	}
	if conceptType == "" {
		return ConceptCatalog{ConceptTypes: conceptTypeSummaries(documents)}, nil
	}

	matching := make([]DocumentRef, 0)
	for _, document := range documents {
		if document.Type == conceptType {
			matching = append(matching, document.Ref())
		}
	}
	sort.Slice(matching, func(i, j int) bool {
		if matching[i].Title == matching[j].Title {
			return matching[i].URI < matching[j].URI
		}
		return matching[i].Title < matching[j].Title
	})
	return ConceptCatalog{Type: conceptType, Concepts: matching}, nil
}

// ListConcepts writes concept type previews, or concepts of an exact type,
// from the vault rooted at root.
func ListConcepts(root, conceptType string, output io.Writer) error {
	catalog, err := Concepts(root, conceptType)
	if err != nil {
		return err
	}
	if catalog.Type == "" {
		writeConceptTypePreviews(output, catalog.ConceptTypes)
		return nil
	}
	if len(catalog.Concepts) == 0 {
		fmt.Fprintf(output, "no concepts with type %q\n", catalog.Type)
		return nil
	}
	for _, document := range catalog.Concepts {
		fmt.Fprintf(output, "Title: %s\n", document.Title)
		fmt.Fprintf(output, "Description: %s\n", document.Description)
		fmt.Fprintf(output, "Link: %s\n\n", document.URI)
	}
	return nil
}

func conceptTypeSummaries(documents []Document) []ConceptTypeSummary {
	descriptions := make(map[string]string)
	uris := make(map[string]string)
	types := make(map[string]struct{})
	for _, document := range documents {
		if document.Type == "ConceptType" {
			types[document.Title] = struct{}{}
			if _, exists := descriptions[document.Title]; !exists {
				descriptions[document.Title] = document.Description
				uris[document.Title] = document.URI
			}
			continue
		}
		types[document.Type] = struct{}{}
	}

	previews := make([]ConceptTypeSummary, 0, len(types))
	for name := range types {
		description := strings.TrimSpace(descriptions[name])
		if description == "" {
			description = name
		}
		previews = append(previews, ConceptTypeSummary{
			Type:        name,
			Description: description,
			URI:         uris[name],
		})
	}
	sort.Slice(previews, func(i, j int) bool {
		return previews[i].Type < previews[j].Type
	})
	return previews
}

func writeConceptTypePreviews(output io.Writer, previews []ConceptTypeSummary) {
	if len(previews) == 0 {
		fmt.Fprintln(output, "no concept types")
		return
	}
	for _, preview := range previews {
		fmt.Fprintf(output, "Type: %s\nDescription: %s\n", preview.Type, preview.Description)
		if preview.URI != "" {
			fmt.Fprintf(output, "Link: %s\n", preview.URI)
		}
		fmt.Fprintln(output)
	}
}
