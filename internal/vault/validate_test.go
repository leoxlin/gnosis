package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsMinimalVault(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", `# Test

[Log](/log.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-07

* Entry.
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
	if result.FilesChecked != 2 {
		t.Fatalf("files checked = %d, want 2", result.FilesChecked)
	}
}

func TestValidateRejectsMissingType(t *testing.T) {
	root := t.TempDir()
	write(t, root, "note.md", `---
title: Test
---

# Test
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsBrokenInternalLink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "note.md", `---
type: Concept
---

# Test

[Missing](/missing.md)
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation error")
	}
}

func TestValidateRequiresRootIndexAndLog(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n")

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("errors = %v, want missing root directory index and log", result.Errors)
	}
}

func TestValidateAllowsDisabledIndexAndLog(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_index = false
vault_log = false
`)
	write(t, root, "note.md", `---
type: Concept
title: Concept
description: Minimal concept.
---

# Concept
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected validation warnings: %v", result.Warnings)
	}
}

func TestValidateHonorsIndexAndLogIndependently(t *testing.T) {
	tests := []struct {
		name   string
		config string
		files  map[string]string
	}{
		{
			name: "index disabled",
			config: `[vault]
vault_index = false
vault_log = true
`,
			files: map[string]string{
				"log.md": "# Log\n\n## 2026-07-10\n",
			},
		},
		{
			name: "log disabled",
			config: `[vault]
vault_index = true
vault_log = false
`,
			files: map[string]string{
				"index.md": "# Index\n\n[Purpose](purpose.md)\n",
				"purpose.md": `---
type: Purpose
title: Purpose
description: Test purpose.
---

# Purpose
`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, test.config)
			for rel, content := range test.files {
				write(t, root, rel, content)
			}

			result, err := Validate(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Errors) != 0 {
				t.Fatalf("unexpected validation errors: %v", result.Errors)
			}
		})
	}
}

func TestValidateRequiresIndexForEveryDirectory(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", `# Index

[Concept](concepts/concept.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-09

* Entry.
`)
	write(t, root, "concepts/concept.md", `---
type: Concept
---

# Concept
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want missing concepts index", result.Errors)
	}
}

func TestValidateWarnsForRecommendedFields(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", `# Index

[Concept](/concept.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-07

* Entry.
`)
	write(t, root, "concept.md", `---
type: Concept
---

# Concept
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected recommended-field warnings")
	}
}

func TestValidateWarnsOnWrongLinkFormat(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
link_format = "relative"
link_format_strict = false
`)
	write(t, root, "index.md", `# Index

[Concept](/concept.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-07

* Entry.
`)
	write(t, root, "concept.md", `---
type: Concept
---

# Concept
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected link-format warning")
	}
}

func TestValidateErrorsOnWrongLinkFormatWhenStrict(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
link_format = "relative"
link_format_strict = true
`)
	write(t, root, "index.md", `# Index

[Concept](/concept.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-07

* Entry.
`)
	write(t, root, "concept.md", `---
type: Concept
---

# Concept
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation error for absolute link")
	}
}

func TestValidateResolvesRelativeLinks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", `# Index

[Concept](concepts/concept.md)
`)
	write(t, root, "log.md", `# Log

## 2026-07-07

* Entry.
`)
	write(t, root, "concepts/concept.md", `---
type: Concept
---

# Concept
`)
	write(t, root, "concepts/index.md", `# Concepts

[Parent Index](../index.md)
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	configPath := filepath.Join(root, "gnosis.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) && rel != "gnosis.toml" {
		writeConfig(t, root, "[vault]\nvault_name = \"Test\"\nvault_root = \".\"")
	}
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, root, content string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(content, "[vault]\n") && !strings.Contains(content, "vault_name") {
		content = strings.Replace(content, "[vault]\n", "[vault]\nvault_name = \"Test\"\nvault_root = \".\"\n", 1)
	}
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
