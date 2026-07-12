package vault

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	inlineLinkDestination = regexp.MustCompile(`(!?\[[^\]\r\n]*\]\()(<[^>\r\n]*>|[^()\s]+)([^)\r\n]*\))`)
	referenceDestination  = regexp.MustCompile(`^(\s*\[[^\]\r\n]+\]:\s*)(<[^>\r\n]*>|[^\s\r\n]+)(.*)$`)
)

// renderDocumentLinks replaces resolved internal Markdown document links with
// their canonical gnosis URIs. It preserves external, unresolved, and asset
// destinations exactly as written.
func renderDocumentLinks(page *searchPage, pages []*searchPage) (string, error) {
	pathURIs := make(map[string]string, len(pages))
	uriPages := make(map[string]*searchPage, len(pages))
	for _, candidate := range pages {
		pathURIs[candidate.path] = candidate.document.URI
		uriPages[candidate.document.URI] = candidate
	}

	rewrite := func(raw string) string {
		destination := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(raw), "<"), ">")
		uri := resolveDocumentTarget(page, destination, pathURIs, uriPages)
		if uri == "" {
			return raw
		}
		target, exists := uriPages[uri]
		if !exists {
			return raw
		}
		return withLinkSuffix(target.document.URI, destination)
	}

	return rewriteMarkdownDestinations(string(page.data), rewrite), nil
}

func withLinkSuffix(canonical, raw string) string {
	target, err := url.Parse(canonical)
	if err != nil {
		return canonical
	}
	original, err := url.Parse(raw)
	if err != nil {
		return canonical
	}
	target.RawQuery = original.RawQuery
	target.ForceQuery = original.ForceQuery
	target.Fragment = original.Fragment
	return target.String()
}

func rewriteMarkdownDestinations(markdown string, rewrite func(string) string) string {
	var output strings.Builder
	inFence := false
	for _, line := range strings.SplitAfter(markdown, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			output.WriteString(line)
			inFence = !inFence
			continue
		}
		if inFence {
			output.WriteString(line)
			continue
		}

		ending := ""
		if strings.HasSuffix(line, "\n") {
			line = strings.TrimSuffix(line, "\n")
			ending = "\n"
		}
		line = referenceDestination.ReplaceAllStringFunc(line, func(match string) string {
			parts := referenceDestination.FindStringSubmatch(match)
			return parts[1] + rewrite(parts[2]) + parts[3]
		})
		output.WriteString(rewriteInlineLinkDestinations(line, rewrite))
		output.WriteString(ending)
	}
	return output.String()
}

func rewriteInlineLinkDestinations(line string, rewrite func(string) string) string {
	parts := strings.Split(line, "`")
	for index := 0; index < len(parts); index += 2 {
		parts[index] = inlineLinkDestination.ReplaceAllStringFunc(parts[index], func(match string) string {
			groups := inlineLinkDestination.FindStringSubmatch(match)
			return groups[1] + rewrite(groups[2]) + groups[3]
		})
	}
	return strings.Join(parts, "`")
}
