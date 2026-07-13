package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v4"
)

type frontmatterFields map[string]any

var yamlFrontmatter = frontmatter.NewFormat("---", "---", yaml.Unmarshal)

type parsedPage struct {
	fields frontmatterFields
	body   string
}

type pageMetadata struct {
	conceptType string
	title       string
	description string
	tags        []string
	aliases     []string
}

type relationshipSpec struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

func reservedPageName(name string) bool {
	return name == "index.md" || name == "log.md"
}

// parsePage is the single YAML interpretation shared by every Concept-record
// consumer. Effective pages retain the result so downstream modules do not
// parse the same authored record again.
func parsePage(data []byte) (parsedPage, error) {
	fields := frontmatterFields{}
	body, err := frontmatter.MustParse(bytes.NewReader(data), &fields, yamlFrontmatter)
	if err != nil {
		return parsedPage{}, frontmatterError(err)
	}
	return parsedPage{fields: fields, body: string(body)}, nil
}

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
	if !exists || value == nil {
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

func requiredPageScalar(fields frontmatterFields, key string) (string, error) {
	value, scalar := frontmatterScalar(fields, key)
	if !scalar {
		if _, exists := fields[key]; exists {
			return "", fmt.Errorf("frontmatter %q must be a scalar", key)
		}
		return "", fmt.Errorf("missing non-empty %q frontmatter", key)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("missing non-empty %q frontmatter", key)
	}
	return value, nil
}

func optionalPageScalar(fields frontmatterFields, key string) (string, error) {
	value, scalar := frontmatterScalar(fields, key)
	if scalar {
		return strings.TrimSpace(value), nil
	}
	if _, exists := fields[key]; exists {
		return "", fmt.Errorf("frontmatter %q must be a scalar", key)
	}
	return "", nil
}

func pageScalars(fields frontmatterFields, key string) ([]string, error) {
	values, valid := frontmatterScalars(fields, key)
	if !valid {
		return nil, fmt.Errorf("frontmatter %q must be a scalar or sequence of scalars", key)
	}
	return values, nil
}

func interpretPageMetadata(fields frontmatterFields) (pageMetadata, []error) {
	metadata := pageMetadata{}
	problems := []error{}
	var err error

	metadata.conceptType, err = requiredPageScalar(fields, "type")
	if err != nil {
		problems = append(problems, err)
	}
	metadata.title, err = optionalPageScalar(fields, "title")
	if err != nil {
		problems = append(problems, err)
	}
	metadata.description, err = optionalPageScalar(fields, "description")
	if err != nil {
		problems = append(problems, err)
	} else if metadata.description == "" {
		metadata.description, err = optionalPageScalar(fields, "summary")
		if err != nil {
			problems = append(problems, err)
		}
	}
	metadata.tags, err = pageScalars(fields, "tags")
	if err != nil {
		problems = append(problems, err)
	}
	metadata.aliases, err = pageScalars(fields, "aliases")
	if err != nil {
		problems = append(problems, err)
	}
	return metadata, problems
}

func relationshipSpecs(fields frontmatterFields) ([]relationshipSpec, error) {
	value, exists := fields["relationships"]
	if !exists || value == nil {
		return nil, nil
	}
	var specs []relationshipSpec
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of type and target mappings: %w", "relationships", err)
	}
	if err := yaml.Unmarshal(encoded, &specs); err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of type and target mappings: %w", "relationships", err)
	}
	for index, spec := range specs {
		if strings.TrimSpace(spec.Type) == "" {
			return nil, fmt.Errorf("relationships[%d] missing non-empty %q", index, "type")
		}
		if strings.TrimSpace(spec.Target) == "" {
			return nil, fmt.Errorf("relationships[%d] missing non-empty %q", index, "target")
		}
	}
	return specs, nil
}

// effectivePage is one parsed Concept record selected into the composed view.
type effectivePage struct {
	root             string
	path             string
	document         Document
	data             []byte
	fields           frontmatterFields
	metadataProblems []error
	parseProblem     error
}

func newTolerantEffectivePage(root, path string, data []byte, origin Origin) (*effectivePage, error) {
	parsed, err := parsePage(data)
	if err != nil {
		page, identityErr := newEffectivePageIdentity(root, path, data, origin)
		if identityErr != nil {
			return nil, identityErr
		}
		page.parseProblem = err
		return page, nil
	}
	metadata, problems := interpretPageMetadata(parsed.fields)
	page, err := buildEffectivePage(root, path, data, origin, parsed, metadata)
	if err != nil {
		return nil, err
	}
	page.metadataProblems = problems
	return page, nil
}

func newEffectivePage(root, path string, data []byte, origin Origin) (*effectivePage, error) {
	parsed, err := parsePage(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	metadata, problems := interpretPageMetadata(parsed.fields)
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s: %w", path, problems[0])
	}
	return buildEffectivePage(root, path, data, origin, parsed, metadata)
}

func buildEffectivePage(root, path string, data []byte, origin Origin, parsed parsedPage, metadata pageMetadata) (*effectivePage, error) {
	if metadata.title == "" {
		metadata.title = firstHeading(parsed.body)
	}
	if metadata.title == "" {
		metadata.title = humanizeName(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	page, err := newEffectivePageIdentity(root, path, data, origin)
	if err != nil {
		return nil, err
	}
	page.fields = parsed.fields
	page.document.Title = metadata.title
	page.document.Description = metadata.description
	page.document.Type = metadata.conceptType
	page.document.Aliases = metadata.aliases
	page.document.Tags = metadata.tags
	page.document.Body = parsed.body
	return page, nil
}

func newEffectivePageIdentity(root, path string, data []byte, origin Origin) (*effectivePage, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}

	return &effectivePage{
		root: root,
		path: filepath.Clean(path),
		data: data,
		document: Document{
			Path:     filepath.ToSlash(relative),
			URI:      documentURI(origin.Vault, filepath.ToSlash(relative)),
			Links:    []string{},
			Edges:    []Edge{},
			Origin:   origin,
			Revision: documentRevision(data),
		},
	}, nil
}

func (p *effectivePage) authoredRecord() map[string]any {
	record := make(map[string]any, len(p.fields)+1)
	for key, value := range p.fields {
		record[key] = value
	}
	record["uri"] = p.document.URI
	return record
}

func documentRevision(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
