package vault

import (
	"os"
	"path/filepath"
	"testing"
)

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
