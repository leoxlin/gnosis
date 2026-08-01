package analyzer

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestValidateResultContract(t *testing.T) {
	request := AnalysisRequest{
		Snapshot: "snapshot", Mode: Delta,
		Documents:    []DocumentChange{{Kind: Upsert, Path: "main.go", Language: "go", ContentDigest: "digest"}},
		Capabilities: []Capability{Parse},
	}
	valid := AnalysisResult{
		Snapshot:   "snapshot",
		Provenance: AnalyzerProvenance{Implementation: "fake", ImplementationVersion: "1", ParserRelease: "1", ParserDigest: "digest", ABI: "14", QueryVersion: "1", NormalizerVersion: "1"},
		Documents: []DocumentAnalysis{{
			Path: "main.go", Language: "go", ContentDigest: "digest",
			Coverage: []Coverage{{Capability: Parse, Level: Complete}},
		}},
	}
	if err := ValidateResult(request, valid, map[string]string{"main.go": "digest"}); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*AnalysisResult){
		"stale snapshot": func(result *AnalysisResult) { result.Snapshot = "old" },
		"omitted upsert": func(result *AnalysisResult) { result.Documents = nil },
		"duplicate":      func(result *AnalysisResult) { result.Documents = append(result.Documents, result.Documents[0]) },
		"unknown indirect": func(result *AnalysisResult) {
			result.Documents = append(result.Documents, DocumentAnalysis{Path: "other.go", Language: "go", ContentDigest: "other", Coverage: []Coverage{{Capability: Parse, Level: Complete}}})
		},
		"digest mismatch": func(result *AnalysisResult) { result.Documents[0].ContentDigest = "changed" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			result := valid
			result.Documents = append([]DocumentAnalysis(nil), valid.Documents...)
			mutate(&result)
			if err := ValidateResult(request, result, map[string]string{"main.go": "digest"}); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	complete := valid
	complete.Complete = true
	if err := ValidateResult(request, complete, map[string]string{"main.go": "digest", "other.go": "other"}); err == nil {
		t.Fatal("expected incomplete complete-result rejection")
	}
	deleteRequest := AnalysisRequest{Snapshot: "snapshot", Mode: Delta, Documents: []DocumentChange{{Kind: Delete, Path: "main.go"}}, Capabilities: []Capability{Parse}}
	if err := ValidateResult(deleteRequest, valid, map[string]string{}); err == nil {
		t.Fatal("expected delete-result rejection")
	}
}

func TestCanonicalize(t *testing.T) {
	result := AnalysisResult{Documents: []DocumentAnalysis{{Path: "z.go"}, {Path: "a.go"}}}
	Canonicalize(&result)
	if result.Documents[0].Path != "a.go" {
		t.Fatalf("documents = %+v", result.Documents)
	}
}

type fakeAnalyzer struct {
	mu       sync.Mutex
	closed   bool
	closeN   int
	response AnalysisResult
	err      error
}

func (fake *fakeAnalyzer) Analyze(ctx context.Context, request AnalysisRequest) (AnalysisResult, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.closed {
		return AnalysisResult{}, ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return AnalysisResult{}, err
	}
	if fake.err != nil {
		return AnalysisResult{}, fake.err
	}
	return fake.response, nil
}

func (fake *fakeAnalyzer) Close() error {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	fake.closeN++
	if fake.closed {
		return ErrClosed
	}
	fake.closed = true
	return nil
}

func TestAnalyzerLifecycleAndCancellation(t *testing.T) {
	fake := &fakeAnalyzer{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fake.Analyze(ctx, AnalysisRequest{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Analyze error = %v", err)
	}
	if err := fake.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fake.Close(); !errors.Is(err, ErrClosed) || fake.closeN != 2 {
		t.Fatalf("second close = %v, count = %d", err, fake.closeN)
	}
}
