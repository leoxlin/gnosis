package codeintel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"gnosis/internal/codeintel/analyzer"
	"gnosis/internal/codeintel/languagepack"
)

type Watermark uint64
type LiveState string

const (
	LiveStarting LiveState = "starting"
	LivePending  LiveState = "pending"
	LiveCurrent  LiveState = "current"
	LiveDegraded LiveState = "degraded"
	LiveClosed   LiveState = "closed"

	liveDebounce       = 100 * time.Millisecond
	liveMaximumLatency = 2 * time.Second
	liveSafetyInterval = 30 * time.Second
	liveRetryLimit     = 5
	liveEventLimit     = 1_024
	livePathLimit      = 4_096
	liveDirectoryLimit = 4_096
	liveBatchLimit     = 512
	liveRetention      = 3
	liveShutdown       = 5 * time.Second
	liveReasonLimit    = 512
)

type LiveBounds struct {
	Debounce       string `json:"debounce"`
	MaximumLatency string `json:"maximum_latency"`
	SafetyInterval string `json:"safety_interval"`
	FreshnessWait  string `json:"freshness_wait"`
	Shutdown       string `json:"shutdown"`
	EventQueue     int    `json:"event_queue"`
	Paths          int    `json:"paths"`
	Directories    int    `json:"directories"`
	Batch          int    `json:"batch"`
	Retries        int    `json:"retries"`
	Retention      int    `json:"retention"`
}

type LiveFreshness struct {
	State     LiveState `json:"state"`
	Epoch     string    `json:"workspace_epoch"`
	Observed  Watermark `json:"observed_watermark"`
	Published Watermark `json:"published_watermark"`
}

type LiveStatus = StatusResult

func (state LiveState) valid() bool {
	return state == LiveStarting || state == LivePending || state == LiveCurrent || state == LiveDegraded || state == LiveClosed
}

func (status StatusResult) Validate() error {
	if !status.Live {
		return nil
	}
	if !status.State.valid() || status.Status != string(status.State) || status.Epoch == "" {
		return errors.New("invalid live code-index status")
	}
	if status.Published > status.Observed || status.State == LiveCurrent && status.Published != status.Observed {
		return errors.New("invalid live code-index watermarks")
	}
	if len(status.Reason) > liveReasonLimit || len(status.Guidance) > liveReasonLimit || status.Bounds == nil {
		return errors.New("invalid live code-index status metadata")
	}
	return nil
}

