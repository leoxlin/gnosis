package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v4"
)

type frontmatterFields map[string]any

var yamlFrontmatter = frontmatter.NewFormat("---", "---", yaml.Unmarshal)

func frontmatterError(err error) error {
	if errors.Is(err, frontmatter.ErrNotFound) {
		return fmt.Errorf("missing YAML frontmatter")
	}
	if err != nil {
		return fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	return nil
}

func frontmatterScalar(fields frontmatterFields, key string) (string, bool) {
	value, exists := fields[key]
	if !exists {
		return "", false
	}
	valueString, scalar := value.(string)
	return valueString, scalar
}

func frontmatterScalars(fields frontmatterFields, key string) ([]string, bool) {
	value, exists := fields[key]
	if !exists {
		return nil, true
	}
	if value == nil {
		return nil, true
	}
	switch value := value.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, true
		}
		return []string{strings.TrimSpace(value)}, true
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			if item == nil {
				continue
			}
			itemString, scalar := item.(string)
			if !scalar {
				return nil, false
			}
			if strings.TrimSpace(itemString) != "" {
				values = append(values, strings.TrimSpace(itemString))
			}
		}
		return values, true
	default:
		return nil, false
	}
}

func frontmatterNonEmpty(fields frontmatterFields, key string) bool {
	value := fields[key]
	if value == nil {
		return false
	}
	if valueString, scalar := value.(string); scalar {
		return strings.TrimSpace(valueString) != ""
	}
	if values, sequence := value.([]any); sequence {
		return len(values) > 0
	}
	return true
}

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

	resolution, err := ResolveConfig(root)
	if err != nil {
		return Result{}, err
	}
	effectiveDocumentPaths := make(map[string]struct{})
	searchSource := &SearchSource{resolution: resolution}
	if pages, pagesErr := searchSource.pages(); pagesErr == nil {
		for _, page := range pages {
			// Relative paths are retained only to resolve authored relative links.
			effectiveDocumentPaths[page.document.Path] = struct{}{}
			effectiveDocumentPaths[page.document.URI] = struct{}{}
		}
	}

	result := Result{}
	for _, source := range resolution.Sources {
		vaultInfo, err := os.Stat(source.Path)
		if err != nil {
			return result, err
		}
		if !vaultInfo.IsDir() {
			return result, fmt.Errorf("%s is not a directory", source.Path)
		}

		if source.Config.LogEnabled() {
			checkRootLog(source.Path, &result)
		}
		err = filepath.WalkDir(source.Path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				result.Errors = append(result.Errors, walkErr.Error())
				return nil
			}
			if entry.IsDir() {
				if path != source.Path && ignoredVaultDir(entry.Name()) {
					return filepath.SkipDir
				}
				if source.Config.IndexEnabled() {
					checkDirectoryIndex(path, &result)
				}
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			result.FilesChecked++
			validateFile(source.Path, path, source.Config, effectiveDocumentPaths, &result)
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

func validateFile(root, path string, config Config, effectiveDocumentPaths map[string]struct{}, result *Result) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	text := string(bytes)

	isReserved := filepath.Base(path) == "index.md" || filepath.Base(path) == "log.md"
	var fields frontmatterFields
	var body string
	if isReserved && !strings.HasPrefix(text, "---\n") {
		fields = frontmatterFields{}
		body = text
	} else {
		var bodyBytes []byte
		bodyBytes, err = frontmatter.MustParse(strings.NewReader(text), &fields, yamlFrontmatter)
		body = string(bodyBytes)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, frontmatterError(err)))
			return
		}
	}

	if !isReserved {
		if value, scalar := frontmatterScalar(fields, "type"); !scalar {
			if frontmatterNonEmpty(fields, "type") {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar", path, "type"))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: missing non-empty %q frontmatter", path, "type"))
			}
		} else if strings.TrimSpace(value) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing non-empty %q frontmatter", path, "type"))
		}
		for _, field := range []string{"title", "description"} {
			value, scalar := frontmatterScalar(fields, field)
			if !scalar && frontmatterNonEmpty(fields, field) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar", path, field))
			} else if strings.TrimSpace(value) == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: missing recommended %q frontmatter", path, field))
			}
		}

		if strings.TrimSpace(body) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: empty markdown body", path))
		}
		validateRelationships(root, path, fields, effectiveDocumentPaths, result)

		if conceptType, scalar := frontmatterScalar(fields, "type"); scalar {
			conceptType = strings.TrimSpace(conceptType)
			if conceptType == "ConceptType" {
				validateConceptTypeName(path, fields, result)
			}
			if isProcessType(conceptType) {
				validateProcessRecord(path, fields, body, result)
			}
		}
	}

	validateReservedName(path, body, result)
	validateLinks(root, path, text, config, effectiveDocumentPaths, result)
}

