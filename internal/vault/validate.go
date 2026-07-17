package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
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

	vault, err := loadEffectiveVault(root)
	if err != nil {
		return Result{}, err
	}
	resolver := newDocumentResolver(nil)
	effectivePages := make(map[string]*effectivePage)
	pages, err := vault.validationPages()
	if err != nil {
		return Result{}, err
	}
	resolver = newDocumentResolver(pages)
	for _, page := range pages {
		effectivePages[page.path] = page
	}

	result := Result{}
	for _, source := range vault.sources {
		vaultInfo, err := os.Stat(source.path)
		if err != nil {
			return result, err
		}
		if !vaultInfo.IsDir() {
			return result, fmt.Errorf("%s is not a directory", source.path)
		}

		if source.config.LogEnabled() {
			checkRootLog(source.path, &result)
		}
		err = filepath.WalkDir(source.path, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				result.Errors = append(result.Errors, walkErr.Error())
				return nil
			}
			if entry.IsDir() {
				if path != source.path && (ignoredVaultDir(entry.Name()) || exemptVaultDir(source.path, path)) {
					return filepath.SkipDir
				}
				if source.config.IndexEnabled() {
					checkDirectoryIndex(path, &result)
				}
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}

			result.FilesChecked++
			validateFile(source.path, path, source.config, resolver, effectivePages[filepath.Clean(path)], &result)
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

func validateFile(root, path string, config Config, resolver *documentResolver, effective *effectivePage, result *Result) {
	var bytes []byte
	var err error
	if effective != nil {
		bytes = effective.data
	} else {
		bytes, err = os.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			return
		}
	}
	text := string(bytes)

	isReserved := filepath.Base(path) == "index.md" || filepath.Base(path) == "log.md"
	if effective != nil && effective.parseProblem != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, effective.parseProblem))
		return
	}
	var fields frontmatterFields
	var body string
	if isReserved && !strings.HasPrefix(text, "---\n") {
		fields = frontmatterFields{}
		body = text
	} else if effective != nil {
		fields = effective.fields
		body = effective.document.Body
	} else {
		parsed, parseErr := parsePage(bytes)
		err = parseErr
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			return
		}
		fields = parsed.fields
		body = parsed.body
	}

	if !isReserved {
		metadata := pageMetadata{}
		problems := []error{}
		if effective != nil {
			metadata = pageMetadata{
				conceptType: effective.document.Type,
				title:       effective.document.Title,
				description: effective.document.Description,
				tags:        effective.document.Tags,
				aliases:     effective.document.Aliases,
			}
			problems = effective.metadataProblems
		} else {
			metadata, problems = interpretPageMetadata(fields)
		}
		for _, problem := range problems {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, problem))
		}
		for _, field := range []string{"title", "description"} {
			value, fieldErr := optionalPageScalar(fields, field)
			if fieldErr == nil && value == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: missing recommended %q frontmatter", path, field))
			}
		}

		if strings.TrimSpace(body) == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: empty markdown body", path))
		}
		validateRelationships(root, path, fields, config, resolver, result)

		if metadata.conceptType != "" {
			conceptType := metadata.conceptType
			if conceptType == "ConceptType" {
				validateConceptTypeName(path, fields, result)
			}
			if isProcedureType(conceptType) {
				validateProcessRecord(path, fields, body, result)
			}
			if conceptType == DirectiveType {
				for _, problem := range parseDirective(fields, body) {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", path, problem))
				}
			}
		}
	}

	validateReservedName(path, body, result)
	validateLinks(root, path, text, config, resolver, result)
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
	_, problems := parseProcedure(fields, body)
	for _, problem := range problems {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", path, problem))
	}
}

func validateRelationships(root, path string, fields frontmatterFields, config Config, resolver *documentResolver, result *Result) {
	specs, err := relationshipSpecs(fields)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	logicalPath, err := filepath.Rel(root, path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: determine vault-relative path: %v", path, err))
		return
	}
	logicalPath = filepath.ToSlash(logicalPath)
	for index, spec := range specs {
		link, include, err := parseLinkDestination(spec.Target)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid relationships[%d] target: %v", path, index, err))
			continue
		}
		if !include {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d] target %q must be an internal Markdown path", path, index, spec.Target))
			continue
		}
		validateLinkFormat(path, link, config, result)
		resolution, err := resolver.resolve(root, path, logicalPath, link)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d]: %v", path, index, err))
			continue
		}
		if !resolution.document {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: relationships[%d] target %q must be a Markdown path", path, index, spec.Target))
			continue
		}
		if resolution.uri != "" {
			continue
		}
		resolved, err := resolution.physicalExists()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: cannot check relationships[%d] target %s: %v", path, index, spec.Target, err))
			continue
		}
		if !resolved {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved relationships[%d] target %s", path, index, spec.Target))
		}
	}
}

func validateLinks(root, path, text string, config Config, resolver *documentResolver, result *Result) {
	links, err := internalLinks(text)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
		return
	}
	logicalPath, err := filepath.Rel(root, path)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: determine vault-relative path: %v", path, err))
		return
	}
	logicalPath = filepath.ToSlash(logicalPath)
	for _, link := range links {
		validateLinkFormat(path, link, config, result)

		resolution, err := resolver.resolve(root, path, logicalPath, link)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if resolution.uri != "" {
			continue
		}
		resolved, err := resolution.physicalExists()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: cannot check link %s: %v", path, link.Raw, err))
			continue
		}
		if !resolved {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: unresolved internal link %s", path, link.Raw))
		}
	}
}

func validateLinkFormat(path string, link Link, config Config, result *Result) {
	vaultName, _, _ := canonicalGnosisParts(link.URI)
	isLocalURI := vaultName != "" && vaultName == config.Vault.Name
	isAbsolute := link.Absolute || isLocalURI
	preferred := config.LinkFormatValue()

	var msg string
	switch {
	case isAbsolute && preferred == LinkFormatRelative:
		msg = fmt.Sprintf("%s: absolute link %q; this vault is configured to prefer relative links", path, link.Raw)
	case link.URI == "" && !link.Absolute && preferred == LinkFormatAbsolute:
		msg = fmt.Sprintf("%s: relative link %q; this vault is configured to prefer absolute links", path, link.Raw)
	default:
		return
	}
	if config.IsStrict() {
		result.Errors = append(result.Errors, msg)
		return
	}
	result.Warnings = append(result.Warnings, msg)
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
