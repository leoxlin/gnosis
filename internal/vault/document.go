package vault

import (
	"fmt"
	"sort"
)

// OriginKind identifies where an effective document came from.
type OriginKind string

const (
	OriginLocal  OriginKind = "local"
	OriginImport OriginKind = "import"
	OriginBundle OriginKind = "bundle"
)

// Origin preserves the selected source behind an effective vault document.
type Origin struct {
	Vault      string     `json:"vault"`
	Kind       OriginKind `json:"kind"`
	Root       string     `json:"root,omitempty"`
	Path       string     `json:"path,omitempty"`
	Precedence int        `json:"precedence"`
}

// Edge is one directed relationship from a document to another effective URI.
type Edge struct {
	To       string `json:"to"`
	Relation string `json:"relation"`
	Raw      string `json:"raw,omitempty"`
	Source   string `json:"source,omitempty"`
}

// Document is a vault page available to retrieval consumers. URI is its sole
// stable, unique identity. Path is used only to resolve authored relative links.
type Document struct {
	Path        string
	URI         string
	Title       string
	Description string
	Type        string
	Aliases     []string
	Tags        []string
	Body        string
	Links       []string
	Edges       []Edge
	Origin      Origin
	Revision    string
}

// DocumentRef is the compact agent-facing representation of a document.
type DocumentRef struct {
	URI         string `json:"uri"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Origin      Origin `json:"origin"`
	Revision    string `json:"revision"`
}

// Ref returns the agent-facing identity for a document.
func (d Document) Ref() DocumentRef {
	return DocumentRef{
		URI:         d.URI,
		Type:        d.Type,
		Title:       d.Title,
		Description: d.Description,
		Origin:      d.Origin,
		Revision:    d.Revision,
	}
}

// Page is an exact vault page and its complete Markdown source.
type Page struct {
	Document DocumentRef `json:"document"`
	Markdown string      `json:"markdown"`
}

// ListPages returns every effective page in deterministic URI order.
func ListPages(root string) ([]DocumentRef, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.pages()
	if err != nil {
		return nil, err
	}
	result := make([]DocumentRef, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.document.Ref())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].URI < result[j].URI })
	return result, nil
}

// ReadPage reads one exact effective page by gnosis URI.
func ReadPage(root, selector string) (Page, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return Page{}, err
	}
	pages, err := vault.pages()
	if err != nil {
		return Page{}, err
	}
	page, ok := selectPage(pages, selector)
	if !ok {
		return Page{}, fmt.Errorf("no document found with URI %q", selector)
	}
	markdown, err := renderDocumentLinks(page, pages)
	if err != nil {
		return Page{}, err
	}
	return Page{Document: page.document.Ref(), Markdown: markdown}, nil
}

func selectPage(pages []*effectivePage, selector string) (*effectivePage, bool) {
	vaultName, pagePath, ok := canonicalGnosisParts(selector)
	if !ok {
		return nil, false
	}
	canonical := documentURI(vaultName, pagePath)
	for _, page := range pages {
		if vaultName == anyVaultAuthority && page.document.Path == pagePath {
			return page, true
		}
		if page.document.URI == canonical {
			return page, true
		}
	}
	return nil, false
}
