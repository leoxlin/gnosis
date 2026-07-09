package vault

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Link represents an internal Markdown destination found in a document.
type Link struct {
	Raw      string
	Path     string
	Absolute bool
}

// internalLinks extracts standard links, reference links, and image
// destinations from Markdown. Goldmark excludes fenced code from the AST nodes
// visited here, and raw HTML is intentionally not interpreted.
func internalLinks(markdown string) ([]Link, error) {
	source := []byte(markdown)
	document := goldmark.DefaultParser().Parse(text.NewReader(source))
	links := []Link{}
	err := ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		var destination []byte
		switch node := node.(type) {
		case *ast.Link:
			destination = node.Destination
		case *ast.Image:
			destination = node.Destination
		default:
			return ast.WalkContinue, nil
		}

		link, include, err := parseLinkDestination(string(destination))
		if err != nil {
			return ast.WalkStop, err
		}
		if include {
			links = append(links, link)
		}
		return ast.WalkContinue, nil
	})
	return links, err
}

func parseLinkDestination(raw string) (Link, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return Link{}, false, nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return Link{}, false, fmt.Errorf("invalid destination %q: %w", raw, err)
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
