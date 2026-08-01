package codeintel

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/codeintel/analyzer"
)

type testAnalyzer struct{}

type failingAnalyzer struct{ result analyzer.AnalysisResult }

func (testAnalyzer) Analyze(_ context.Context, request analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	documents := make([]analyzer.DocumentAnalysis, 0, len(request.Documents))
	for _, change := range request.Documents {
		coverage := make([]analyzer.Coverage, 0, len(request.Capabilities))
		for _, capability := range request.Capabilities {
			coverage = append(coverage, analyzer.Coverage{Capability: capability, Level: analyzer.Complete})
		}
		documents = append(documents, analyzer.DocumentAnalysis{
			Path: change.Path, Language: change.Language, ContentDigest: change.ContentDigest, Coverage: coverage,
			Symbols:   []analyzer.Symbol{{Kind: "function", Name: "Main", Span: analyzer.Span{EndByte: 4}}},
			Relations: []analyzer.Relation{{Kind: "call", Source: "Main", Target: "Main", Evidence: analyzer.Syntactic, Resolution: analyzer.Unresolved, Span: analyzer.Span{EndByte: 4}}},
		})
	}
	return analyzer.AnalysisResult{Snapshot: request.Snapshot, Complete: true, Documents: documents, Provenance: analyzer.AnalyzerProvenance{Implementation: "fake", ImplementationVersion: "1", ParserRelease: "1", ParserDigest: "digest", ABI: "14", QueryVersion: "1", NormalizerVersion: "1"}}, nil
}

func (testAnalyzer) Close() error { return nil }

func (provider failingAnalyzer) Analyze(_ context.Context, _ analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	if provider.result.Snapshot != "" {
		return provider.result, nil
	}
	return analyzer.AnalysisResult{}, context.Canceled
}

func (failingAnalyzer) Close() error { return nil }