func validateConceptTypeName(path string, fields frontmatterFields, result *Result) {
	title, scalar := frontmatterScalar(fields, "title")
	title = strings.TrimSpace(title)
	if !scalar || title == "" || isTypeName(title) {
		return
	}
	result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter \"title\" %q must use the TypeName convention", path, title))
}

func isTypeName(value string) bool {
	for index, character := range value {
		if index == 0 {
			if !unicode.IsUpper(character) {
				return false
			}
			continue
		}
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			return false
		}
	}
	return value != ""
}

func validateProcessRecord(path string, fields frontmatterFields, body string, result *Result) {
	_, missing, duplicates := parseProcessSections(body)
	for _, section := range missing {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: missing required section %q", path, section))
	}
	for _, section := range duplicates {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: duplicate process section %q", path, section))
	}
	useWhen, valid := frontmatterScalars(fields, "use_when")
	if !valid {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar or sequence of scalars", path, "use_when"))
	} else if len(useWhen) == 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: process requires at least one non-empty %q frontmatter value", path, "use_when"))
	}
	if description, scalar := frontmatterScalar(fields, "description"); !scalar || strings.TrimSpace(description) == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: process requires non-empty %q frontmatter", path, "description"))
	}

	if invocation, scalar := frontmatterScalar(fields, "invocation"); scalar {
		invocation = strings.TrimSpace(invocation)
		if invocation != "" && !validProcessInvocation(invocation) {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be %q or %q", path, "invocation", "model", "explicit"))
		}
	} else if _, exists := fields["invocation"]; exists {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: frontmatter %q must be a scalar", path, "invocation"))
	}

	for _, field := range []string{"effects", "relationships"} {
		if _, exists := fields[field]; exists {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: procedure frontmatter must not contain %q", path, field))
		}
	}
}

func validateRelationships(root, path string, fields frontmatterFields, effectiveDocumentPaths map[string]struct{}, result *Result) {
	specs, err := relationshipSpecs(fields)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	for index, spec := range specs {
		if _, exists := effectiveDocumentPaths[strings.TrimSpace(spec.Target)]; exists {
			continue
		}
		link, include, err := parseLinkDestination(spec.Target)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid relationships[%d] target: %v", path, index, err))
			continue
		}
		if !include {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d] target %q must be an internal Markdown path", path, index, spec.Target))
			continue
		}
		extension := strings.ToLower(filepath.Ext(link.Path))
		if extension != "" && extension != ".md" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d] target %q must be a Markdown path", path, index, spec.Target))
			continue
		}
		target, err := linkTarget(root, path, link)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d]: %v", path, index, err))
			continue
		}
		candidates := []string{target}
		if extension == "" {
			candidates = append(candidates, target+".md")
		}
		resolved := resolvesPhysicalCandidate(candidates)
		if !resolved {
			resolved = resolvesEffectiveCandidate(root, path, link, effectiveDocumentPaths)
		}
		if !resolved {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved relationships[%d] target %s", path, index, spec.Target))
		}
	}
}

func resolvesPhysicalCandidate(candidates []string) bool {
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

func resolvesEffectiveCandidate(root, sourcePath string, link Link, effectiveDocumentPaths map[string]struct{}) bool {
	pagePath, err := filepath.Rel(root, sourcePath)
	if err != nil {
		return false
	}
	for _, candidate := range logicalDocumentCandidates(filepath.ToSlash(pagePath), link) {
		if _, exists := effectiveDocumentPaths[candidate]; exists {
			return true
		}
	}
	return false
}

func validateLinks(root, path, text string, config Config, effectiveDocumentPaths map[string]struct{}, result *Result) {
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
				if !resolvesEffectiveCandidate(root, path, link, effectiveDocumentPaths) {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved internal link %s", path, link.Raw))
				}
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: cannot check link %s: %v", path, link.Raw, err))
			}
		}
	}
}

func checkRootLog(root string, result *Result) {
	path := filepath.Join(root, "log.md")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing reserved root file", path))
		} else {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
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
