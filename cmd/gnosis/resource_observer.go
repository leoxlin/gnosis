package main

import (
	"context"
	"errors"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/vault"
)

type observedResource struct {
	title       string
	description string
	revision    string
	origin      vault.Origin
}

// resourceObserver owns one MCP server's observed effective-resource state.
type resourceObserver struct {
	mu        sync.Mutex
	vaultPath string
	server    *mcp.Server
	known     map[string]observedResource
}

func newResourceObserver(vaultPath string) *resourceObserver {
	return &resourceObserver{
		vaultPath: vaultPath,
		known:     map[string]observedResource{},
	}
}

func (observer *resourceObserver) attach(server *mcp.Server) {
	observer.server = server
	current, err := observer.snapshot()
	if err != nil {
		return
	}
	observer.known = current
	for uri, resource := range current {
		server.AddResource(mcpResource(uri, resource), observer.read)
	}
}

func (observer *resourceObserver) subscribe(
	_ context.Context,
	request *mcp.SubscribeRequest,
) error {
	if !vault.IsCanonicalURI(request.Params.URI) {
		return errors.New("resource subscription URI must be a canonical gnosis URI")
	}
	if _, err := vault.ReadResource(observer.vaultPath, request.Params.URI); err != nil {
		if errors.Is(err, vault.ErrPageNotFound) {
			return mcp.ResourceNotFoundError(request.Params.URI)
		}
		return err
	}
	return nil
}

func (observer *resourceObserver) unsubscribe(
	_ context.Context,
	request *mcp.UnsubscribeRequest,
) error {
	if !vault.IsCanonicalURI(request.Params.URI) {
		return errors.New("resource subscription URI must be a canonical gnosis URI")
	}
	return nil
}

func (observer *resourceObserver) observe(ctx context.Context) error {
	observer.mu.Lock()
	current, err := observer.snapshot()
	if err != nil {
		observer.mu.Unlock()
		return err
	}
	previous := observer.known
	observer.known = current
	observer.mu.Unlock()

	for uri, before := range previous {
		after, exists := current[uri]
		if !exists {
			if err := observer.server.ResourceUpdated(
				ctx,
				&mcp.ResourceUpdatedNotificationParams{URI: uri},
			); err != nil {
				return err
			}
			observer.server.RemoveResources(uri)
			continue
		}
		if before.revision != after.revision || before.origin != after.origin {
			if err := observer.server.ResourceUpdated(
				ctx,
				&mcp.ResourceUpdatedNotificationParams{URI: uri},
			); err != nil {
				return err
			}
		}
	}
	for uri, resource := range current {
		if _, existed := previous[uri]; !existed {
			observer.server.AddResource(mcpResource(uri, resource), observer.read)
		}
	}
	return nil
}

func (observer *resourceObserver) snapshot() (map[string]observedResource, error) {
	pages, err := vault.ListPages(observer.vaultPath)
	if err != nil {
		return nil, err
	}
	result := make(map[string]observedResource, len(pages))
	for _, page := range pages {
		result[page.URI] = observedResource{
			title: page.Title, description: page.Description,
			revision: page.Revision, origin: page.Origin,
		}
	}
	return result, nil
}

func (observer *resourceObserver) read(
	_ context.Context,
	request *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	return readMCPPageResource(observer.vaultPath, request.Params.URI)
}

func mcpResource(uri string, resource observedResource) *mcp.Resource {
	return &mcp.Resource{
		URI: uri, Name: uri, Title: resource.title, Description: resource.description,
		MIMEType: vault.ResourceMediaType,
	}
}
