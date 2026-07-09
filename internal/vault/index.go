package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IndexOptions controls how generated index files are written.
type IndexOptions struct {
	Overwrite bool
}

type indexEntry struct {
	Title       string
	Link        string
	Description string
}

// GenerateIndexes creates index.md files for every directory in the vault.
func GenerateIndexes(root string, options IndexOptions) ([]string, error) {
	root = filepath.Clean(root)
	dirs, err := indexDirectories(root)
	if err != nil {
		return nil, err
	}

	written := []string{}
	for _, dir := range dirs {
		path := filepath.Join(dir, "index.md")
		if !options.Overwrite {
			if _, err := os.Stat(path); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return written, err
			}
		}

		content, err := renderIndex(root, dir)
		if err != nil {
			return written, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return written, err
		}
		written = append(written, path)
	}

	return written, nil
}

func indexDirectories(root string) ([]string, error) {
	dirs := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root && ignoredVaultDir(entry.Name()) {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	return dirs, nil
}

func renderIndex(root, dir string) (string, error) {
	folders, pages, err := indexEntries(root, dir)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if rel == "." {
		buf.WriteString("---\n")
		buf.WriteString("okf_version: \"0.1\"\n")
		buf.WriteString("---\n\n")
		buf.WriteString("# Knowledge Bundle\n\n")
		for _, folder := range folders {
			writeIndexBullet(&buf, folder)
		}
		return buf.String(), nil
	}

	fmt.Fprintf(&buf, "# %s\n\n", directoryTitle(rel))
	writeParentIndexLink(&buf, dir)

	if len(folders) > 0 {
		buf.WriteString("## Folders\n\n")
		for _, folder := range folders {
			writeIndexBullet(&buf, folder)
		}
		buf.WriteString("\n")
	}

	if len(pages) > 0 {
		buf.WriteString("## Pages\n\n")
		for _, page := range pages {
			writeIndexBullet(&buf, page)
		}
		buf.WriteString("\n")
	}

	if len(folders) == 0 && len(pages) == 0 {
		buf.WriteString("No pages yet.\n")
	}

	return buf.String(), nil
}

func indexEntries(root, dir string) ([]indexEntry, []indexEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	folders := []indexEntry{}
	pages := []indexEntry{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			if ignoredVaultDir(name) {
				continue
			}
			child := filepath.Join(dir, name)
			link, err := relativeMarkdownLink(dir, filepath.Join(child, "index.md"))
			if err != nil {
				return nil, nil, err
			}
			childRel, err := filepath.Rel(root, child)
			if err != nil {
				return nil, nil, err
			}
			folders = append(folders, indexEntry{
				Title: directoryTitle(childRel),
				Link:  link,
			})
			continue
		}

		if filepath.Ext(name) != ".md" || name == "index.md" {
			continue
		}
		path := filepath.Join(dir, name)
		title, description := markdownTitleAndDescription(path)
		if title == "" {
			title = humanizeName(strings.TrimSuffix(name, filepath.Ext(name)))
		}
		link, err := relativeMarkdownLink(dir, path)
		if err != nil {
			return nil, nil, err
		}
		pages = append(pages, indexEntry{
			Title:       title,
			Link:        link,
			Description: description,
		})
	}

	return folders, pages, nil
}

func ignoredVaultDir(name string) bool {
	switch name {
	case ".git", ".obsidian":
		return true
	default:
		return false
	}
}

func writeParentIndexLink(buf *strings.Builder, dir string) {
	link, err := relativeMarkdownLink(dir, filepath.Join(filepath.Dir(dir), "index.md"))
	if err != nil {
		return
	}
	fmt.Fprintf(buf, "[Parent Index](%s)\n\n", link)
}

func writeIndexBullet(buf *strings.Builder, entry indexEntry) {
	fmt.Fprintf(buf, "* [%s](%s)", entry.Title, entry.Link)
	if entry.Description != "" {
		fmt.Fprintf(buf, " - %s", entry.Description)
	}
	buf.WriteString("\n")
}

func relativeMarkdownLink(fromDir, target string) (string, error) {
	rel, err := filepath.Rel(fromDir, target)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func directoryTitle(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		words = append(words, humanizeName(part))
	}
	return strings.Join(words, " ")
}

func markdownTitleAndDescription(path string) (string, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	text := string(data)
	if strings.HasPrefix(text, "---\n") {
		fields, body, err := parseFrontmatter(text)
		if err == nil {
			title := strings.TrimSpace(fields["title"])
			if title == "" {
				title = firstHeading(body)
			}
			return title, strings.TrimSpace(fields["description"])
		}
	}
	return firstHeading(text), ""
}

func firstHeading(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func humanizeName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
