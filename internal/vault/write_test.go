package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDocumentWritesToURITarget(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"Workspace\"\nvault_root = \"local\"\n")
	write(t, root, "local/types/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	content := []byte("---\ntype: Note\ntitle: A New Note\n---\n\n# A New Note\n")

	target := "gnosis://Workspace/notes/custom-name.md"
	written, err := WriteDocument(root, target, content, false)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "local", "notes", "custom-name.md")
	if written != want {
		t.Fatalf("path = %q, want %q", written, want)
	}
	got, err := os.ReadFile(want)
	if err != nil || string(got) != string(content) {
		t.Fatalf("content = %q err = %v", got, err)
	}
}

func TestWriteDocumentAcceptsAnyVaultTargetByPrecedence(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, "[vault]\nvault_name = \"Workspace\"\nvault_root = \"local\"\n\n[[vaults]]\nvault_name = \"Imported\"\nvault_root = \"imported\"\n")
	writeConfig(t, imported, "[vault]\nvault_name = \"Imported\"\nvault_root = \".\"\n")
	write(t, workspace, "local/types/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	write(t, imported, "notes/existing.md", "---\ntype: Note\ntitle: Existing\n---\n")
	content := []byte("---\ntype: Note\ntitle: Any vault\n---\n")

	written, err := WriteDocument(workspace, "gnosis://_/notes/new.md", content, false)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(workspace, "local", "notes", "new.md")
	if written != want {
		t.Fatalf("path = %q, want %q", written, want)
	}

	if _, err := WriteDocument(workspace, "gnosis://_/notes/existing.md", content, false); err == nil || !strings.Contains(err.Error(), "--update") {
		t.Fatalf("collision error = %v", err)
	}
	if _, err := WriteDocument(workspace, "gnosis://_/notes/existing.md", content, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "local", "notes", "existing.md")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteDocumentValidatesTargetURIAndConceptPath(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"Local\"\nvault_root = \".\"\n")
	write(t, root, "types/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	content := []byte("---\ntype: Note\ntitle: A Note\n---\n")

	for _, test := range []struct {
		uri  string
		want string
	}{
		{"gnosis://Other/notes/a-note.md", "current local vault"},
		{"gnosis://Local/notes/../a-note.md", "canonical"},
		{"gnosis://Local/notes%5Ca-note.md", "canonical"},
		{"gnosis://Local/other/a-note.md", "outside Concept Type"},
		{"gnosis://Local/notes/a-note.md?view=full", "canonical"},
		{"gnosis://Local/notes/a-note", "lowercase .md"},
		{"gnosis://Local/notes/a-note.txt", "lowercase .md"},
		{"gnosis://Local/notes/a-note.MD", "lowercase .md"},
		{"gnosis://Local/notes/index.md", "reserved name"},
		{"gnosis://Local/notes/log.md", "reserved name"},
	} {
		if _, err := WriteDocument(root, test.uri, content, false); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("WriteDocument(%q) error = %v, want %q", test.uri, err, test.want)
		}
	}
}

func TestWriteDocumentRequiresUpdateToShadowExternalTarget(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, "[vault]\nvault_name = \"Workspace\"\nvault_root = \"local\"\n\n[[vaults]]\nvault_name = \"Imported\"\nvault_root = \"imported\"\n")
	writeConfig(t, imported, "[vault]\nvault_name = \"Imported\"\nvault_root = \".\"\n")
	write(t, workspace, "local/types/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
	write(t, imported, "notes/imported-note.md", "---\ntype: Note\ntitle: Imported Note\n---\n")
	content := []byte("---\ntype: Note\ntitle: Imported Note\n---\n\n# Local\n")
	target := "gnosis://Workspace/notes/imported-note.md"

	if _, err := WriteDocument(workspace, target, content, false); err == nil || !strings.Contains(err.Error(), "--update") {
		t.Fatalf("collision error = %v", err)
	}
	if _, err := WriteDocument(workspace, target, content, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "local", "notes", "imported-note.md")); err != nil {
		t.Fatal(err)
	}
}

func TestWriteDocumentRequiresCurrentLocalVault(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, "[[vaults]]\nvault_name = \"Imported\"\nvault_root = \"imported\"\n")
	writeConfig(t, imported, "[vault]\nvault_name = \"Imported\"\nvault_root = \".\"\n")
	content := []byte("---\ntype: Note\ntitle: A Note\n---\n")
	if _, err := WriteDocument(workspace, "gnosis://Imported/notes/a-note.md", content, false); err == nil || !strings.Contains(err.Error(), "local vault") {
		t.Fatalf("error = %v", err)
	}
}

func TestWriteGeneratedFileSkipsUnchangedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.md")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := WriteGeneratedFile(path, []byte("same"), true)
	if err != nil || changed {
		t.Fatalf("changed = %t, err = %v", changed, err)
	}
}

func TestAtomicWriteFileCleansUpAfterRenameFailure(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "target")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(destination, []byte("content"), 0o644); err == nil {
		t.Fatal("expected replacing a directory to fail")
	}
	temps, err := filepath.Glob(filepath.Join(root, ".target.tmp-*"))
	if err != nil || len(temps) != 0 {
		t.Fatalf("temporary files = %v err = %v", temps, err)
	}
}
