package forge

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gnosis/internal/vault"
)

// ConceptOptions controls how forge concept definitions are written.
type ConceptOptions struct {
	Force bool
}

type conceptData struct {
	Timestamp string
	Date      string
}

//go:embed templates/*.tmpl
var conceptTemplatesFS embed.FS

var conceptTemplates = template.Must(template.ParseFS(conceptTemplatesFS, "templates/*.tmpl"))

// Concepts writes the reusable repository concept definitions (purpose,
// decision, directive) used by the gnosis-forge workflow into root/concepts.
// Existing files are left alone unless opts.Force is set.
func Concepts(root string, opts ConceptOptions) ([]string, error) {
	root = filepath.Clean(root)
	if err := os.MkdirAll(filepath.Join(root, "concepts"), 0o755); err != nil {
		return nil, err
	}

	now := time.Now()
	data := conceptData{
		Timestamp: now.Format(time.RFC3339),
		Date:      now.Format("2006-01-02"),
	}

	files := []struct {
		rel      string
		tmplName string
	}{
		{"concepts/repository-purpose.md", "repository-purpose.md.tmpl"},
		{"concepts/repository-decision.md", "repository-decision.md.tmpl"},
		{"concepts/repository-directive.md", "repository-directive.md.tmpl"},
	}

	created := []string{}
	for _, file := range files {
		path := filepath.Join(root, file.rel)
		var buf strings.Builder
		if err := conceptTemplates.ExecuteTemplate(&buf, file.tmplName, data); err != nil {
			return created, err
		}
		changed, err := vault.WriteGeneratedFile(path, []byte(buf.String()), opts.Force)
		if err != nil {
			return created, err
		}
		if changed {
			created = append(created, path)
		}
	}
	return created, nil
}
