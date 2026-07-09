package vault

import (
	"bufio"
	"fmt"
	"strings"
)

// Frontmatter stores the small YAML subset Gnosis validators need.
type Frontmatter map[string]string

func parseFrontmatter(markdown string) (Frontmatter, string, error) {
	if !strings.HasPrefix(markdown, "---\n") {
		return nil, "", fmt.Errorf("missing YAML frontmatter")
	}

	rest := markdown[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("unterminated YAML frontmatter")
	}

	block := rest[:end]
	body := strings.TrimPrefix(rest[end+len("\n---"):], "\n")
	fields := Frontmatter{}
	scanner := bufio.NewScanner(strings.NewReader(block))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "-") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, "", fmt.Errorf("invalid frontmatter line %q", line)
		}
		fields[strings.TrimSpace(key)] = cleanScalar(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	return fields, body, nil
}

func cleanScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}
