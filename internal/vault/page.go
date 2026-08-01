package vault

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

type frontmatterFields map[string]any

var errMissingYAMLFrontmatter = errors.New("missing YAML frontmatter")

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
	maintenance []MaintenanceAnnotation
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
	data, found := bytes.CutPrefix(data, []byte("---\n"))
	if !found {
		return parsedPage{}, errMissingYAMLFrontmatter
	}
	header, body, found := bytes.Cut(data, []byte("\n---\n"))
	if !found {
		return parsedPage{}, errors.New("invalid YAML frontmatter: missing closing delimiter")
	}
	fields := frontmatterFields{}
	if err := yaml.Unmarshal(header, &fields); err != nil {
		return parsedPage{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	return parsedPage{fields: fields, body: string(body)}, nil
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
	metadata.maintenance, err = maintenanceAnnotations(fields)
	if err != nil {
		problems = append(problems, err)
	}
	return metadata, problems
}

func maintenanceAnnotations(fields frontmatterFields) ([]MaintenanceAnnotation, error) {
	value, exists := fields["maintenance"]
	if !exists || value == nil {
		return nil, nil
	}
	if _, ok := value.([]any); !ok {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of mappings", "maintenance")
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of mappings", "maintenance")
	}
	var items []map[string]any
	if err := yaml.Unmarshal(encoded, &items); err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of mappings", "maintenance")
	}
	annotations := make([]MaintenanceAnnotation, 0, len(items))
	for index, fields := range items {
		for key := range fields {
			switch key {
			case "kind", "reason", "observed_at", "author", "target":
			default:
				return nil, fmt.Errorf("maintenance[%d] has unknown field %q", index, key)
			}
		}
		required := func(key string) (string, error) {
			raw, exists := fields[key]
			value, ok := raw.(string)
			if !exists || !ok || strings.TrimSpace(value) == "" {
				return "", fmt.Errorf("maintenance[%d] missing non-empty %q", index, key)
			}
			return strings.TrimSpace(value), nil
		}
		kind, err := required("kind")
		if err != nil {
			return nil, err
		}
		if kind != "stale" && kind != "incorrect" && kind != "duplicate" {
			return nil, fmt.Errorf("maintenance[%d] kind %q must be stale, incorrect, or duplicate", index, kind)
		}
		reason, err := required("reason")
		if err != nil {
			return nil, err
		}
		observedAt := metadataTime(fields, "observed_at")
		if observedAt == "" {
			return nil, fmt.Errorf("maintenance[%d] missing valid RFC3339 %q", index, "observed_at")
		}
		observed, err := time.Parse(time.RFC3339, observedAt)
		if err != nil {
			return nil, fmt.Errorf("maintenance[%d] %q must be RFC3339", index, "observed_at")
		}
		annotation := MaintenanceAnnotation{
			Kind: kind, Reason: reason, ObservedAt: observed.UTC().Format(time.RFC3339Nano),
		}
		if raw, exists := fields["author"]; exists {
			author, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("maintenance[%d] %q must be a scalar", index, "author")
			}
			annotation.Author = strings.TrimSpace(author)
		}
		rawTarget, hasTarget := fields["target"]
		target, targetScalar := rawTarget.(string)
		target = strings.TrimSpace(target)
		if kind == "duplicate" {
			if !hasTarget || !targetScalar || target == "" {
				return nil, fmt.Errorf("maintenance[%d] duplicate missing non-empty %q", index, "target")
			}
			if !IsCanonicalURI(target) {
				return nil, fmt.Errorf("maintenance[%d] duplicate target %q must be a canonical gnosis URI", index, target)
			}
			annotation.Target = &MaintenanceTarget{Authored: target}
		} else if hasTarget {
			return nil, fmt.Errorf("maintenance[%d] kind %q must not have a target", index, kind)
		}
		annotations = append(annotations, annotation)
	}
	return annotations, nil
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
	page.document.Metadata = make(map[string]any, len(parsed.fields))
	for key, value := range parsed.fields {
		page.document.Metadata[key] = value
	}
	page.document.Aliases = metadata.aliases
	page.document.Tags = metadata.tags
	page.document.Maintenance = metadata.maintenance
	page.document.Body = parsed.body
	page.document.Trust = initialTrust(page.document)
	return page, nil
}

func newEffectivePageIdentity(root, path string, data []byte, origin Origin) (*effectivePage, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}

	page := &effectivePage{
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
	}
	page.document.Trust = initialTrust(page.document)
	return page, nil
}

func (p *effectivePage) authoredRecord() map[string]any {
	record := make(map[string]any, len(p.fields)+1)
	for key, value := range p.fields {
		record[key] = value
	}
	record["uri"] = p.document.URI
	record["trust"] = p.document.Trust
	return record
}

func documentRevision(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}
