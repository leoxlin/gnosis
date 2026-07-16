package vault

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const anyVaultAuthority = "_"

// Link represents an internal Markdown destination found in a document.
type Link struct {
	Raw      string
	Path     string
	Absolute bool
	URI      string
}

// documentURI creates the sole canonical identity emitted for a gnosis page.
func documentURI(vaultName, pagePath string) string {
	vaultName = strings.TrimSpace(vaultName)
	if vaultName == "" {
		vaultName = "default"
	}
	u := &url.URL{
		Scheme: "gnosis",
		Host:   vaultName,
		Path:   pathpkg.Join("/", pagePath),
	}
	return u.String()
}

// canonicalGnosisURI accepts only a canonical page identity without a query
// or fragment. It is used by selectors and write targets.
func canonicalGnosisURI(raw string) (string, bool) {
	return parseCanonicalGnosisURI(raw, false)
}

func canonicalGnosisParts(raw string) (string, string, bool) {
	canonical, ok := canonicalGnosisURI(raw)
	if !ok {
		return "", "", false
	}
	u, err := url.Parse(canonical)
	if err != nil {
		return "", "", false
	}
	return u.Host, strings.TrimPrefix(u.Path, "/"), true
}

// canonicalGnosisLink accepts a canonical identity with an optional query or
// fragment and returns the underlying page identity.
func canonicalGnosisLink(raw string) (string, bool) {
	return parseCanonicalGnosisURI(raw, true)
}

func parseCanonicalGnosisURI(raw string, allowSuffix bool) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed != raw {
		return "", false
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme != "gnosis" || u.Host == "" || u.User != nil || u.Opaque != "" {
		return "", false
	}
	if !allowSuffix && (u.RawQuery != "" || u.ForceQuery || u.Fragment != "") {
		return "", false
	}
	escapedPath := strings.TrimPrefix(u.EscapedPath(), "/")
	pagePath, err := url.PathUnescape(escapedPath)
	if err != nil || pagePath == "" || strings.Contains(pagePath, "\\") || pathpkg.IsAbs(pagePath) || pathpkg.Clean(pagePath) != pagePath || strings.HasPrefix(pagePath, "../") {
		return "", false
	}
	canonical := documentURI(u.Host, pagePath)
	expected := canonical
	if allowSuffix {
		expected = withLinkSuffix(canonical, trimmed)
	}
	if expected != trimmed {
		return "", false
	}
	return canonical, true
}

type markdownDestination struct {
	raw        string
	start, end int
}

type parsedMarkdownLinks struct {
	destinations []markdownDestination
}

// internalLinks extracts standard links, reference links, and image
// destinations from Markdown. Goldmark excludes fenced code from the AST nodes
// visited here, and raw HTML is intentionally not interpreted.
func internalLinks(markdown string) ([]Link, error) {
	destinations, _ := markdownASTDestinations([]byte(markdown))
	links := []Link{}
	for _, destination := range destinations {
		link, include, err := parseLinkDestination(destination)
		if err != nil {
			return nil, err
		}
		if include {
			links = append(links, link)
		}
	}
	return links, nil
}

// parseMarkdownLinks locates rewrite spans using the same Goldmark AST
// interpretation as link extraction.
func parseMarkdownLinks(markdown string) parsedMarkdownLinks {
	source := []byte(markdown)
	_, nodes := markdownASTDestinations(source)
	parsed := parsedMarkdownLinks{}
	for index, node := range nodes {
		span, ok := locateMarkdownDestination(source, nodes, index)
		if !ok {
			continue
		}
		span.raw = node.raw
		parsed.destinations = append(parsed.destinations, span)
	}
	sort.Slice(parsed.destinations, func(i, j int) bool {
		return parsed.destinations[i].start < parsed.destinations[j].start
	})
	return parsed
}

type markdownRewriteNode struct {
	raw   string
	start int
}

func markdownASTDestinations(source []byte) ([]string, []markdownRewriteNode) {
	document := goldmark.DefaultParser().Parse(text.NewReader(source))
	links := []string{}
	nodes := []markdownRewriteNode{}
	usedReferences := map[string]struct{}{}
	type referenceDefinition struct {
		label string
		node  markdownRewriteNode
	}
	definitions := []referenceDefinition{}
	definedReferences := map[string]struct{}{}
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := node.(type) {
		case *ast.Link:
			links = append(links, string(node.Destination))
			if node.Reference == nil && len(node.Destination) > 0 {
				nodes = append(nodes, markdownRewriteNode{raw: string(node.Destination), start: node.Pos()})
			} else if node.Reference != nil {
				usedReferences[util.ToLinkReference(node.Reference.Value)] = struct{}{}
			}
		case *ast.Image:
			links = append(links, string(node.Destination))
			if node.Reference == nil && len(node.Destination) > 0 {
				nodes = append(nodes, markdownRewriteNode{raw: string(node.Destination), start: node.Pos()})
			} else if node.Reference != nil {
				usedReferences[util.ToLinkReference(node.Reference.Value)] = struct{}{}
			}
		case *ast.AutoLink:
			if node.AutoLinkType == ast.AutoLinkURL {
				links = append(links, string(node.URL(source)))
			}
		case *ast.LinkReferenceDefinition:
			label := util.ToLinkReference(node.Label)
			if _, defined := definedReferences[label]; defined {
				break
			}
			definedReferences[label] = struct{}{}
			if len(node.Destination) > 0 {
				definitions = append(definitions, referenceDefinition{
					label: label,
					node:  markdownRewriteNode{raw: string(node.Destination), start: node.Pos()},
				})
			}
		}
		return ast.WalkContinue, nil
	})
	for _, definition := range definitions {
		if _, used := usedReferences[definition.label]; used {
			nodes = append(nodes, definition.node)
		}
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].start < nodes[j].start })
	return links, nodes
}

