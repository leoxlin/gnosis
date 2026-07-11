package vault

import (
	"strings"
	"testing"
)

func TestSearchPrefersExactMetadataOverBodyMatches(t *testing.T) {
	engine := New([]Document{
		{
			ID:          "concepts/rate-limiting.md",
			Title:       "Rate Limiting",
			Description: "Controls request volume.",
			Body:        "A concise concept page.",
		},
		{
			ID:    "references/networking.md",
			Title: "Networking Notes",
			Body:  strings.Repeat("rate limiting ", 30),
		},
	})

	hits := engine.Search("What is rate limiting?", 2)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Document.ID != "concepts/rate-limiting.md" || !hits[0].Exact {
		t.Fatalf("top hit = %+v", hits[0])
	}
}

func TestSearchUsesDescriptionTagsBodyAndTechnicalTokens(t *testing.T) {
	engine := New([]Document{
		{
			ID:          "decisions/navigation.md",
			Title:       "Navigation Settings",
			Description: "Controls optional vault indexes.",
			Tags:        []string{"vault_index", "configuration"},
		},
		{
			ID:    "references/query.md",
			Title: "Query Reference",
			Body:  "The body documents semantic-recall behavior.",
		},
	})

	for _, test := range []struct {
		query string
		want  string
	}{
		{query: "vault index", want: "decisions/navigation.md"},
		{query: "optional indexes", want: "decisions/navigation.md"},
		{query: "semantic recall", want: "references/query.md"},
	} {
		t.Run(test.query, func(t *testing.T) {
			hits := engine.Search(test.query, 1)
			if len(hits) != 1 || hits[0].Document.ID != test.want {
				t.Fatalf("hits = %+v, want %s", hits, test.want)
			}
		})
	}
}

func TestSearchPreservesDuplicateBasenamesByFullID(t *testing.T) {
	engine := New([]Document{
		{ID: "docs/concepts/indexing.md", Title: "Concept Indexing"},
		{ID: "notes/indexing.md", Title: "Personal Indexing"},
	})

	hits := engine.Search("indexing", 2)
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	if hits[0].Document.ID == hits[1].Document.ID {
		t.Fatalf("duplicate IDs in hits: %+v", hits)
	}
}

func TestQueryCanUseReplaceableRetriever(t *testing.T) {
	document := Document{ID: "external/page.md", Title: "External Page"}
	retriever := &staticRetriever{hits: []Hit{{Document: document, Score: 42}}}
	engine := NewWithRetriever([]Document{document}, retriever)

	result := engine.Query("anything", QueryOptions{Top: 1, MaxRead: 1, MaxDepth: 1})
	if len(result.Candidates) != 1 || result.Candidates[0].Page != document.ID || result.Candidates[0].Score != 42 {
		t.Fatalf("result = %+v", result)
	}
	if retriever.query != "anything" || retriever.limit != 1 {
		t.Fatalf("retriever call = %q limit %d", retriever.query, retriever.limit)
	}
}

func TestQueryExactDescriptionIsIndexOnly(t *testing.T) {
	engine := New([]Document{
		{
			ID:          "concepts/transformer.md",
			Title:       "Transformer Architecture",
			Description: "A self-attention sequence model.",
		},
		{
			ID:          "references/transformer-history.md",
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
	for _, id := range []string{"a", "b", "c", "d"} {
		documents = append(documents, Document{
			ID:          id + ".md",
			Title:       "Supporting " + id,
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
	engine := New([]Document{{ID: "page.md", Title: "Page", Body: "search term"}})
	result := engine.Query("search term", QueryOptions{Top: 3, MaxRead: 0, MaxDepth: 3})
	if len(result.ShouldRead) != 0 {
		t.Fatalf("should_read = %v", result.ShouldRead)
	}
}

func TestQueryNoMatchesIsCompleteWithoutReads(t *testing.T) {
	engine := New([]Document{{ID: "page.md", Title: "Page"}})
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
		{ID: "a.md", Title: "Alpha", Description: "A.", Links: []string{"b.md", "d.md"}},
		{ID: "b.md", Title: "Beta", Description: "B.", Links: []string{"c.md"}},
		{ID: "c.md", Title: "Gamma", Description: "C."},
		{ID: "d.md", Title: "Delta", Description: "D.", Links: []string{"c.md"}},
	})

	result := engine.Query("How is Alpha connected to Gamma?", QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 2})
	if result.AnswerType != AnswerPath {
		t.Fatalf("answer type = %q", result.AnswerType)
	}
	want := []string{"a.md", "b.md", "c.md"}
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
		{ID: "source.md", Links: []string{"target.md"}},
		{ID: "target.md"},
	})
	path := engine.FindPath("target.md", "source.md", 1)
	if strings.Join(path, ",") != "target.md,source.md" {
		t.Fatalf("path = %v", path)
	}
}

type staticRetriever struct {
	hits  []Hit
	query string
	limit int
}

func (r *staticRetriever) Search(query string, limit int) []Hit {
	r.query = query
	r.limit = limit
	if len(r.hits) > limit {
		return append([]Hit(nil), r.hits[:limit]...)
	}
	return append([]Hit(nil), r.hits...)
}
