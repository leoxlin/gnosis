package vault

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// ScaffoldOptions controls how scaffold writes files.
type ScaffoldOptions struct {
	Force bool
}

// scaffoldData provides values for scaffold templates.
type scaffoldData struct {
	Timestamp string
}

//go:embed templates/*.tmpl
var scaffoldTemplatesFS embed.FS

var scaffoldTemplates = template.Must(template.ParseFS(scaffoldTemplatesFS, "templates/*.tmpl"))

// Scaffold creates the base Gnosis vault shape. Existing files are left alone
// unless Force is set.
func Scaffold(root string, options ScaffoldOptions) ([]string, error) {
	root = filepath.Clean(root)
	dirs := []string{
		"archive",
		"guides",
		"okr",
		"ontogpt",
		"people",
		"projects",
		"references",
		"resources",
		"routines",
		"schemas",
		"templates",
		"tooling",
		"workflows",
		"docs/gnosis/agent-context",
		"docs/gnosis/coding-agents",
		"docs/gnosis/plans",
		"docs/gnosis/qa",
	}

	created := []string{}
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return created, err
		}
		created = append(created, path)
	}

	data := scaffoldData{Timestamp: time.Now().Format(time.RFC3339)}
	files := map[string]string{
		"README.md": "README.md.tmpl",
		"index.md":  "index.md.tmpl",
		"log.md":    "log.md.tmpl",
		"AGENTS.md": "AGENTS.md.tmpl",
	}
	for rel, tmplName := range files {
		path := filepath.Join(root, rel)
		if !options.Force {
			if _, err := os.Stat(path); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return created, err
			}
		}

		var buf strings.Builder
		if err := scaffoldTemplates.ExecuteTemplate(&buf, tmplName, data); err != nil {
			return created, err
		}

		if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
			return created, err
		}
		created = append(created, path)
	}

	return created, nil
}