// locateMarkdownDestination asks Goldmark to confirm each candidate source
// span. This keeps Markdown grammar in the parser while preserving every byte
// outside the authored destination, including block-container prefixes.
func locateMarkdownDestination(source []byte, nodes []markdownRewriteNode, index int) (markdownDestination, bool) {
	node := nodes[index]
	if node.start < 0 || node.start >= len(source) || node.raw == "" {
		return markdownDestination{}, false
	}
	limit := len(source)
	raw := []byte(node.raw)
	for search := node.start; search+len(raw) <= limit; {
		offset := bytes.Index(source[search:limit], raw)
		if offset < 0 {
			break
		}
		start := search + offset
		if markdownDestinationProbeMatches(source, start, start+len(raw), index) {
			return markdownDestination{start: start, end: start + len(raw)}, true
		}
		search = start + 1
	}
	return markdownDestination{}, false
}

func markdownDestinationProbeMatches(source []byte, start, end, index int) bool {
	probeDestination := fmt.Sprintf("gnosis-span-probe-%d.md", index)
	probe := make([]byte, 0, len(source)-end+start+len(probeDestination))
	probe = append(probe, source[:start]...)
	probe = append(probe, probeDestination...)
	probe = append(probe, source[end:]...)
	_, nodes := markdownASTDestinations(probe)
	return index < len(nodes) && nodes[index].raw == probeDestination
}

func rewriteMarkdownDestinations(markdown string, rewrite func(string) string) string {
	parsed := parseMarkdownLinks(markdown)
	if len(parsed.destinations) == 0 {
		return markdown
	}

	var output strings.Builder
	last := 0
	for _, destination := range parsed.destinations {
		if destination.start < last {
			continue
		}
		output.WriteString(markdown[last:destination.start])
		replacement := rewrite(destination.raw)
		if replacement == destination.raw {
			output.WriteString(markdown[destination.start:destination.end])
		} else {
			output.WriteString(replacement)
		}
		last = destination.end
	}
	output.WriteString(markdown[last:])
	return output.String()
}

func withLinkSuffix(canonical, raw string) string {
	target, err := url.Parse(canonical)
	if err != nil {
		return canonical
	}
	original, err := url.Parse(markdownDestinationValue(raw))
	if err != nil {
		return canonical
	}
	target.RawQuery = original.RawQuery
	target.ForceQuery = original.ForceQuery
	target.Fragment = original.Fragment
	target.RawFragment = original.RawFragment
	return target.String()
}

func parseLinkDestination(raw string) (Link, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return Link{}, false, nil
	}

	semantic := markdownDestinationValue(raw)
	parsed, err := url.Parse(semantic)
	if err != nil {
		return Link{}, false, fmt.Errorf("invalid destination %q: %w", raw, err)
	}
	if strings.EqualFold(parsed.Scheme, "gnosis") {
		canonical, ok := canonicalGnosisLink(semantic)
		if !ok {
			return Link{}, false, fmt.Errorf("invalid destination %q: must be a canonical gnosis URI", raw)
		}
		return Link{Raw: raw, URI: canonical}, true, nil
	}
	if parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(raw, "//") {
		return Link{}, false, nil
	}
	if parsed.Path == "" {
		return Link{}, false, nil
	}

	path, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return Link{}, false, fmt.Errorf("invalid destination %q: %w", raw, err)
	}
	path = filepath.FromSlash(path)
	return Link{
		Raw:      raw,
		Path:     path,
		Absolute: strings.HasPrefix(path, string(filepath.Separator)),
	}, true, nil
}

func markdownDestinationValue(raw string) string {
	value := util.UnescapePunctuations([]byte(raw))
	value = util.ResolveNumericReferences(value)
	value = util.ResolveEntityNames(value)
	return string(value)
}

type documentResolver struct {
	pagesByURI   map[string]*effectivePage
	uriByPath    map[string]string
	uriByLogical map[string]string
}

type documentResolution struct {
	uri                string
	physicalCandidates []string
	document           bool
}

type resolvedRelationshipLink struct {
	spec       relationshipSpec
	resolution documentResolution
	include    bool
}

