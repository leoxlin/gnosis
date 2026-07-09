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
	Force           bool
	IncludeConcepts bool
}

// scaffoldData provides values for scaffold templates.
type scaffoldData struct {
	Timestamp string
	Date      string
}

//go:embed templates/*.tmpl
var scaffoldTemplatesFS embed.FS

var scaffoldTemplates = template.Must(template.ParseFS(scaffoldTemplatesFS, "templates/*.tmpl"))

// Scaffold creates the base OKF vault shape. Existing files are left alone
// unless Force is set.
func Scaffold(root string, options ScaffoldOptions) ([]string, error) {
	root = filepath.Clean(root)
	dirs := []string{
		"concepts",
		"references",
	}

	created := []string{}
	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return created, err
		}
		created = append(created, path)
	}

	now := time.Now()
	data := scaffoldData{
		Timestamp: now.Format(time.RFC3339),
		Date:      now.Format("2006-01-02"),
	}
	type scaffoldFile struct {
		rel      string
		tmplName string
	}
	files := []scaffoldFile{
		{"index.md", "index.md.tmpl"},
		{"log.md", "log.md.tmpl"},
		{"AGENTS.md", "AGENTS.md.tmpl"},
	}
	if options.IncludeConcepts {
		files = append(files,
			scaffoldFile{"concepts/index.md", "concepts-index.md.tmpl"},
			scaffoldFile{"concepts/repository-purpose.md", "repository-purpose.md.tmpl"},
			scaffoldFile{"concepts/repository-decision.md", "repository-decision.md.tmpl"},
			scaffoldFile{"concepts/repository-directive.md", "repository-directive.md.tmpl"},
			scaffoldFile{"concepts/repository-delta.md", "repository-delta.md.tmpl"},
		)
	}
	for _, file := range files {
		path := filepath.Join(root, file.rel)
		if !options.Force {
			if _, err := os.Stat(path); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return created, err
			}
		}

		var buf strings.Builder
		if err := scaffoldTemplates.ExecuteTemplate(&buf, file.tmplName, data); err != nil {
			return created, err
		}

		if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
			return created, err
		}
		created = append(created, path)
	}

	return created, nil
}
