package vault

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
)

const (
	ResourceMediaType = "text/markdown"
	resourcePageSize  = 100
)

var (
	ErrInvalidResourceCursor = errors.New("invalid resource cursor")
	ErrPageNotFound          = errors.New("page not found")
)

// ResourceDescriptor is the bounded discovery representation of one effective page.
type ResourceDescriptor struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	MediaType   string `json:"media_type"`
	Size        int64  `json:"size"`
	Origin      Origin `json:"origin"`
	Revision    string `json:"revision"`
}

// ResourceContent is one effective page rendered as protocol-neutral content.
type ResourceContent struct {
	URI       string `json:"uri"`
	MediaType string `json:"media_type"`
	Markdown  string `json:"markdown"`
	Origin    Origin `json:"origin"`
	Revision  string `json:"revision"`
}

// ResourcePage is one bounded page of deterministic resource discovery.
type ResourcePage struct {
	Resources  []ResourceDescriptor `json:"resources"`
	NextCursor string               `json:"next_cursor,omitempty"`
}

// ListResourcePage lists effective pages in canonical URI order.
func ListResourcePage(root, cursor string) (ResourcePage, error) {
	effective, err := loadEffectiveVault(root)
	if err != nil {
		return ResourcePage{}, err
	}
	pages, err := effective.resolvedPages()
	if err != nil {
		return ResourcePage{}, err
	}
	resources := make([]ResourceDescriptor, 0, len(pages))
	for _, page := range pages {
		markdown, err := renderDocumentLinks(page, pages)
		if err != nil {
			return ResourcePage{}, err
		}
		resources = append(resources, resourceDescriptor(page.document.Ref(), markdown))
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].URI < resources[j].URI })
	return paginateResources(resources, cursor)
}

// ReadResource reads an effective page through the shared exact page reader.
func ReadResource(root, uri string) (ResourceContent, error) {
	page, err := ReadPage(root, uri)
	if err != nil {
		return ResourceContent{}, err
	}
	return ResourceContent{
		URI:       page.Document.URI,
		MediaType: ResourceMediaType,
		Markdown:  page.Markdown,
		Origin:    page.Document.Origin,
		Revision:  page.Document.Revision,
	}, nil
}

func resourceDescriptor(document DocumentRef, markdown string) ResourceDescriptor {
	return ResourceDescriptor{
		URI:         document.URI,
		Title:       document.Title,
		Description: document.Description,
		MediaType:   ResourceMediaType,
		Size:        int64(len(markdown)),
		Origin:      document.Origin,
		Revision:    document.Revision,
	}
}

func paginateResources(resources []ResourceDescriptor, cursor string) (ResourcePage, error) {
	start := 0
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return ResourcePage{}, fmt.Errorf("%w: malformed", ErrInvalidResourceCursor)
		}
		previous := string(decoded)
		start = sort.Search(len(resources), func(i int) bool { return resources[i].URI >= previous })
		if start == len(resources) || resources[start].URI != previous {
			return ResourcePage{}, fmt.Errorf("%w: unknown position", ErrInvalidResourceCursor)
		}
		start++
	}
	end := min(start+resourcePageSize, len(resources))
	page := ResourcePage{Resources: resources[start:end]}
	if end < len(resources) {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(resources[end-1].URI))
	}
	return page, nil
}
