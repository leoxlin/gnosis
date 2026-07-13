package vault

// renderDocumentLinks replaces resolved internal Markdown document links with
// their canonical gnosis URIs. It preserves external, unresolved, and asset
// destinations exactly as written.
func renderDocumentLinks(page *effectivePage, pages []*effectivePage) (string, error) {
	resolver := newDocumentResolver(pages)
	resolved, err := resolver.resolvePageLinks(page)
	if err != nil {
		return "", err
	}
	byRawDestination := make(map[string]documentResolution, len(resolved.body))
	for _, body := range resolved.body {
		byRawDestination[body.link.Raw] = body.resolution
	}

	rewrite := func(raw string) string {
		resolution, exists := byRawDestination[raw]
		if !exists || resolution.uri == "" {
			return raw
		}
		target, exists := resolver.page(resolution.uri)
		if !exists {
			return raw
		}
		return withLinkSuffix(target.document.URI, raw)
	}

	return rewriteMarkdownDestinations(string(page.data), rewrite), nil
}
