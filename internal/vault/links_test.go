package vault

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestInternalLinksUsesMarkdownAST(t *testing.T) {
	markdown := `# Links

[Inline](notes/a.md?view=full#section)
[Spaced](<notes/my file.md>)
[Reference][target]
![Image](images/example.png)
[Anchor](#section)
[External](https://example.com/page)
[Protocol relative](//cdn.example.com/file)

[target]: references/item.md

` + "```markdown\n[Ignored](missing.md)\n```\n" + `<a href="raw.md">Raw</a>
`

	links, err := internalLinks(markdown)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"notes/a.md?view=full#section",
		"notes/my file.md",
		"references/item.md",
		"images/example.png",
	}
	if len(links) != len(want) {
		t.Fatalf("links = %+v, want %d", links, len(want))
	}
	for i, raw := range want {
		if links[i].Raw != raw {
			t.Fatalf("links[%d].Raw = %q, want %q", i, links[i].Raw, raw)
		}
	}
	if links[1].Path != filepath.Join("notes", "my file.md") {
		t.Fatalf("spaced path = %q", links[1].Path)
	}
}

func TestInternalLinksRejectsInvalidEscapes(t *testing.T) {
	_, err := internalLinks("[Invalid](bad%ZZ.md)\n")
	if err == nil || !strings.Contains(err.Error(), "invalid destination") {
		t.Fatalf("error = %v", err)
	}
}

func TestLinkTargetRejectsVaultEscape(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "notes", "note.md")
	link := Link{Raw: "../../outside.md", Path: filepath.Join("..", "..", "outside.md")}

	_, err := linkTarget(root, file, link)
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateSupportsEncodedPathsAndFragments(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", "# Index\n\n[Log](log.md)\n")
	write(t, root, "log.md", "# Log\n\n## 2026-07-09\n")
	write(t, root, "my note.md", "---\ntype: Note\n---\n\n# Note\n")
	write(t, root, "source.md", "---\ntype: Note\n---\n\n[Note](my%20note.md?view=full#heading)\n")

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %v", result.Errors)
	}
}

func TestValidateRejectsLinkOutsideVault(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "vault")
	write(t, root, "index.md", "# Index\n\n[Log](log.md)\n")
	write(t, root, "log.md", "# Log\n\n## 2026-07-09\n")
	write(t, parent, "outside.md", "# Outside\n")
	write(t, root, "source.md", "---\ntype: Note\n---\n\n[Outside](../outside.md)\n")

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "escapes") {
		t.Fatalf("errors = %v", result.Errors)
	}
}
