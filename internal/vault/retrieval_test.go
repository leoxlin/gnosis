package vault

import (
	"strings"
	"testing"
)

func document(path string) Document {
	return Document{Path: path, URI: "gnosis://test/" + path}
}

func TestSearchPrefersExactMetadataOverBodyMatches(t *testing.T) {
	engine := New([]Document{
		{
			Path:        "concepts/rate-limiting.md",
			URI:         "gnosis://test/concepts/rate-limiting.md",
			Title:       "Rate Limiting",
			Description: "Controls request volume.",
			Body:        "A concise concept page.",
		},
		{
			Path:  "references/networking.md",
			URI:   "gnosis://test/references/networking.md",
			Title: "Networking Notes",
			Body:  strings.Repeat("rate limiting ", 30),
		},
	})

	hits := engine.Search("What is rate limiting?", 2)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Document.URI != "gnosis://test/concepts/rate-limiting.md" || !hits[0].Exact {
		t.Fatalf("top hit = %+v", hits[0])
	}
}

func TestSearchUsesDescriptionTagsBodyAndTechnicalTokens(t *testing.T) {
	engine := New([]Document{
		{
			Path:        "references/navigation.md",
			URI:         "gnosis://test/references/navigation.md",
			Title:       "Navigation Settings",
			Description: "Controls optional vault indexes.",
			Tags:        []string{"vault_index", "configuration"},
		},
		{
			Path:  "references/query.md",
			URI:   "gnosis://test/references/query.md",
			Title: "Query Reference",
			Body:  "The body documents semantic-recall behavior.",
		},
	})

	for _, test := range []struct {
		query string
		want  string
	}{
		{query: "vault index", want: "gnosis://test/references/navigation.md"},
		{query: "optional indexes", want: "gnosis://test/references/navigation.md"},
		{query: "semantic recall", want: "gnosis://test/references/query.md"},
	} {
		t.Run(test.query, func(t *testing.T) {
			hits := engine.Search(test.query, 1)
			if len(hits) != 1 || hits[0].Document.URI != test.want {
				t.Fatalf("hits = %+v, want %s", hits, test.want)
			}
		})
	}
}

func TestSearchPreservesDuplicateBasenamesByFullID(t *testing.T) {
	engine := New([]Document{
		{Path: "docs/concepts/indexing.md", URI: "gnosis://test/docs/concepts/indexing.md", Title: "Concept Indexing"},
		{Path: "notes/indexing.md", URI: "gnosis://test/notes/indexing.md", Title: "Personal Indexing"},
	})

	hits := engine.Search("indexing", 2)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Document.URI == hits[1].Document.URI {
		t.Fatalf("duplicate URIs in hits: %+v", hits)
	}
}

func TestQueryExactDescriptionIsIndexOnly(t *testing.T) {
	engine := New([]Document{
		{
			Path:        "concepts/transformer.md",
			URI:         "gnosis://test/concepts/transformer.md",
			Title:       "Transformer Architecture",
			Description: "A self-attention sequence model.",
		},
		{
			Path:        "references/transformer-history.md",
			URI:         "gnosis://test/references/transformer-history.md",
			Title:       "Transformer History",
			Description: "Related background.",
		},
	})

	result := engine.Query("What is Transformer Architecture?", QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3})
	if !result.IndexOnly {
		t.Fatalf("result = %+v", result)
	}
	if len(result.ShouldRead) != 0 {
		t.Fatalf("should_read = %v", result.ShouldRead)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("index-only candidates = %+v", result.Candidates)
	}
}

