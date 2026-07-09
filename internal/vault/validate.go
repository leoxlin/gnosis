package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Result is the aggregate output of a vault validation run.
type Result struct {
	FilesChecked int
	Errors       []string
	Warnings     []string
}

// Validate checks markdown frontmatter and absolute internal markdown links.
func Validate(root string) (Result, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("%s is not a directory", root)
	}

	result := Result{}
	checkRootFiles(root, &result)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Errors = append(result.Errors, walkErr.Error())
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".obsidian":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		result.FilesChecked++
		validateFile(root, path, &result)
		return nil
	})
	if err != nil {
		return result, err
	}

	sort.Strings(result.Errors)
	sort.Strings(result.Warnings)
	return result, nil
}

func validateFile(root, path string, result *Result) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	text := string(bytes)
	fields, body, err := parseFrontmatter(text)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}

	required := []string{"type"}
	for _, field := range required {
		if strings.TrimSpace(fields[field]) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing non-empty %q frontmatter", path, field))
		}
	}
	for _, field := range []string{"title", "description", "tags", "timestamp"} {
		if strings.TrimSpace(fields[field]) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: missing recommended %q frontmatter", path, field))
		}
	}

	if strings.TrimSpace(body) == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: empty markdown body", path))
	}
	validateReservedName(path, fields, body, result)

	for _, link := range absoluteInternalLinks(text) {
		target := filepath.Join(root, strings.TrimPrefix(link, "/"))
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved internal link %s", path, link))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: cannot check link %s: %v", path, link, err))
			}
		}
	}
}

func checkRootFiles(root string, result *Result) {
	for _, rel := range []string{"index.md", "log.md"} {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: missing reserved root file", path))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			}
		}
	}
}

func validateReservedName(path string, fields Frontmatter, body string, result *Result) {
	switch filepath.Base(path) {
	case "index.md":
		if !strings.Contains(body, "# ") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: index file should include a heading", path))
		}
		if len(absoluteInternalLinks(body)) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: index file should include navigation links", path))
		}
	case "log.md":
		if fields["type"] != "Log" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: log file should use type Log", path))
		}
		if !strings.Contains(body, "## ") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: log file should include dated sections", path))
		}
		if !strings.Contains(body, "Actor:") || !strings.Contains(body, "Action:") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: log file should include Actor and Action entries", path))
		}
	}
}
