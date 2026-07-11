package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDocumentWritesToCurrentLocalVaultUsingConceptTypePath(t *testing.T) {
	workspace := t.TempDir()
	writeConfig(t, workspace, `[vault]
vault_name = "Workspace"
vault_root = "local"
`)
	write(t, workspace, "local/types/note.md", `---
type: Concept Type
title: Note
path: notes
---
`)
	content := []byte(`---
type: Note
title: A New Note
---

# A New Note
`)

	path, err := WriteDocument(workspace, "Note", "A New Note", content, false)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(workspace, "local", "notes", "a-new-note.md")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("content = %q, want %q", got, content)
	}
	replacement := []byte(`---
type: Note
title: A New Note
---

# Replacement
`)
	if _, err := WriteDocument(workspace, "Note", "A New Note", replacement, false); err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(replacement) {
		t.Fatalf("replacement = %q, want %q", got, replacement)
	}
}

func TestWriteDocumentRejectsInvalidContentAndConceptTypePath(t *testing.T) {
	workspace := t.TempDir()
	writeConfig(t, workspace, `[vault]
vault_name = "Workspace"
vault_root = "."
`)
	write(t, workspace, "types/note.md", `---
type: Concept Type
title: Note
---
`)
	valid := []byte("---\ntype: Note\ntitle: A Note\n---\n")

	if _, err := WriteDocument(workspace, "Note", "A Note", valid, false); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("missing path error = %v", err)
	}
	write(t, workspace, "types/note.md", `---
type: Concept Type
title: Note
path: ../outside
---
`)
	if _, err := WriteDocument(workspace, "Note", "A Note", valid, false); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("unsafe path error = %v", err)
	}
	write(t, workspace, "types/note.md", `---
type: Concept Type
title: Note
path: notes
---
`)
	wrongType := []byte("---\ntype: Other\ntitle: A Note\n---\n")
	if _, err := WriteDocument(workspace, "Note", "A Note", wrongType, false); err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("type mismatch error = %v", err)
	}
	wrongTitle := []byte("---\ntype: Note\ntitle: Other\n---\n")
	if _, err := WriteDocument(workspace, "Note", "A Note", wrongTitle, false); err == nil || !strings.Contains(err.Error(), "title") {
		t.Fatalf("title mismatch error = %v", err)
	}
}

func TestWriteDocumentRequiresOverwriteForImportedOrBuiltInDocuments(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[vault]
vault_name = "Workspace"
vault_root = "local"

[[vaults]]
vault_name = "Imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "Imported"
vault_root = "."
`)
	write(t, workspace, "local/types/note.md", `---
type: Concept Type
title: Note
path: notes
---
`)
	write(t, imported, "notes/imported-note.md", "---\ntype: Note\ntitle: Imported Note\n---\n")
	importedContent := []byte("---\ntype: Note\ntitle: Imported Note\n---\n\n# Local\n")
	if _, err := WriteDocument(workspace, "Note", "Imported Note", importedContent, false); err == nil || !strings.Contains(err.Error(), "-overwrite") {
		t.Fatalf("imported collision error = %v", err)
	}
	if _, err := WriteDocument(workspace, "Note", "Imported Note", importedContent, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "local", "notes", "imported-note.md")); err != nil {
		t.Fatalf("local override = %v", err)
	}
	resolvedImported, err := Read(workspace, "Note", "Imported Note")
	if err != nil {
		t.Fatal(err)
	}
	if string(resolvedImported) != string(importedContent) {
		t.Fatalf("resolved imported document = %q, want %q", resolvedImported, importedContent)
	}

	write(t, workspace, "local/concepts/vault-process.md", `---
type: Concept Type
title: Vault Process
path: vault/processes
---
`)
	builtInContent := []byte("---\ntype: Vault Process\ntitle: query-vault\n---\n\n# Local query vault\n")
	if _, err := WriteDocument(workspace, "Vault Process", "query-vault", builtInContent, false); err == nil || !strings.Contains(err.Error(), "-overwrite") {
		t.Fatalf("built-in collision error = %v", err)
	}
	if _, err := WriteDocument(workspace, "Vault Process", "query-vault", builtInContent, true); err != nil {
		t.Fatal(err)
	}
	resolvedBuiltIn, err := Read(workspace, "Vault Process", "query-vault")
	if err != nil {
		t.Fatal(err)
	}
	if string(resolvedBuiltIn) != string(builtInContent) {
		t.Fatalf("resolved built-in document = %q, want %q", resolvedBuiltIn, builtInContent)
	}
}

func TestWriteDocumentRequiresCurrentDirectoryLocalVault(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "Imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "Imported"
vault_root = "."
`)
	write(t, imported, "types/note.md", "---\ntype: Concept Type\ntitle: Note\npath: notes\n---\n")
	content := []byte("---\ntype: Note\ntitle: A Note\n---\n")
	if _, err := WriteDocument(workspace, "Note", "A Note", content, false); err == nil || !strings.Contains(err.Error(), "local vault") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(imported, "notes", "a-note.md")); !os.IsNotExist(err) {
		t.Fatalf("write to imported vault = %v", err)
	}
}

func TestWriteGeneratedFileSkipsUnchangedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.md")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := WriteGeneratedFile(path, []byte("same"), true)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("byte-identical content should not be rewritten")
	}
}

func TestAtomicWriteFileReplacesContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.md")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := atomicWriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q", content)
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
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("temporary files remain: %v", temps)
	}
	if info, err := os.Stat(destination); err != nil || !info.IsDir() {
		t.Fatalf("destination changed after failed write: info=%v err=%v", info, err)
	}
}
