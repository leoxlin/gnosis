package vault

import (
	"errors"
	"fmt"
	"strings"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v4"
)

// Frontmatter stores parsed top-level YAML fields.
type Frontmatter map[string]*yaml.Node

func parseFrontmatter(markdown string) (Frontmatter, string, error) {
	var document yaml.Node
	body, err := frontmatter.MustParse(strings.NewReader(markdown), &document,
		frontmatter.NewFormat("---", "---", yaml.Unmarshal))
	if errors.Is(err, frontmatter.ErrNotFound) {
		return nil, "", fmt.Errorf("missing YAML frontmatter")
	}
	if err != nil {
		return nil, "", fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, "", fmt.Errorf("YAML frontmatter must be a mapping")
	}

	mapping := document.Content[0]
	fields := make(Frontmatter, len(mapping.Content)/2)
	for i := 0; i < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		value := mapping.Content[i+1]
		if key.Kind != yaml.ScalarNode || strings.TrimSpace(key.Value) == "" {
			return nil, "", fmt.Errorf("YAML frontmatter keys must be non-empty scalars")
		}
		if _, exists := fields[key.Value]; exists {
			return nil, "", fmt.Errorf("duplicate YAML frontmatter key %q", key.Value)
		}
		fields[key.Value] = value
	}

	return fields, string(body), nil
}

func (f Frontmatter) scalar(key string) (string, bool) {
	node, exists := f[key]
	if !exists {
		return "", false
	}
	node = resolveAlias(node)
	if node == nil || node.Kind != yaml.ScalarNode {
		return "", false
	}
	return node.Value, true
}

// scalars returns a frontmatter scalar or sequence of scalars. The boolean is
// false only when the field exists with an incompatible shape.
func (f Frontmatter) scalars(key string) ([]string, bool) {
	node, exists := f[key]
	if !exists {
		return nil, true
	}
	node = resolveAlias(node)
	if node == nil {
		return nil, true
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" {
			return nil, true
		}
		return []string{strings.TrimSpace(node.Value)}, true
	case yaml.SequenceNode:
		values := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			item = resolveAlias(item)
			if item == nil || item.Kind != yaml.ScalarNode {
				return nil, false
			}
			if item.Tag == "!!null" || strings.TrimSpace(item.Value) == "" {
				continue
			}
			values = append(values, strings.TrimSpace(item.Value))
		}
		return values, true
	default:
		return nil, false
	}
}

func (f Frontmatter) nonEmpty(key string) bool {
	node, exists := f[key]
	if !exists {
		return false
	}
	node = resolveAlias(node)
	if node == nil {
		return false
	}
	if node.Kind == yaml.ScalarNode {
		return strings.TrimSpace(node.Value) != "" && node.Tag != "!!null"
	}
	return len(node.Content) > 0
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}
