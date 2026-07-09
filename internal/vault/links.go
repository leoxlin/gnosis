package vault

import (
	"regexp"
	"strings"
)

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\((/[^)#]+)(#[^)]+)?\)`)

func absoluteInternalLinks(markdown string) []string {
	matches := markdownLinkPattern.FindAllStringSubmatch(markdown, -1)
	links := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		link := strings.TrimSpace(match[1])
		if link != "" {
			links = append(links, link)
		}
	}
	return links
}
