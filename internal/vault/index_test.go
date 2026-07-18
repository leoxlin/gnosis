package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIndexesSkipsRootDocumentation(t *testing.T) {
	root := t.TempDir()
	write(t, root, "concepts/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	write(t, root, "documentation/guide.md", "# Guide\n\nSee [missing](missing.md).\n")
	write(t, root, "notes/documentation/thing.md", "---\ntype: Note\ntitle: Thing\n---\n")
	written, err := GenerateIndexes(root, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "documentation", "index.md")); !os.IsNotExist(err) {
		t.Fatalf("root documentation was indexed: %v", written)
	}
	rootIndex, err := os.ReadFile(filepath.Join(root, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rootIndex), "documentation/index.md") {
		t.Fatalf("root index links documentation: %q", rootIndex)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "documentation", "index.md")); err != nil {
		t.Fatalf("nested documentation dir was skipped: %v", err)
	}
}
