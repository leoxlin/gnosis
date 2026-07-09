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

// Validate checks markdown frontmatter and internal markdown links according
// to the vault's gnosis.toml configuration.
func Validate(root string) (Result, error) {
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return Result{}, err
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("%s is not a directory", root)
	}

	config, vaultRoots, err := LoadConfig(root)
	if err != nil {
		return Result{}, err
	}

	result := Result{}
	for _, vaultRoot := range vaultRoots {
		vaultInfo, err := os.Stat(vaultRoot)
		if err != nil {
			return result, err
		}
		if !vaultInfo.IsDir() {
			return result, fmt.Errorf("%s is not a directory", vaultRoot)
		}

		checkRootFiles(vaultRoot, &result)
		err = filepath.WalkDir(vaultRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				result.Errors = append(result.Errors, walkErr.Error())
				return nil
			}
			if entry.IsDir() {
				if path != vaultRoot && ignoredVaultDir(entry.Name()) {
					return filepath.SkipDir
				}
				checkDirectoryIndex(path, &result)
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			result.FilesChecked++
			validateFile(vaultRoot, path, config, &result)
			return nil
		})
		if err != nil {
			return result, err
		}
	}

	sort.Strings(result.Errors)
	sort.Strings(result.Warnings)
	return result, nil
}

func validateFile(root, path string, config Config, result *Result) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	text := string(bytes)

	isReserved := filepath.Base(path) == "index.md" || filepath.Base(path) == "log.md"
	var fields Frontmatter
	var body string
	if isReserved && !strings.HasPrefix(text, "---\n") {
		fields = Frontmatter{}
		body = text
	} else {
		fields, body, err = parseFrontmatter(text)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			return
		}
	}

	if !isReserved {
		if value, scalar := fields.scalar("type"); !scalar {
			if fields.nonEmpty("type") {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar", path, "type"))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: missing non-empty %q frontmatter", path, "type"))
			}
		} else if strings.TrimSpace(value) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing non-empty %q frontmatter", path, "type"))
		}
		for _, field := range []string{"title", "description", "timestamp"} {
			value, scalar := fields.scalar(field)
			if !scalar && fields.nonEmpty(field) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar", path, field))
			} else if strings.TrimSpace(value) == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: missing recommended %q frontmatter", path, field))
			}
		}
		if !fields.nonEmpty("tags") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: missing recommended %q frontmatter", path, "tags"))
		}

		if strings.TrimSpace(body) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: empty markdown body", path))
		}
	}

	validateReservedName(path, body, result)
	validateLinks(root, path, text, config, result)
}

func validateLinks(root, path, text string, config Config, result *Result) {
	preferred := config.LinkFormatValue()
	links, err := internalLinks(text)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	for _, link := range links {
		if link.Absolute && preferred == LinkFormatRelative {
			msg := fmt.Sprintf("%s: absolute link %q; this vault is configured to prefer relative links", path, link.Raw)
			if config.IsStrict() {
				result.Errors = append(result.Errors, msg)
			} else {
				result.Warnings = append(result.Warnings, msg)
			}
		}
		if !link.Absolute && preferred == LinkFormatAbsolute {
			msg := fmt.Sprintf("%s: relative link %q; this vault is configured to prefer absolute links", path, link.Raw)
			if config.IsStrict() {
				result.Errors = append(result.Errors, msg)
			} else {
				result.Warnings = append(result.Warnings, msg)
			}
		}

		target, err := linkTarget(root, path, link)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved internal link %s", path, link.Raw))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: cannot check link %s: %v", path, link.Raw, err))
			}
		}
	}
}

func checkRootFiles(root string, result *Result) {
	for _, rel := range []string{"log.md"} {
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

func checkDirectoryIndex(dir string, result *Result) {
	path := filepath.Join(dir, "index.md")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing directory index file", path))
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		}
	}
}

func validateReservedName(path string, body string, result *Result) {
	switch filepath.Base(path) {
	case "index.md":
		if !strings.Contains(body, "# ") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: index file should include a heading", path))
		}
		links, _ := internalLinks(body)
		if len(links) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: index file should include navigation links", path))
		}
	case "log.md":
		if !strings.Contains(body, "## ") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: log file should include dated sections", path))
		}
	}
}