func TestBuildPublishesOneDeterministicGeneration(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	repository := t.TempDir()
	runGit(t, repository, "init")
	runGit(t, repository, "config", "user.email", "test@example.test")
	runGit(t, repository, "config", "user.name", "Test")
	writeTestFile(t, repository, "main.go", "package main\nfunc Main() {}\n")
	writeTestFile(t, repository, "ignored.txt", "ignore\n")
	runGit(t, repository, "add", "main.go", "ignored.txt")
	runGit(t, repository, "commit", "-m", "fixture")
	writeTestFile(t, repository, "gnosis.toml", "[[code_scopes]]\nname = \"app\"\nroot = \".\"\nlanguages = [\"go\"]\n")

	scope, err := ResolveScope(repository, "app")
	if err != nil {
		t.Fatal(err)
	}
	first, err := BuildWithAnalyzer(context.Background(), scope, testAnalyzer{})
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot, _ := CacheRoot()
	second, err := BuildWithAnalyzer(context.Background(), scope, testAnalyzer{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Generation != second.Generation || first.Documents != 1 || second.Status != "current" {
		t.Fatalf("first = %+v, second = %+v", first, second)
	}
	reader, err := Open(repository, "app")
	if err != nil {
		entries := []string{}
		filepath.Walk(cache, func(path string, _ os.FileInfo, _ error) error { entries = append(entries, path); return nil })
		t.Fatalf("%v; cache = %v", err, entries)
	}
	defer reader.Close()
	result := reader.Search("main", "go", 10)
	if result.Total != 1 || len(result.Symbols) != 1 || result.Symbols[0].Name != "Main" {
		t.Fatalf("search = %+v", result)
	}
	trace, err := reader.Trace(result.Symbols[0].ID, "outgoing", 10)
	if err != nil || len(trace.Relations) != 1 || trace.Relations[0].Resolution != analyzer.Resolved || len(trace.Relations[0].Candidates) != 1 {
		t.Fatalf("trace = %+v, err = %v", trace, err)
	}
	if err := reader.CheckCurrent(context.Background()); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repository, "main.go", "package main\nfunc Changed() {}\n")
	if err := reader.CheckCurrent(context.Background()); err != ErrNotCurrent {
		t.Fatalf("stale error = %v", err)
	}
	if _, err := BuildWithAnalyzer(context.Background(), scope, failingAnalyzer{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("failed build error = %v", err)
	}
	current, err := readCurrent(filepath.Join(cacheRoot, safeName(scope.Name)))
	if err != nil || current != first.Generation {
		t.Fatalf("current after failure = %q, %v", current, err)
	}
	if _, err := BuildWithAnalyzer(context.Background(), scope, failingAnalyzer{result: analyzer.AnalysisResult{Snapshot: "wrong", Complete: true}}); err == nil {
		t.Fatal("expected malformed analyzer result rejection")
	}
	current, err = readCurrent(filepath.Join(cacheRoot, safeName(scope.Name)))
	if err != nil || current != first.Generation {
		t.Fatalf("current after malformed result = %q, %v", current, err)
	}
	third, err := BuildWithAnalyzer(context.Background(), scope, testAnalyzer{})
	if err != nil || third.Generation == first.Generation {
		t.Fatalf("dirty build = %+v, err = %v", third, err)
	}
	changed, err := DisposeGeneration(repository, "app", first.Generation)
	if err != nil || !changed {
		t.Fatalf("dispose old generation = %v, %v", changed, err)
	}
	if _, err := DisposeGeneration(repository, "app", third.Generation); err == nil {
		t.Fatal("expected current-generation disposal rejection")
	}
	if data, err := os.ReadFile(filepath.Join(repository, "main.go")); err != nil || string(data) != "package main\nfunc Changed() {}\n" {
		t.Fatalf("source changed during disposal: %q, %v", data, err)
	}
}

func TestWriterLeaseRejectsCompetitor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writer.lock")
	first, err := acquireLease(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	if _, err := acquireLease(path); err == nil {
		t.Fatal("expected competing lease failure")
	}
}

func TestSourceSnapshotConfinesAndFiltersTrackedFiles(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init")
	runGit(t, repository, "config", "user.email", "test@example.test")
	runGit(t, repository, "config", "user.name", "Test")
	writeTestFile(t, repository, "main.go", "package main\n")
	writeTestFile(t, repository, "vendor/dependency.go", "package dependency\n")
	if err := os.WriteFile(filepath.Join(repository, "binary.go"), []byte{'p', 0, 'q'}, 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repository, "large.go", string(bytes.Repeat([]byte("x"), 64)))
	writeTestFile(t, repository, "generated.go", "// Code generated by fixture. DO NOT EDIT.\npackage generated\n")
	outside := filepath.Join(t.TempDir(), "secret.go")
	if err := os.WriteFile(outside, []byte("package secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repository, "escape.go")); err != nil {
		t.Fatal(err)
	}
	runGit(t, repository, "add", "-f", "main.go", "vendor/dependency.go", "binary.go", "large.go", "generated.go", "escape.go")
	runGit(t, repository, "commit", "-m", "fixture")
	writeTestFile(t, repository, "untracked.go", "package untracked\n")
	scope := Scope{Name: "app", Root: repository, Languages: []string{"go"}, MaxFiles: 10, MaxFileBytes: 32, MaxRecords: 100, MaxDiagnostics: 10, MaxResults: 10, MaxTraversal: 10}
	documents, clean, diagnostics, err := readSnapshot(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].Path != "main.go" {
		t.Fatalf("documents = %+v", documents)
	}
	if len(diagnostics) != 4 {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	writeTestFile(t, repository, "main.go", "package changed\n")
	_, dirty, _, err := readSnapshot(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if clean.Commit != dirty.Commit || clean.SourceDigest == dirty.SourceDigest || clean.ID == dirty.ID {
		t.Fatalf("clean = %+v, dirty = %+v", clean, dirty)
	}
}

func TestReaderTraversalIsBoundedAndCycleSafe(t *testing.T) {
	reader := &Reader{
		scope: Scope{Name: "app", MaxTraversal: 10},
		manifest: Manifest{
			Generation: "generation",
			Symbols: []Symbol{
				{ID: "a", DocumentID: "a-doc", Name: "A"},
				{ID: "b", DocumentID: "b-doc", Name: "B"},
				{ID: "c", DocumentID: "c-doc", Name: "C"},
			},
			Relations: []Relation{
				{ID: "ab", Source: "A", Target: "B"},
				{ID: "ba", Source: "B", Target: "A"},
				{ID: "bc", Source: "B", Target: "C"},
			},
		},
		byID: map[string]Symbol{
			"a": {ID: "a", DocumentID: "a-doc", Name: "A"},
			"b": {ID: "b", DocumentID: "b-doc", Name: "B"},
			"c": {ID: "c", DocumentID: "c-doc", Name: "C"},
		},
	}
	neighbors, err := reader.Neighbors("a", "outgoing", 1)
	if err != nil || neighbors.Total != 1 || len(neighbors.Symbols) != 1 || neighbors.Symbols[0].ID != "b" {
		t.Fatalf("neighbors = %+v, err = %v", neighbors, err)
	}
	path, err := reader.Path("a", "c", "outgoing", 3, 10)
	if err != nil || len(path.Symbols) != 3 || path.Symbols[2].ID != "c" || len(path.Relations) != 2 {
		t.Fatalf("path = %+v, err = %v", path, err)
	}
	bounded, err := reader.Path("a", "c", "outgoing", 3, 1)
	if err != nil || !bounded.Truncated || len(bounded.Symbols) != 0 {
		t.Fatalf("bounded path = %+v, err = %v", bounded, err)
	}
}

func TestGenerationValidationRejectsChangedRecords(t *testing.T) {
	manifest := Manifest{Schema: SchemaVersion, Symbols: []Symbol{{ID: "one", Name: "One"}}}
	manifest.Generation = digestJSON(manifest)
	if !validGeneration(manifest, manifest.Generation) {
		t.Fatal("expected valid generation")
	}
	manifest.Symbols[0].Name = "Changed"
	if validGeneration(manifest, manifest.Generation) {
		t.Fatal("expected changed generation to be rejected")
	}
}

func TestProviderDependencyDoesNotLeakOutsideAdapter(t *testing.T) {
	providerImport := "xberg-io/" + "tree-sitter-language-pack"
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		adapterPath := strings.Contains(filepath.ToSlash(path), "/languagepack/")
		if err != nil || info.IsDir() || filepath.Ext(path) != ".go" || adapterPath {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), providerImport) {
			t.Errorf("provider dependency leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", arguments, err, output)
	}
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
