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
	Force        bool
	Name         string
	Concepts     bool
	DisableIndex bool
	DisableLog   bool
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
	created := []string{}
	target, err := resolveVaultTarget(root)
	if err != nil {
		return created, err
	}
	root = target.root
	if err := os.MkdirAll(root, 0o755); err != nil {
		return created, err
	}
	changed, err := writeVaultConfig(root, options.Name, options.DisableIndex, options.DisableLog, options.Force)
	if err != nil {
		return created, err
	}
	if changed {
		created = append(created, filepath.Join(root, "gnosis.toml"))
	}
	config, err := loadConfig(root)
	if err != nil {
		return created, err
	}
	options.DisableIndex = !config.IndexEnabled()
	options.DisableLog = !config.LogEnabled()
	dirs := []string{
		"concepts",
		"references",
	}

	for _, dir := range dirs {
		path := filepath.Join(root, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return created, err
		}
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
		{"AGENTS.md", "AGENTS.md.tmpl"},
	}
	if !options.DisableLog {
		files = append(files, scaffoldFile{"log.md", "log.md.tmpl"})
	}
	for _, file := range files {
		path := filepath.Join(root, file.rel)
		var buf strings.Builder
		if err := scaffoldTemplates.ExecuteTemplate(&buf, file.tmplName, data); err != nil {
			return created, err
		}
		changed, err := WriteGeneratedFile(path, []byte(buf.String()), options.Force)
		if err != nil {
			return created, err
		}
		if changed {
			created = append(created, path)
		}
	}

	if options.Concepts {
		documents, err := BundledConcepts()
		if err != nil {
			return created, err
		}
		for _, document := range documents {
			path := filepath.Join(root, document.Path)
			changed, err := WriteGeneratedFile(path, document.Data, options.Force)
			if err != nil {
				return created, err
			}
			if changed {
				created = append(created, path)
			}
		}
	}

	if !options.DisableIndex {
		paths, err := GenerateIndexes(root, IndexOptions{Overwrite: options.Force})
		if err != nil {
			return created, err
		}
		created = append(created, paths...)
	}

	if len(created) > 0 && target.backend != nil {
		if err := target.backend.publish("gnosis: create vault"); err != nil {
			return created, err
		}
	}
	return created, nil
}
