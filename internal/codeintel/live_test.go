package codeintel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"gnosis/internal/codeintel/analyzer"
)

type fakeLiveSource struct {
	events chan eventHint
	errors chan error
	once   sync.Once
}

func newFakeLiveSource() *fakeLiveSource {
	return &fakeLiveSource{events: make(chan eventHint, liveEventLimit), errors: make(chan error, 1)}
}

func (source *fakeLiveSource) Events() <-chan eventHint              { return source.events }
func (source *fakeLiveSource) Errors() <-chan error                  { return source.errors }
func (source *fakeLiveSource) WatchDocuments([]SourceDocument) error { return nil }
func (source *fakeLiveSource) Close() error {
	source.once.Do(func() { close(source.events); close(source.errors) })
	return nil
}

type liveAnalyzer struct {
	mu     sync.Mutex
	modes  []analyzer.AnalysisMode
	closed int
}

type blockingLiveAnalyzer struct {
	liveAnalyzer
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (provider *blockingLiveAnalyzer) Analyze(ctx context.Context, request analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	if request.Mode == analyzer.Delta {
		provider.once.Do(func() { close(provider.started) })
		select {
		case <-ctx.Done():
			return analyzer.AnalysisResult{}, ctx.Err()
		case <-provider.release:
		}
	}
	return provider.liveAnalyzer.Analyze(ctx, request)
}

func (provider *liveAnalyzer) Analyze(_ context.Context, request analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	provider.mu.Lock()
	provider.modes = append(provider.modes, request.Mode)
	provider.mu.Unlock()
	result := analyzer.AnalysisResult{Snapshot: request.Snapshot, Complete: request.Mode == analyzer.Reset, Provenance: analyzer.AnalyzerProvenance{
		Implementation: "fake", ImplementationVersion: "1", ParserRelease: "1", ParserDigest: "digest", ABI: "14", QueryVersion: "1", NormalizerVersion: "1",
	}}
	for _, change := range request.Documents {
		if change.Kind == analyzer.Delete {
			continue
		}
		coverage := make([]analyzer.Coverage, 0, len(request.Capabilities))
		for _, capability := range request.Capabilities {
			coverage = append(coverage, analyzer.Coverage{Capability: capability, Level: analyzer.Complete})
		}
		result.Documents = append(result.Documents, analyzer.DocumentAnalysis{
			Path: change.Path, Language: change.Language, ContentDigest: change.ContentDigest, Coverage: coverage,
			Symbols: []analyzer.Symbol{{Kind: "function", Name: string(change.Content), Span: analyzer.Span{EndByte: len(change.Content)}}},
		})
	}
	return result, nil
}

func (provider *liveAnalyzer) Close() error {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.closed++
	return nil
}

func TestWorkspaceConvergesPublishesAndClosesExactlyOnce(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	repository := liveRepository(t)
	scope := liveTestScope(repository)
	scopeDir := filepath.Join(cache, "gnosis", "code-indexes", safeName(scope.Name))
	if err := os.MkdirAll(scopeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writer, err := acquireLease(filepath.Join(scopeDir, "writer.lock"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repository)
	if err != nil {
		t.Fatal(err)
	}
	source := newFakeLiveSource()
	provider := &liveAnalyzer{}
	workspace, err := openWorkspace(context.Background(), "", scope, scopeDir, writer, provider, source, root, realLiveClock{})
	if err != nil {
		t.Fatal(err)
	}
	initial := workspace.Status()
	if initial.State != LiveCurrent || initial.Observed != initial.Published || initial.Generation == "" {
		t.Fatalf("initial status = %+v", initial)
	}
	if err := initial.Validate(); err != nil {
		t.Fatal(err)
	}
	selected, err := readSelector(scopeDir)
	if err != nil || selected.Version != 2 || selected.Live == nil || selected.Live.WorkspaceEpoch != initial.Epoch {
		t.Fatalf("selector = %+v, %v", selected, err)
	}

	writeTestFile(t, repository, "main.go", "Changed")
	source.events <- eventHint{path: "main.go"}
	changed := waitLiveStatus(t, workspace, func(status LiveStatus) bool {
		return status.State == LiveCurrent && status.Generation != initial.Generation
	})
	if changed.Published != changed.Observed {
		t.Fatalf("changed status = %+v", changed)
	}
	provider.mu.Lock()
	if len(provider.modes) < 2 || provider.modes[0] != analyzer.Reset || provider.modes[1] != analyzer.Delta {
		t.Fatalf("analysis modes = %v", provider.modes)
	}
	provider.mu.Unlock()

	source.events <- eventHint{path: "main.go"}
	noOp := waitLiveStatus(t, workspace, func(status LiveStatus) bool {
		return status.State == LiveCurrent && status.Observed > changed.Observed
	})
	if noOp.Generation != changed.Generation || noOp.Published != noOp.Observed {
		t.Fatalf("no-op status = %+v, prior = %+v", noOp, changed)
	}

	var retained *Reader
	if err := workspace.ReadCurrent(context.Background(), func(view *Reader) error {
		retained = view
		result := view.Search("changed", "go", 10)
		if result.Total != 1 || result.Freshness == nil || result.Freshness.Published != noOp.Published {
			t.Fatalf("live search = %+v", result)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	assertClosedReader(t, retained)

	if _, err := BuildWithAnalyzer(context.Background(), scope, testAnalyzer{}); !errors.Is(err, ErrScopeBusy) {
		t.Fatalf("competing build error = %v", err)
	}
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Close(); err != nil {
		t.Fatal(err)
	}
	provider.mu.Lock()
	if provider.closed != 1 {
		t.Fatalf("analyzer close count = %d", provider.closed)
	}
	provider.mu.Unlock()
	if err := workspace.ReadCurrent(context.Background(), func(*Reader) error { return nil }); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("closed read error = %v", err)
	}
}

func TestSelectorReadsV2AndRejectsUnknownVersion(t *testing.T) {
	scopeDir := t.TempDir()
	generation := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := writeCurrent(scopeDir, generation); err != nil {
		t.Fatal(err)
	}
	selected, err := readSelector(scopeDir)
	if err != nil || selected.Version != 2 || selected.Live != nil {
		t.Fatalf("v2 selector = %+v, %v", selected, err)
	}
	if err := os.WriteFile(filepath.Join(scopeDir, "current"), []byte(`{"version":3,"generation_id":"`+generation+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSelector(scopeDir); err == nil {
		t.Fatal("expected unknown selector version error")
	}
}

func TestWorkspaceBoundsPendingReadsAndRecovers(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	repository := liveRepository(t)
	scope := liveTestScope(repository)
	scope.FreshnessWait = 50 * time.Millisecond
	scopeDir := filepath.Join(cache, "gnosis", "code-indexes", safeName(scope.Name))
	if err := os.MkdirAll(scopeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writer, err := acquireLease(filepath.Join(scopeDir, "writer.lock"))
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(repository)
	if err != nil {
		t.Fatal(err)
	}
	source := newFakeLiveSource()
	provider := &blockingLiveAnalyzer{started: make(chan struct{}), release: make(chan struct{})}
	workspace, err := openWorkspace(context.Background(), "", scope, scopeDir, writer, provider, source, root, realLiveClock{})
	if err != nil {
		t.Fatal(err)
	}
	defer workspace.Close()
	writeTestFile(t, repository, "main.go", "Pending")
	source.events <- eventHint{path: "main.go"}
	select {
	case <-provider.started:
	case <-time.After(3 * time.Second):
		t.Fatal("delta analysis did not start")
	}
	err = workspace.ReadCurrent(context.Background(), func(*Reader) error { return nil })
	var freshness *FreshnessError
	if !errors.As(err, &freshness) || freshness.State != LivePending || freshness.Observed <= freshness.Published {
		t.Fatalf("pending read error = %#v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := workspace.ReadCurrent(canceled, func(*Reader) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled read error = %v", err)
	}
	close(provider.release)
	waitLiveStatus(t, workspace, func(status LiveStatus) bool {
		return status.State == LiveCurrent && status.Observed == status.Published
	})
}

func TestFSNotifySourceOpensNotifiesAndCloses(t *testing.T) {
	repository := liveRepository(t)
	source, err := newFSNotifySource(repository, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.WatchDocuments([]SourceDocument{{Path: "main.go"}}); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, repository, "main.go", "notified")
	select {
	case hint := <-source.Events():
		if hint.path != "main.go" && !hint.full {
			t.Fatalf("hint = %+v", hint)
		}
	case err := <-source.Errors():
		t.Fatal(err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for filesystem notification")
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	for {
		select {
		case _, ok := <-source.Events():
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("event source did not close")
		}
	}
}

func liveRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	runGit(t, repository, "init")
	runGit(t, repository, "config", "user.email", "test@example.test")
	runGit(t, repository, "config", "user.name", "Test")
	writeTestFile(t, repository, "main.go", "Initial")
	runGit(t, repository, "add", "main.go")
	runGit(t, repository, "commit", "-m", "fixture")
	return repository
}

func liveTestScope(repository string) Scope {
	return Scope{
		Name: "app", Root: repository, Languages: []string{"go"}, Live: true, FreshnessWait: 500 * time.Millisecond,
		MaxFiles: 100, MaxFileBytes: 1 << 20, MaxRecords: 10_000, MaxDiagnostics: 100, MaxResults: 100, MaxTraversal: 100,
	}
}

func waitLiveStatus(t *testing.T, workspace *Workspace, ready func(LiveStatus) bool) LiveStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status := workspace.Status()
		if ready(status) {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("workspace did not converge: %+v", workspace.Status())
	return LiveStatus{}
}

func assertClosedReader(t *testing.T, reader *Reader) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("retained reader remained usable")
		}
	}()
	reader.Search("", "", 1)
}
