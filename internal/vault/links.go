package vault

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// markdownLinkPattern matches standard markdown links and captures the URL.
	markdownLinkPattern = regexp.MustCompile(`\[[^\]]*\]\(([^\s)#]+)(?:#[^)]*)?\)`)

	// fencedCodeBlockPattern matches fenced code blocks so their contents can be
	// excluded from link extraction.
	fencedCodeBlockPattern = regexp.MustCompile("(?s)```.*?```")
)

// Link represents an internal markdown link found in a document.
type Link struct {
	Raw      string
	Absolute bool
}

// internalLinks extracts all internal markdown links from the given markdown,
// excluding links that appear inside fenced code blocks.
// Internal links are those that do not use a known external scheme and are not
// pure anchors. Absolute bundle-relative links start with "/".
func internalLinks(markdown string) []Link {
	clean := fencedCodeBlockPattern.ReplaceAllString(markdown, "")
	matches := markdownLinkPattern.FindAllStringSubmatch(clean, -1)
	links := make([]Link, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(match[1])
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		if isExternalLink(raw) {
			continue
		}
		links = append(links, Link{
			Raw:      raw,
			Absolute: strings.HasPrefix(raw, "/"),
		})
	}
	return links
}

// isExternalLink reports whether the raw link points outside the bundle.
func isExternalLink(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "ftp", "ftps", "mailto", "file":
		return true
	}
	return false
}

// resolveLink resolves an internal link against the vault root and the file path.
func resolveLink(root, file, raw string) string {
	if strings.HasPrefix(raw, "/") {
		return filepath.Join(root, strings.TrimPrefix(raw, "/"))
	}
	dir := filepath.Dir(file)
	return filepath.Join(dir, raw)
}
