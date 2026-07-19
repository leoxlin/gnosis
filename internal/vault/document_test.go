package vault

import (
	"path/filepath"
	"testing"
)

func TestReadPageAcceptsOnlyCanonicalGnosisURIs(t *testing.T) {
	root := apiTestVault(t)
	canonical := "gnosis://agent-test/processes/query-vault.md"

	page, err := ReadPage(root, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if page.Document.URI != canonical {
		t.Fatalf("URI = %q, want %q", page.Document.URI, canonical)
	}

	if _, err := ReadPage(root, "  "+canonical+"  "); err == nil {
		t.Fatal("ReadPage accepted a noncanonical whitespace-padded URI")
	}
}

func TestAnyVaultSelectorResolvesEffectivePageByPrecedence(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "imported"
vault_root = "."
`)
	write(t, workspace, "local/shared.md", "---\ntype: Note\ntitle: Local\n---\n")
	write(t, imported, "shared.md", "---\ntype: Note\ntitle: Imported\n---\n")
	write(t, imported, "imported-only.md", "---\ntype: Note\ntitle: Imported only\n---\n")

	for _, test := range []struct {
		selector string
		wantURI  string
	}{
		{"gnosis://_/shared.md", "gnosis://workspace/shared.md"},
		{"gnosis://_/imported-only.md", "gnosis://imported/imported-only.md"},
	} {
		page, err := ReadPage(workspace, test.selector)
		if err != nil {
			t.Fatal(err)
		}
		if page.Document.URI != test.wantURI {
			t.Fatalf("ReadPage(%q) URI = %q, want %q", test.selector, page.Document.URI, test.wantURI)
		}
	}
}