type FreshnessError struct {
	State          LiveState `json:"state"`
	Observed       Watermark `json:"observed_watermark"`
	Published      Watermark `json:"published_watermark"`
	LastGeneration string    `json:"last_generation,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	Guidance       string    `json:"guidance"`
}

func (failure *FreshnessError) Error() string {
	return fmt.Sprintf("code index is %s (observed=%d, published=%d): %s", failure.State, failure.Observed, failure.Published, failure.Guidance)
}

func (failure *FreshnessError) Unwrap() error {
	if failure.State == LiveClosed {
		return os.ErrClosed
	}
	return ErrNotCurrent
}

type LiveConfig struct {
	Workspace string
	Scope     string
}

type eventHint struct {
	path string
	full bool
}

type eventSource interface {
	Events() <-chan eventHint
	Errors() <-chan error
	WatchDocuments([]SourceDocument) error
	Close() error
}

type liveLease interface{ Close() error }

type liveClock interface {
	After(time.Duration) <-chan time.Time
}

type realLiveClock struct{}

func (realLiveClock) After(duration time.Duration) <-chan time.Time { return time.After(duration) }

type Workspace struct {
	scope      Scope
	configRoot string
	scopeDir   string
	epoch      string
	provider   analyzer.Analyzer
	events     eventSource
	sourceRoot *os.Root
	lease      liveLease
	clock      liveClock

	ctx    context.Context
	cancel context.CancelFunc
	wake   chan struct{}
	wg     sync.WaitGroup
	reads  sync.WaitGroup

	mu           sync.Mutex
	state        LiveState
	observed     Watermark
	published    Watermark
	reason       string
	guidance     string
	pending      map[string]bool
	full         bool
	manifest     *Manifest
	documents    map[string]SourceDocument
	pins         map[string]int
	lastFallback string
	closing      bool
	closeOnce    sync.Once
	closeErr     error
}

func OpenWorkspace(ctx context.Context, config LiveConfig) (*Workspace, error) {
	scope, err := ResolveScope(config.Workspace, config.Scope)
	if err != nil {
		return nil, err
	}
	if !scope.Live {
		return nil, fmt.Errorf("code scope %q does not enable live indexing", scope.Name)
	}
	root, err := CacheRoot()
	if err != nil {
		return nil, err
	}
	scopeDir := filepath.Join(root, safeName(scope.Name))
	if err := os.MkdirAll(scopeDir, 0o700); err != nil {
		return nil, err
	}
	writer, err := acquireLease(filepath.Join(scopeDir, "writer.lock"))
	if err != nil {
		return nil, fmt.Errorf("%w: scope %q is owned by another live host or explicit build; stop it and retry", ErrScopeBusy, scope.Name)
	}
	parserCache, err := languagepack.DefaultCacheDir()
	if err != nil {
		writer.Close()
		return nil, err
	}
	provider, err := languagepack.New(parserCache, scope.Languages)
	if err != nil {
		writer.Close()
		return nil, err
	}
	source, err := newFSNotifySource(scope.Root, liveDirectoryLimit)
	if err != nil {
		provider.Close()
		writer.Close()
		return nil, fmt.Errorf("open live observation for scope %q: %w", scope.Name, err)
	}
	rootHandle, err := os.OpenRoot(scope.Root)
	if err != nil {
		source.Close()
		provider.Close()
		writer.Close()
		return nil, err
	}
	return openWorkspace(ctx, config.Workspace, scope, scopeDir, writer, provider, source, rootHandle, realLiveClock{})
}

func openWorkspace(ctx context.Context, configRoot string, scope Scope, scopeDir string, writer liveLease, provider analyzer.Analyzer, source eventSource, root *os.Root, clock liveClock) (*Workspace, error) {
	workerCtx, cancel := context.WithCancel(ctx)
	workspace := &Workspace{
		scope: scope, configRoot: configRoot, scopeDir: scopeDir, epoch: newWorkspaceEpoch(), provider: provider,
		events: source, sourceRoot: root, lease: writer, clock: clock, ctx: workerCtx,
		cancel: cancel, wake: make(chan struct{}, 1), state: LiveStarting, observed: 1,
		pending: map[string]bool{}, documents: map[string]SourceDocument{}, pins: map[string]int{},
	}
	workspace.wg.Add(1)
	go workspace.observe()
	if err := workspace.reconcile(workerCtx, true); err != nil {
		workspace.degrade(err, recoveryGuidance(err))
		workspace.Close()
		return nil, fmt.Errorf("establish live baseline for scope %q: %w", scope.Name, err)
	}
	workspace.wg.Add(1)
	go workspace.run()
	return workspace, nil
}

func (workspace *Workspace) observe() {
	defer workspace.wg.Done()
	for {
		select {
		case <-workspace.ctx.Done():
			return
		case hint, ok := <-workspace.events.Events():
			if !ok {
				return
			}
			workspace.recordHint(hint)
		case err, ok := <-workspace.events.Errors():
			if ok && err != nil {
				workspace.recordHint(eventHint{full: true})
				workspace.mu.Lock()
				workspace.reason = bounded(err.Error(), liveReasonLimit)
				workspace.mu.Unlock()
			}
		}
	}
}

func (workspace *Workspace) recordHint(hint eventHint) {
	workspace.mu.Lock()
	if workspace.closing {
		workspace.mu.Unlock()
		return
	}
	workspace.observed++
	if hint.full || len(workspace.pending) >= livePathLimit {
		workspace.full = true
		clear(workspace.pending)
	} else if hint.path != "" {
		workspace.pending[hint.path] = true
	}
	if workspace.state != LiveStarting {
		workspace.state = LivePending
	}
	workspace.mu.Unlock()
	select {
	case workspace.wake <- struct{}{}:
	default:
	}
}

func (workspace *Workspace) run() {
	defer workspace.wg.Done()
	safety := time.NewTicker(liveSafetyInterval)
	defer safety.Stop()
	for {
		select {
		case <-workspace.ctx.Done():
			return
		case <-safety.C:
			workspace.recordHint(eventHint{full: true})
		case <-workspace.wake:
			select {
			case <-workspace.ctx.Done():
				return
			case <-workspace.clock.After(liveDebounce):
			}
			var err error
			for attempt := 0; attempt < liveRetryLimit; attempt++ {
				err = workspace.reconcile(workspace.ctx, false)
				if err == nil || errors.Is(err, context.Canceled) {
					break
				}
				if requiresRestart(err) {
					break
				}
				select {
				case <-workspace.ctx.Done():
					return
				case <-workspace.clock.After(time.Duration(attempt+1) * 100 * time.Millisecond):
				}
			}
			if err != nil && !errors.Is(err, context.Canceled) {
				workspace.degrade(err, recoveryGuidance(err))
			}
		}
	}
}

func (workspace *Workspace) reconcile(ctx context.Context, reset bool) error {
	if workspace.configRoot != "" {
		configured, err := ResolveScope(workspace.configRoot, workspace.scope.Name)
		if err != nil {
			return fmt.Errorf("gnosis configuration changed; restart required: %w", err)
		}
		if digestJSON(configured) != digestJSON(workspace.scope) {
			return errors.New("gnosis configuration changed; restart required")
		}
	}
	workspace.mu.Lock()
	captured := workspace.observed
	full := workspace.full
	workspace.full = false
	clear(workspace.pending)
	previous := workspace.manifest
	priorDocuments := cloneDocuments(workspace.documents)
	workspace.mu.Unlock()

	documents, snapshot, sourceDiagnostics, err := readSnapshot(ctx, workspace.scope)
	if err != nil {
		return err
	}
	if err := workspace.events.WatchDocuments(documents); err != nil {
		return err
	}
	currentDocuments := documentMap(documents)
	changes := documentChanges(priorDocuments, currentDocuments)
	if len(changes) > liveBatchLimit {
		reset = true
	}
	if full && previous != nil && previous.Snapshot.ID != snapshot.ID {
		reset = true
	}
	if previous == nil {
		reset = true
	}
	if previous != nil && previous.Snapshot.ParserDigest != snapshot.ParserDigest {
		return errors.New("trusted parser manifest changed; restart required")
	}
	if len(changes) == 0 && previous != nil && previous.Snapshot.ID == snapshot.ID {
		return workspace.selectCurrent(previous, currentDocuments, captured)
	}
	if reset {
		changes = resetChanges(documents)
	}
	request := analyzer.AnalysisRequest{Snapshot: analyzer.SnapshotID(snapshot.ID), Mode: analyzer.Delta, Documents: changes, Capabilities: liveCapabilities()}
	if reset {
		request.Mode = analyzer.Reset
	}
	result, err := workspace.provider.Analyze(ctx, request)
	if err != nil {
		return err
	}
	digests := map[string]string{}
	for _, document := range documents {
		digests[document.Path] = document.ContentDigest
	}
	if err := analyzer.ValidateResult(request, result, digests); err != nil {
		return fmt.Errorf("invalid analyzer result: %w", err)
	}
	verifiedDocuments, verifiedSnapshot, _, err := readSnapshot(ctx, workspace.scope)
	if err != nil {
		return err
	}
	if verifiedSnapshot.ID != snapshot.ID {
		workspace.recordHint(eventHint{full: true})
		return errors.New("source changed during live analysis; full reconciliation scheduled")
	}
	manifest, err := mergeLiveManifest(workspace.scope, snapshot, previous, result, sourceDiagnostics, currentDocuments)
	if err != nil {
		return err
	}
	manifest.CreatedAt = time.Time{}
	if err := writeGeneration(workspace.scopeDir, &manifest); err != nil {
		return err
	}
	return workspace.selectCurrent(&manifest, documentMap(verifiedDocuments), captured)
}

func (workspace *Workspace) selectCurrent(manifest *Manifest, documents map[string]SourceDocument, captured Watermark) error {
	workspace.mu.Lock()
	if workspace.closing {
		workspace.mu.Unlock()
		return context.Canceled
	}
	if workspace.observed != captured {
		workspace.mu.Unlock()
		return errors.New("new source state arrived before publication")
	}
	previous := ""
	if workspace.manifest != nil {
		previous = workspace.manifest.Generation
	}
	selected := selector{Version: 2, GenerationID: manifest.Generation, Live: &liveSelection{WorkspaceEpoch: workspace.epoch, VerifiedWatermark: captured}}
	if err := writeSelector(workspace.scopeDir, selected); err != nil {
		workspace.mu.Unlock()
		return err
	}
	workspace.manifest = manifest
	workspace.documents = documents
	workspace.published = captured
	workspace.state = LiveCurrent
	workspace.reason = ""
	workspace.guidance = ""
	if previous != "" && previous != manifest.Generation {
		workspace.lastFallback = previous
	}
	workspace.mu.Unlock()
	workspace.retainGenerations()
	return nil
}

func (workspace *Workspace) degrade(err error, guidance string) {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	if workspace.closing {
		return
	}
	workspace.state = LiveDegraded
	workspace.reason = bounded(err.Error(), liveReasonLimit)
	workspace.guidance = bounded(guidance, liveReasonLimit)
}

func (workspace *Workspace) Status() LiveStatus {
	workspace.mu.Lock()
	defer workspace.mu.Unlock()
	status := StatusResult{
		Scope: workspace.scope.Name, Status: string(workspace.state), Live: true, State: workspace.state,
		Epoch: workspace.epoch, Observed: workspace.observed, Published: workspace.published,
		Reason: workspace.reason, Guidance: workspace.guidance,
	}
	bounds := workspace.bounds()
	status.Bounds = &bounds
	if workspace.manifest != nil {
		manifest := workspace.manifest
		status.Generation = manifest.Generation
		status.Snapshot = &manifest.Snapshot
		status.Provenance = &manifest.Provenance
		status.Documents = len(manifest.Documents)
		status.Symbols = len(manifest.Symbols)
		status.Relations = len(manifest.Relations)
		status.Diagnostics = len(manifest.Diagnostics)
	}
	return status
}

func (workspace *Workspace) ReadCurrent(ctx context.Context, callback func(*Reader) error) error {
	if callback == nil {
		return errors.New("read callback is required")
	}
	deadline := time.NewTimer(workspace.scope.FreshnessWait)
	defer deadline.Stop()
	for {
		workspace.mu.Lock()
		state := workspace.state
		if !workspace.closing && state == LiveCurrent && workspace.observed == workspace.published && workspace.manifest != nil {
			generation := workspace.manifest.Generation
			freshness := &LiveFreshness{State: state, Epoch: workspace.epoch, Observed: workspace.observed, Published: workspace.published}
			workspace.pins[generation]++
			workspace.reads.Add(1)
			workspace.mu.Unlock()
			reader, err := openGeneration(workspace.scope, workspace.scopeDir, generation)
			if err == nil {
				reader.freshness = freshness
				err = callback(reader)
				reader.Close()
			}
			workspace.mu.Lock()
			workspace.pins[generation]--
			if workspace.pins[generation] == 0 {
				delete(workspace.pins, generation)
			}
			workspace.mu.Unlock()
			workspace.reads.Done()
			return err
		}
		failure := workspace.freshnessErrorLocked()
		workspace.mu.Unlock()
		if state == LiveDegraded || state == LiveClosed || workspace.closing {
			return failure
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return failure
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (workspace *Workspace) freshnessErrorLocked() error {
	generation := ""
	if workspace.manifest != nil {
		generation = workspace.manifest.Generation
	}
	guidance := workspace.guidance
	if guidance == "" {
		guidance = "wait for live reconciliation and retry"
	}
	return &FreshnessError{State: workspace.state, Observed: workspace.observed, Published: workspace.published, LastGeneration: generation, Reason: workspace.reason, Guidance: guidance}
}

func (workspace *Workspace) Close() error {
	workspace.closeOnce.Do(func() {
		workspace.mu.Lock()
		workspace.closing = true
		workspace.state = LiveClosed
		workspace.guidance = "restart the owning host to reopen live indexing"
		workspace.mu.Unlock()
		workspace.cancel()
		workspace.events.Close()
		waited := make(chan struct{})
		go func() {
			workspace.wg.Wait()
			workspace.reads.Wait()
			close(waited)
		}()
		select {
		case <-waited:
		case <-time.After(liveShutdown):
			workspace.closeErr = errors.New("live workspace shutdown exceeded its bound")
		}
		if err := workspace.closeChildren(); workspace.closeErr == nil {
			workspace.closeErr = err
		}
	})
	return workspace.closeErr
}

func (workspace *Workspace) closeChildren() error {
	var failures []error
	if workspace.sourceRoot != nil {
		failures = append(failures, workspace.sourceRoot.Close())
		workspace.sourceRoot = nil
	}
	if workspace.provider != nil {
		failures = append(failures, workspace.provider.Close())
		workspace.provider = nil
	}
	if workspace.lease != nil {
		failures = append(failures, workspace.lease.Close())
		workspace.lease = nil
	}
	return errors.Join(failures...)
}

func (workspace *Workspace) bounds() LiveBounds {
	return LiveBounds{
		Debounce: liveDebounce.String(), MaximumLatency: liveMaximumLatency.String(), SafetyInterval: liveSafetyInterval.String(),
		FreshnessWait: workspace.scope.FreshnessWait.String(), Shutdown: liveShutdown.String(), EventQueue: liveEventLimit,
		Paths: livePathLimit, Directories: liveDirectoryLimit, Batch: liveBatchLimit, Retries: liveRetryLimit, Retention: liveRetention,
	}
}

func (workspace *Workspace) retainGenerations() {
	entries, err := os.ReadDir(filepath.Join(workspace.scopeDir, "generations"))
	if err != nil {
		return
	}
	workspace.mu.Lock()
	protected := map[string]bool{workspace.manifest.Generation: true, workspace.lastFallback: true}
	for generation := range workspace.pins {
		protected[generation] = true
	}
	workspace.mu.Unlock()
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && validGenerationID(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	for len(names) > liveRetention {
		name := names[0]
		names = names[1:]
		if protected[name] {
			continue
		}
		_ = os.RemoveAll(filepath.Join(workspace.scopeDir, "generations", name))
	}
}

func openGeneration(scope Scope, scopeDir, generation string) (*Reader, error) {
	data, err := os.ReadFile(filepath.Join(scopeDir, "generations", generation, "generation.json"))
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := jsonUnmarshalGeneration(data, &manifest, generation); err != nil {
		return nil, err
	}
	reader := &Reader{scope: scope, manifest: manifest, byID: map[string]Symbol{}}
	for _, symbol := range manifest.Symbols {
		reader.byID[symbol.ID] = symbol
	}
	return reader, nil
}

func jsonUnmarshalGeneration(data []byte, manifest *Manifest, generation string) error {
	if err := json.Unmarshal(data, manifest); err != nil || !validGeneration(*manifest, generation) {
		return errors.New("code-index generation is corrupt")
	}
	return nil
}

func mergeLiveManifest(scope Scope, snapshot Snapshot, previous *Manifest, result analyzer.AnalysisResult, sourceDiagnostics []Diagnostic, current map[string]SourceDocument) (Manifest, error) {
	if previous == nil || result.Complete {
		return normalizedManifest(scope, snapshot, result, sourceDiagnostics)
	}
	changed := map[string]bool{}
	for _, document := range result.Documents {
		changed[document.Path] = true
	}
	documents := append([]analyzer.DocumentAnalysis(nil), result.Documents...)
	for _, document := range previous.Documents {
		if changed[document.Path] {
			continue
		}
		if _, exists := current[document.Path]; !exists {
			continue
		}
		converted := analyzer.DocumentAnalysis{Path: document.Path, Language: document.Language, ContentDigest: document.ContentDigest, Coverage: document.Coverage}
		for _, symbol := range previous.Symbols {
			if symbol.Path == document.Path {
				converted.Symbols = append(converted.Symbols, analyzer.Symbol{Kind: symbol.Kind, Name: symbol.Name, QualifiedName: symbol.QualifiedName, Signature: symbol.Signature, Span: symbol.Span})
			}
		}
		for _, relation := range previous.Relations {
			if relation.Path == document.Path {
				converted.Relations = append(converted.Relations, analyzer.Relation{Kind: relation.Kind, Source: relation.Source, Target: relation.Target, Candidates: relation.Candidates, Evidence: relation.Evidence, Resolution: relation.Resolution, Span: relation.Span})
			}
		}
		for _, diagnostic := range previous.Diagnostics {
			if diagnostic.Path == document.Path {
				converted.Diagnostics = append(converted.Diagnostics, analyzer.Diagnostic{Category: diagnostic.Category, Severity: diagnostic.Severity, Message: diagnostic.Message, Span: diagnostic.Span, Usable: true})
			}
		}
		documents = append(documents, converted)
	}
	result.Documents = documents
	result.Complete = true
	return normalizedManifest(scope, snapshot, result, sourceDiagnostics)
}

func documentMap(documents []SourceDocument) map[string]SourceDocument {
	result := make(map[string]SourceDocument, len(documents))
	for _, document := range documents {
		result[document.Path] = document
	}
	return result
}

func cloneDocuments(documents map[string]SourceDocument) map[string]SourceDocument {
	result := make(map[string]SourceDocument, len(documents))
	for path, document := range documents {
		result[path] = document
	}
	return result
}

func documentChanges(previous, current map[string]SourceDocument) []analyzer.DocumentChange {
	changes := make([]analyzer.DocumentChange, 0)
	for path, document := range current {
		prior, exists := previous[path]
		if !exists || prior.ContentDigest != document.ContentDigest || prior.Language != document.Language {
			changes = append(changes, analyzer.DocumentChange{Kind: analyzer.Upsert, Path: path, Language: document.Language, Content: document.Content, ContentDigest: document.ContentDigest})
		}
	}
	for path := range previous {
		if _, exists := current[path]; !exists {
			changes = append(changes, analyzer.DocumentChange{Kind: analyzer.Delete, Path: path})
		}
	}
	slices.SortFunc(changes, func(a, b analyzer.DocumentChange) int { return strings.Compare(a.Path, b.Path) })
	return changes
}

func resetChanges(documents []SourceDocument) []analyzer.DocumentChange {
	changes := make([]analyzer.DocumentChange, 0, len(documents))
	for _, document := range documents {
		changes = append(changes, analyzer.DocumentChange{Kind: analyzer.Upsert, Path: document.Path, Language: document.Language, Content: document.Content, ContentDigest: document.ContentDigest})
	}
	return changes
}

func liveCapabilities() []analyzer.Capability {
	return []analyzer.Capability{analyzer.Parse, analyzer.Structure, analyzer.Imports, analyzer.Exports, analyzer.Definitions, analyzer.References, analyzer.Calls, analyzer.Injections, analyzer.SemanticResolution}
}

func newWorkspaceEpoch() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		panic(err)
	}
	return hex.EncodeToString(value)
}

func recoveryGuidance(err error) string {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "parser") || strings.Contains(message, "configuration") || strings.Contains(message, "abi") {
		return "fix parser/configuration integrity and restart the owning host"
	}
	return "fix the reported watcher, source, analyzer, or storage failure; gnosis will retry with full reconciliation"
}

func requiresRestart(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "restart required") || strings.Contains(message, "parser integrity") || strings.Contains(message, "abi")
}
