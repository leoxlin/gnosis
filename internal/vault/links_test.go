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
<gnosis://Test/notes/autolink.md>

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
		"gnosis://Test/notes/autolink.md",
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

func TestInternalLinksHandlesBalancedDestinationsAndMarkdownCode(t *testing.T) {
	markdown := "[Balanced](notes/a_(draft).md)\n\n" +
		"``[Code span](notes/code.md)``\n\n" +
		"````markdown\n[Fenced](notes/fenced.md)\n~~~\n[Still fenced](notes/still-fenced.md)\n````\n\n" +
		"[After](notes/after.md)\n"

	links, err := internalLinks(markdown)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"notes/a_(draft).md", "notes/after.md"}
	if len(links) != len(want) {
		t.Fatalf("links = %+v, want %v", links, want)
	}
	for index, raw := range want {
		if links[index].Raw != raw {
			t.Fatalf("links[%d].Raw = %q, want %q", index, links[index].Raw, raw)
		}
	}
}

func TestInternalLinksUsesMarkdownDestinationSemantics(t *testing.T) {
	markdown := `[Escaped](notes/a\(draft\).md)
[Entity](notes/a&amp;b.md)
`
	links, err := internalLinks(markdown)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join("notes", "a(draft).md"),
		filepath.Join("notes", "a&b.md"),
	}
	if len(links) != len(want) {
		t.Fatalf("links = %+v, want %v", links, want)
	}
	for index, path := range want {
		if links[index].Path != path {
			t.Fatalf("links[%d].Path = %q, want %q", index, links[index].Path, path)
		}
	}
	if links[0].Raw != `notes/a\(draft\).md` {
		t.Fatalf("raw destination = %q", links[0].Raw)
	}
}

func TestRewriteMarkdownDestinationsPreservesAuthoredSyntax(t *testing.T) {
	markdown := `[Balanced](notes/a_(draft).md "Draft title")
[Reference][target]

[target]: <notes/reference file.md?view=full#section> 'Reference title'
`

	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		switch raw {
		case "notes/a_(draft).md":
			return "gnosis://Test/notes/a_(draft).md"
		case "notes/reference file.md?view=full#section":
			return "gnosis://Test/notes/reference%20file.md?view=full#section"
		default:
			return raw
		}
	})
	want := `[Balanced](gnosis://Test/notes/a_(draft).md "Draft title")
[Reference][target]

[target]: <gnosis://Test/notes/reference%20file.md?view=full#section> 'Reference title'
`
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestRewriteMarkdownDestinationsLeavesUnusedReferencesUntouched(t *testing.T) {
	markdown := "[Used][used]\n\n[used]: notes/used.md\n[unused]: notes/unused.md\n"
	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		return "gnosis://Test/" + raw
	})
	want := "[Used][used]\n\n[used]: gnosis://Test/notes/used.md\n[unused]: notes/unused.md\n"
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestRewriteMarkdownDestinationsUsesFirstDuplicateReference(t *testing.T) {
	markdown := "[foo]: notes/first.md\n[FOO]: notes/ignored.md\n\n[Foo][foo]\n"
	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		return "gnosis://Test/" + raw
	})
	want := "[foo]: gnosis://Test/notes/first.md\n[FOO]: notes/ignored.md\n\n[Foo][foo]\n"
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestRewriteMarkdownDestinationsHonorsEmptyFirstReference(t *testing.T) {
	markdown := "[foo]: <>\n[FOO]: notes/ignored.md\n\n[Foo][foo]\n"
	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		return "gnosis://Test/" + raw
	})
	if got != markdown {
		t.Fatalf("rewrite = %q, want source unchanged", got)
	}
}

func TestRewriteMarkdownDestinationsUsesASTSpansInContainersAndRichLabels(t *testing.T) {
	markdown := `> [target]:
> notes/target.md
>
> [Reference][target]

[<span title="]">label</span>](notes/target.md)
`
	want := `> [target]:
> gnosis://Test/notes/target.md
>
> [Reference][target]

[<span title="]">label</span>](gnosis://Test/notes/target.md)
`
	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		if raw == "notes/target.md" {
			return "gnosis://Test/notes/target.md"
		}
		return raw
	})
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestRewriteMarkdownDestinationsHandlesLinkedImages(t *testing.T) {
	markdown := "[![Preview](images/preview.png)](notes/target.md)\n"
	got := rewriteMarkdownDestinations(markdown, func(raw string) string {
		switch raw {
		case "images/preview.png":
			return "gnosis://Test/images/preview.png"
		case "notes/target.md":
			return "gnosis://Test/notes/target.md"
		}
		return raw
	})
	want := "[![Preview](gnosis://Test/images/preview.png)](gnosis://Test/notes/target.md)\n"
	if got != want {
		t.Fatalf("rewrite = %q, want %q", got, want)
	}
}

func TestCanonicalGnosisURISemantics(t *testing.T) {
	canonical := "gnosis://Test/notes/a%20note.md"
	if got, ok := canonicalGnosisURI(canonical); !ok || got != canonical {
		t.Fatalf("canonical URI = %q, %t", got, ok)
	}
	if got, ok := canonicalGnosisLink(canonical + "?view=full#part"); !ok || got != canonical {
		t.Fatalf("canonical link = %q, %t", got, ok)
	}
	if got, ok := canonicalGnosisLink(canonical + "#section%2Fchild"); !ok || got != canonical {
		t.Fatalf("encoded-fragment canonical link = %q, %t", got, ok)
	}
	link, include, err := parseLinkDestination(canonical + "#part")
	if err != nil || !include || link.URI != canonical {
		t.Fatalf("parsed canonical link = %+v, include=%t, err=%v", link, include, err)
	}

	for _, raw := range []string{
		"gnosis://Test/notes/../a.md",
		"gnosis://Test//notes/a.md",
		"gnosis://user@Test/notes/a.md",
		"gnosis://Test/notes%2Fa.md",
		"GNOSIS://Test/notes/a.md",
		canonical + "#part",
	} {
		if got, ok := canonicalGnosisURI(raw); ok {
			t.Fatalf("canonicalGnosisURI(%q) = %q, want rejected", raw, got)
		}
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