type resolvedBodyLink struct {
	link       Link
	resolution documentResolution
}

type resolvedPageLinks struct {
	relationships []resolvedRelationshipLink
	body          []resolvedBodyLink
}

func newDocumentResolver(pages []*effectivePage) *documentResolver {
	resolver := &documentResolver{
		pagesByURI:   make(map[string]*effectivePage, len(pages)),
		uriByPath:    make(map[string]string, len(pages)),
		uriByLogical: make(map[string]string, len(pages)),
	}
	for _, page := range pages {
		resolver.pagesByURI[page.document.URI] = page
		resolver.uriByPath[filepath.Clean(page.path)] = page.document.URI
		resolver.uriByLogical[page.document.Path] = page.document.URI
	}
	return resolver
}

func (r *documentResolver) resolvePage(page *effectivePage, raw string) (documentResolution, bool, error) {
	link, include, err := parseLinkDestination(raw)
	if err != nil || !include {
		return documentResolution{}, include, err
	}
	resolution, err := r.resolve(page.root, page.path, page.document.Path, link)
	return resolution, true, err
}

func (r *documentResolver) resolvePageLinks(page *effectivePage) (resolvedPageLinks, error) {
	result := resolvedPageLinks{}
	specs, err := relationshipSpecs(page.fields)
	if err != nil {
		return result, err
	}
	for _, spec := range specs {
		resolution, include, err := r.resolvePage(page, spec.Target)
		if err != nil {
			return result, fmt.Errorf("relationship target %q: %w", spec.Target, err)
		}
		result.relationships = append(result.relationships, resolvedRelationshipLink{
			spec:       spec,
			resolution: resolution,
			include:    include,
		})
	}

	links, err := internalLinks(page.document.Body)
	if err != nil {
		return result, err
	}
	for _, link := range links {
		resolution, err := r.resolve(page.root, page.path, page.document.Path, link)
		if err != nil {
			return result, fmt.Errorf("body link %q: %w", link.Raw, err)
		}
		result.body = append(result.body, resolvedBodyLink{link: link, resolution: resolution})
	}
	return result, nil
}

func (r *documentResolver) resolve(root, sourcePath, logicalSourcePath string, link Link) (documentResolution, error) {
	resolution := documentResolution{}
	if link.URI != "" {
		resolution.document = true
		targetURI := link.URI
		vaultName, pagePath, _ := canonicalGnosisParts(link.URI)
		if vaultName == anyVaultAuthority {
			targetURI = r.uriByLogical[pagePath]
		}
		if _, exists := r.pagesByURI[targetURI]; exists {
			resolution.uri = targetURI
		}
		return resolution, nil
	}

	extension := strings.ToLower(filepath.Ext(link.Path))
	resolution.document = extension == "" || extension == ".md"
	target, err := linkTarget(root, sourcePath, link)
	if err != nil {
		return documentResolution{}, err
	}
	resolution.physicalCandidates = []string{target}
	if extension == "" {
		resolution.physicalCandidates = append(resolution.physicalCandidates, target+".md")
	}
	if resolution.document {
		for _, candidate := range resolution.physicalCandidates {
			if uri, exists := r.uriByPath[filepath.Clean(candidate)]; exists {
				resolution.uri = uri
				return resolution, nil
			}
		}
		for _, candidate := range logicalDocumentCandidates(logicalSourcePath, link) {
			if uri, exists := r.uriByLogical[candidate]; exists {
				resolution.uri = uri
				return resolution, nil
			}
		}
	}
	return resolution, nil
}

func (r *documentResolver) page(uri string) (*effectivePage, bool) {
	page, exists := r.pagesByURI[uri]
	return page, exists
}

func (r documentResolution) physicalExists() (bool, error) {
	for _, candidate := range r.physicalCandidates {
		_, err := os.Stat(candidate)
		if err == nil {
			return true, nil
		}
		if !os.IsNotExist(err) {
			return false, err
		}
	}
	return false, nil
}

func logicalDocumentCandidates(sourcePath string, link Link) []string {
	logical := filepath.ToSlash(link.Path)
	if link.Absolute {
		logical = strings.TrimPrefix(logical, "/")
	} else {
		logical = pathpkg.Join(pathpkg.Dir(sourcePath), logical)
	}
	logical = pathpkg.Clean(logical)
	if logical == ".." || strings.HasPrefix(logical, "../") {
		return nil
	}
	candidates := []string{logical}
	if filepath.Ext(link.Path) == "" {
		candidates = append(candidates, logical+".md")
	}
	return candidates
}

// resolveLink resolves an internal link against the vault root and source file.
func resolveLink(root, file, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Join(root, strings.TrimPrefix(path, string(filepath.Separator)))
	}
	return filepath.Join(filepath.Dir(file), path)
}

func linkTarget(root, file string, link Link) (string, error) {
	target := filepath.Clean(resolveLink(root, file, link.Path))
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("destination %q escapes the vault root", link.Raw)
	}
	return target, nil
}
