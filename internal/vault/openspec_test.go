package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSpecArtifactMutationIsRejected(t *testing.T) {
	repository := openSpecTestRepository(t)
	nested := filepath.Join(repository, "docs", "openspec")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("## Why\n\nThis is native OpenSpec Markdown without YAML frontmatter.\n")
	target := "gnosis://local/openspec/changes/add-atlas/proposal.md"

	_, err := WriteDocument(nested, target, content, false)
	if err == nil || !strings.Contains(err.Error(), "OpenSpec artifacts are read-only through gnosis") {
		t.Fatalf("WriteDocument error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repository, "docs", "openspec", "changes", "add-atlas", "proposal.md")); !os.IsNotExist(statErr) {
		t.Fatalf("proposal stat error = %v, want not written", statErr)
	}
}

func TestFrontmatterFreeMarkdownIsIgnoredByValidation(t *testing.T) {
	repository := openSpecTestRepository(t)
	writeOpenSpecTestFile(t, repository, "docs/notes/plain.md", "# Plain\n\nNo frontmatter.\n")

	result, err := Validate(filepath.Join(repository, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Warnings) != 0 || result.FilesChecked != 0 {
		t.Fatalf("validation = %+v", result)
	}
}

func openSpecTestRepository(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repository := t.TempDir()
	writeOpenSpecTestFile(t, repository, ".git/HEAD", "ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Join(repository, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repository
}

func writeOpenSpecTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