func TestQueryBoundsContextAndTruncatesDescription(t *testing.T) {
	documents := []Document{}
	for _, path := range []string{"a", "b", "c", "d"} {
		documents = append(documents, Document{
			Path:        path + ".md",
			URI:         "gnosis://test/" + path + ".md",
			Title:       "Supporting " + path,
			Description: strings.Repeat("knowledge ", 40),
			Body:        "bounded context",
		})
	}
	engine := New(documents)

	result := engine.Query("bounded context", QueryOptions{Top: 3, MaxRead: 2, MaxDepth: 3})
	if len(result.Candidates) != 3 {
		t.Fatalf("candidates = %d", len(result.Candidates))
	}
	if len(result.ShouldRead) != 2 {
		t.Fatalf("should_read = %v", result.ShouldRead)
	}
	if got := len([]rune(result.Candidates[0].Description)); got > maxDescriptionRune {
		t.Fatalf("description length = %d", got)
	}
	if strings.Contains(result.Candidates[0].Description, "\n") {
		t.Fatalf("description contains newline: %q", result.Candidates[0].Description)
	}
}

func TestQueryMaxReadZeroReturnsNoPageRecommendations(t *testing.T) {
	page := document("page.md")
	page.Title, page.Body = "Page", "search term"
	engine := New([]Document{page})
	result := engine.Query("search term", QueryOptions{Top: 3, MaxRead: 0, MaxDepth: 3})
	if len(result.ShouldRead) != 0 {
		t.Fatalf("should_read = %v", result.ShouldRead)
	}
}

func TestQueryNoMatchesIsCompleteWithoutReads(t *testing.T) {
	page := document("page.md")
	page.Title = "Page"
	engine := New([]Document{page})
	result := engine.Query("zzznomatch", QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3})
	if !result.IndexOnly || len(result.Candidates) != 0 || len(result.ShouldRead) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestQueryClassifiesListAndGap(t *testing.T) {
	engine := New(nil)
	for _, test := range []struct {
		question string
		want     AnswerType
	}{
		{question: "List all pages about indexing", want: AnswerList},
		{question: "What gaps remain in indexing?", want: AnswerGap},
		{question: "What is indexing?", want: AnswerDirect},
	} {
		result := engine.Query(test.question, QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3})
		if result.AnswerType != test.want {
			t.Fatalf("%q answer type = %q, want %q", test.question, result.AnswerType, test.want)
		}
	}
}

func TestQueryFindsBoundedShortestPath(t *testing.T) {
	engine := New([]Document{
		{Path: "a.md", URI: "gnosis://test/a.md", Title: "Alpha", Description: "A.", Links: []string{"gnosis://test/b.md", "gnosis://test/d.md"}},
		{Path: "b.md", URI: "gnosis://test/b.md", Title: "Beta", Description: "B.", Links: []string{"gnosis://test/c.md"}},
		{Path: "c.md", URI: "gnosis://test/c.md", Title: "Gamma", Description: "C."},
		{Path: "d.md", URI: "gnosis://test/d.md", Title: "Delta", Description: "D.", Links: []string{"gnosis://test/c.md"}},
	})

	result := engine.Query("How is Alpha connected to Gamma?", QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 2})
	if result.AnswerType != AnswerPath {
		t.Fatalf("answer type = %q", result.AnswerType)
	}
	want := []string{"gnosis://test/a.md", "gnosis://test/b.md", "gnosis://test/c.md"}
	if strings.Join(result.Path, ",") != strings.Join(want, ",") {
		t.Fatalf("path = %v, want %v", result.Path, want)
	}
	if len(result.ShouldRead) != 3 {
		t.Fatalf("should_read = %v", result.ShouldRead)
	}

	shallow := engine.Query("How is Alpha connected to Gamma?", QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 1})
	if len(shallow.Path) != 0 {
		t.Fatalf("shallow path = %v", shallow.Path)
	}
}

func TestFindPathTraversesReverseLinks(t *testing.T) {
	engine := New([]Document{
		{Path: "source.md", URI: "gnosis://test/source.md", Links: []string{"gnosis://test/target.md"}},
		{Path: "target.md", URI: "gnosis://test/target.md"},
	})
	path := engine.FindPath("gnosis://test/target.md", "gnosis://test/source.md", 1)
	if strings.Join(path, ",") != "gnosis://test/target.md,gnosis://test/source.md" {
		t.Fatalf("path = %v", path)
	}
}
