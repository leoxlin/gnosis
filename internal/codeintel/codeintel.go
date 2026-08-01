package codeintel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gnosis/internal/codeintel/analyzer"
	"gnosis/internal/codeintel/languagepack"
	"gnosis/internal/vault"
)

const SchemaVersion = 1

var (
	ErrScopeBusy  = errors.New("code scope is busy")
	ErrNotCurrent = errors.New("code index is not current; rebuild it")
)

type Scope struct {
	Name           string        `json:"name"`
	Root           string        `json:"root"`
	Languages      []string      `json:"languages"`
	Live           bool          `json:"live"`
	FreshnessWait  time.Duration `json:"freshness_wait"`
	MaxFiles       int           `json:"max_files"`
	MaxFileBytes   int64         `json:"max_file_bytes"`
	MaxRecords     int           `json:"max_records"`
	MaxDiagnostics int           `json:"max_diagnostics"`
	MaxResults     int           `json:"max_results"`
	MaxTraversal   int           `json:"max_traversal"`
}

type Snapshot struct {
	ID                  string `json:"id"`
	Scope               string `json:"scope"`
	Repository          string `json:"repository"`
	Commit              string `json:"commit"`
	Branch              string `json:"branch,omitempty"`
	SourceDigest        string `json:"source_digest"`
	ConfigurationDigest string `json:"configuration_digest"`
	AnalyzerDigest      string `json:"analyzer_digest"`
	ParserDigest        string `json:"parser_digest"`
	QueryDigest         string `json:"query_digest"`
	NormalizerDigest    string `json:"normalizer_digest"`
	SchemaDigest        string `json:"schema_digest"`
	BoundsDigest        string `json:"bounds_digest"`
}

type SourceDocument struct {
	Path          string
	Language      string
	Content       []byte
	ContentDigest string
}

type Manifest struct {
	Schema      int                         `json:"schema"`
	Generation  string                      `json:"generation"`
	Snapshot    Snapshot                    `json:"snapshot"`
	Provenance  analyzer.AnalyzerProvenance `json:"provenance"`
	CreatedAt   time.Time                   `json:"created_at"`
	Documents   []Document                  `json:"documents"`
	Symbols     []Symbol                    `json:"symbols"`
	Relations   []Relation                  `json:"relations"`
	Diagnostics []Diagnostic                `json:"diagnostics"`
}

type Document struct {
	ID            string              `json:"id"`
	Path          string              `json:"path"`
	Language      string              `json:"language"`
	ContentDigest string              `json:"content_digest"`
	Coverage      []analyzer.Coverage `json:"coverage"`
}

type Symbol struct {
	ID            string        `json:"id"`
	DocumentID    string        `json:"document_id"`
	Path          string        `json:"path"`
	Language      string        `json:"language"`
	Kind          string        `json:"kind"`
	Name          string        `json:"name"`
	QualifiedName string        `json:"qualified_name,omitempty"`
	Signature     string        `json:"signature,omitempty"`
	Span          analyzer.Span `json:"span"`
}

type Relation struct {
	ID         string                   `json:"id"`
	DocumentID string                   `json:"document_id"`
	Path       string                   `json:"path"`
	Kind       string                   `json:"kind"`
	Source     string                   `json:"source"`
	Target     string                   `json:"target,omitempty"`
	Candidates []string                 `json:"candidates,omitempty"`
	Evidence   analyzer.EvidenceLevel   `json:"evidence"`
	Resolution analyzer.ResolutionState `json:"resolution"`
	Span       analyzer.Span            `json:"span"`
}

type Diagnostic struct {
	ID       string         `json:"id"`
	Path     string         `json:"path"`
	Language string         `json:"language"`
	Category string         `json:"category"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Span     *analyzer.Span `json:"span,omitempty"`
}

type BuildResult struct {
	Scope       string `json:"scope"`
	Generation  string `json:"generation"`
	Status      string `json:"status"`
	Documents   int    `json:"documents"`
	Symbols     int    `json:"symbols"`
	Relations   int    `json:"relations"`
	Diagnostics int    `json:"diagnostics"`
}

type SearchResult struct {
	Scope      string                      `json:"scope"`
	Generation string                      `json:"generation"`
	Snapshot   Snapshot                    `json:"snapshot"`
	Provenance analyzer.AnalyzerProvenance `json:"provenance"`
	Total      int                         `json:"total"`
	Truncated  bool                        `json:"truncated"`
	Symbols    []Symbol                    `json:"symbols"`
	Coverage   []analyzer.Coverage         `json:"coverage,omitempty"`
	Freshness  *LiveFreshness              `json:"freshness,omitempty"`
}

type SymbolResult struct {
	Scope      string                      `json:"scope"`
	Generation string                      `json:"generation"`
	Snapshot   Snapshot                    `json:"snapshot"`
	Provenance analyzer.AnalyzerProvenance `json:"provenance"`
	Symbol     Symbol                      `json:"symbol"`
	Coverage   []analyzer.Coverage         `json:"coverage,omitempty"`
	Freshness  *LiveFreshness              `json:"freshness,omitempty"`
}

type DiagnosticResult struct {
	Scope       string                      `json:"scope"`
	Generation  string                      `json:"generation"`
	Snapshot    Snapshot                    `json:"snapshot"`
	Provenance  analyzer.AnalyzerProvenance `json:"provenance"`
	Total       int                         `json:"total"`
	Truncated   bool                        `json:"truncated"`
	Diagnostics []Diagnostic                `json:"diagnostics"`
	Coverage    []analyzer.Coverage         `json:"coverage,omitempty"`
	Freshness   *LiveFreshness              `json:"freshness,omitempty"`
}

type TraceResult struct {
	Mode       string                      `json:"mode"`
	Scope      string                      `json:"scope"`
	Generation string                      `json:"generation"`
	Snapshot   Snapshot                    `json:"snapshot"`
	Provenance analyzer.AnalyzerProvenance `json:"provenance"`
	Total      int                         `json:"total"`
	Truncated  bool                        `json:"truncated"`
	Relations  []Relation                  `json:"relations"`
	Symbols    []Symbol                    `json:"symbols,omitempty"`
	Coverage   []analyzer.Coverage         `json:"coverage,omitempty"`
	Freshness  *LiveFreshness              `json:"freshness,omitempty"`
}

type StatusResult struct {
	Scope       string                       `json:"scope"`
	Status      string                       `json:"status"`
	Generation  string                       `json:"generation,omitempty"`
	Snapshot    *Snapshot                    `json:"snapshot,omitempty"`
	Provenance  *analyzer.AnalyzerProvenance `json:"provenance,omitempty"`
	Documents   int                          `json:"documents"`
	Symbols     int                          `json:"symbols"`
	Relations   int                          `json:"relations"`
	Diagnostics int                          `json:"diagnostics"`
	Live        bool                         `json:"live,omitempty"`
	State       LiveState                    `json:"state,omitempty"`
	Epoch       string                       `json:"workspace_epoch,omitempty"`
	Observed    Watermark                    `json:"observed_watermark,omitempty"`
	Published   Watermark                    `json:"published_watermark,omitempty"`
	Reason      string                       `json:"reason,omitempty"`
	Guidance    string                       `json:"guidance,omitempty"`
	Bounds      *LiveBounds                  `json:"bounds,omitempty"`
}

func ResolveScope(workspace, name string) (Scope, error) {
	configured, err := vault.CodeScope(workspace, name)
	if err != nil {
		return Scope{}, err
	}
	return Scope{
		Name: configured.Name, Root: configured.Root, Languages: append([]string(nil), configured.Languages...),
		Live: configured.Live, FreshnessWait: configured.FreshnessWaitDuration(),
		MaxFiles: configured.MaxFiles, MaxFileBytes: configured.MaxFileBytes,
		MaxRecords: configured.MaxRecords, MaxDiagnostics: configured.MaxDiagnostics,
		MaxResults: configured.MaxResults, MaxTraversal: configured.MaxTraversal,
	}, nil
}

func CacheRoot() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "gnosis", "code-indexes"), nil
}

func Build(ctx context.Context, workspace, scopeName string) (BuildResult, error) {
	scope, err := ResolveScope(workspace, scopeName)
	if err != nil {
		return BuildResult{}, err
	}
	parserCache, err := languagepack.DefaultCacheDir()
	if err != nil {
		return BuildResult{}, err
	}
	provider, err := languagepack.New(parserCache, scope.Languages)
	if err != nil {
		return BuildResult{}, err
	}
	defer provider.Close()
	return BuildWithAnalyzer(ctx, scope, provider)
}

func BuildWithAnalyzer(ctx context.Context, scope Scope, provider analyzer.Analyzer) (BuildResult, error) {
	root, err := CacheRoot()
	if err != nil {
		return BuildResult{}, err
	}
	scopeDir := filepath.Join(root, safeName(scope.Name))
	if err := os.MkdirAll(scopeDir, 0o700); err != nil {
		return BuildResult{}, err
	}
	lease, err := acquireLease(filepath.Join(scopeDir, "writer.lock"))
	if err != nil {
		return BuildResult{}, fmt.Errorf("%w: %s", ErrScopeBusy, scope.Name)
	}
	defer lease.Close()
	documents, snapshot, diagnostics, err := readSnapshot(ctx, scope)
	if err != nil {
		return BuildResult{}, err
	}
	if current, ok, err := currentBuild(scopeDir, snapshot.ID); err != nil {
		return BuildResult{}, err
	} else if ok {
		return current, nil
	}
	changes := make([]analyzer.DocumentChange, 0, len(documents))
	digests := map[string]string{}
	for _, document := range documents {
		changes = append(changes, analyzer.DocumentChange{Kind: analyzer.Upsert, Path: document.Path, Language: document.Language, Content: document.Content, ContentDigest: document.ContentDigest})
		digests[document.Path] = document.ContentDigest
	}
	capabilities := []analyzer.Capability{analyzer.Parse, analyzer.Structure, analyzer.Imports, analyzer.Exports, analyzer.Definitions, analyzer.References, analyzer.Calls, analyzer.Injections, analyzer.SemanticResolution}
	analysis, err := provider.Analyze(ctx, analyzer.AnalysisRequest{Snapshot: analyzer.SnapshotID(snapshot.ID), Mode: analyzer.Reset, Documents: changes, Capabilities: capabilities})
	if err != nil {
		return BuildResult{}, err
	}
	if err := analyzer.ValidateResult(analyzer.AnalysisRequest{Snapshot: analyzer.SnapshotID(snapshot.ID), Mode: analyzer.Reset, Documents: changes, Capabilities: capabilities}, analysis, digests); err != nil {
		return BuildResult{}, fmt.Errorf("invalid analyzer result: %w", err)
	}
	manifest, err := normalizedManifest(scope, snapshot, analysis, diagnostics)
	if err != nil {
		return BuildResult{}, err
	}
	previous, _ := readCurrent(scopeDir)
	if err := publish(scopeDir, &manifest); err != nil {
		return BuildResult{}, err
	}
	status := "built"
	if previous == manifest.Generation {
		status = "current"
	}
	return BuildResult{Scope: scope.Name, Generation: manifest.Generation, Status: status, Documents: len(manifest.Documents), Symbols: len(manifest.Symbols), Relations: len(manifest.Relations), Diagnostics: len(manifest.Diagnostics)}, nil
}

func currentBuild(scopeDir, snapshotID string) (BuildResult, bool, error) {
	generation, err := readCurrent(scopeDir)
	if errors.Is(err, os.ErrNotExist) {
		return BuildResult{}, false, nil
	}
	if err != nil {
		return BuildResult{}, false, err
	}
	data, err := os.ReadFile(filepath.Join(scopeDir, "generations", generation, "generation.json"))
	if err != nil {
		return BuildResult{}, false, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil || !validGeneration(manifest, generation) {
		return BuildResult{}, false, errors.New("current code-index generation is corrupt")
	}
	if manifest.Snapshot.ID != snapshotID {
		return BuildResult{}, false, nil
	}
	return BuildResult{Scope: manifest.Snapshot.Scope, Generation: generation, Status: "current", Documents: len(manifest.Documents), Symbols: len(manifest.Symbols), Relations: len(manifest.Relations), Diagnostics: len(manifest.Diagnostics)}, true, nil
}

func DisposeGeneration(workspace, scopeName, generation string) (bool, error) {
	if len(generation) != 64 || strings.Trim(generation, "0123456789abcdef") != "" {
		return false, errors.New("generation must be an exact code-index generation ID")
	}
	scope, err := ResolveScope(workspace, scopeName)
	if err != nil {
		return false, err
	}
	root, err := CacheRoot()
	if err != nil {
		return false, err
	}
	scopeDir := filepath.Join(root, safeName(scope.Name))
	lease, err := acquireLease(filepath.Join(scopeDir, "writer.lock"))
	if err != nil {
		return false, fmt.Errorf("%w: %s", ErrScopeBusy, scope.Name)
	}
	defer lease.Close()
	current, err := readCurrent(scopeDir)
	if err != nil {
		return false, err
	}
	if current == generation {
		return false, errors.New("cannot dispose the current code-index generation")
	}
	target := filepath.Join(scopeDir, "generations", generation)
	if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := os.RemoveAll(target); err != nil {
		return false, err
	}
	return true, syncDir(filepath.Dir(target))
}

func readSnapshot(ctx context.Context, scope Scope) ([]SourceDocument, Snapshot, []Diagnostic, error) {
	repository, err := git(ctx, scope.Root, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, Snapshot{}, nil, fmt.Errorf("code scope %q is not a local Git repository: %w", scope.Name, err)
	}
	repository, err = filepath.EvalSymlinks(repository)
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	configured, err := filepath.EvalSymlinks(scope.Root)
	if err != nil || filepath.Clean(repository) != filepath.Clean(configured) {
		return nil, Snapshot{}, nil, fmt.Errorf("code scope %q root must be the canonical Git root", scope.Name)
	}
	commit, err := git(ctx, repository, "rev-parse", "HEAD")
	if err != nil {
		return nil, Snapshot{}, nil, fmt.Errorf("read HEAD: %w", err)
	}
	branch, _ := git(ctx, repository, "branch", "--show-current")
	tracked, err := gitBytes(ctx, repository, "ls-files", "-z")
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	paths := strings.Split(strings.TrimSuffix(string(tracked), "\x00"), "\x00")
	slices.Sort(paths)
	root, err := os.OpenRoot(repository)
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	defer root.Close()
	allowed := map[string]bool{}
	for _, language := range scope.Languages {
		allowed[strings.ToLower(language)] = true
	}
	documents := make([]SourceDocument, 0)
	diagnostics := make([]Diagnostic, 0)
	for _, path := range paths {
		if path == "" || excluded(path) {
			continue
		}
		if len(documents) >= scope.MaxFiles {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_bound", Severity: "warning", Message: "maximum source-file count reached"})
			break
		}
		file, err := root.Open(path)
		if err != nil {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_open", Severity: "warning", Message: bounded(err.Error(), 256)})
			continue
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_stat", Severity: "warning", Message: bounded(err.Error(), 256)})
			continue
		}
		if !info.Mode().IsRegular() {
			file.Close()
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "unsupported_source", Severity: "warning", Message: "tracked source is not a regular file"})
			continue
		}
		if info.Size() > scope.MaxFileBytes {
			file.Close()
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_bound", Severity: "warning", Message: "tracked source exceeds max_file_bytes"})
			continue
		}
		content, err := io.ReadAll(io.LimitReader(file, scope.MaxFileBytes+1))
		file.Close()
		if err != nil {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_read", Severity: "warning", Message: bounded(err.Error(), 256)})
			continue
		}
		if int64(len(content)) > scope.MaxFileBytes {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "source_bound", Severity: "warning", Message: "tracked source exceeds max_file_bytes"})
			continue
		}
		if strings.IndexByte(string(content), 0) >= 0 {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "unsupported_source", Severity: "warning", Message: "tracked source is binary"})
			continue
		}
		generatedPrefix := content
		if len(generatedPrefix) > 1024 {
			generatedPrefix = generatedPrefix[:1024]
		}
		if bytes.Contains(generatedPrefix, []byte("Code generated")) && bytes.Contains(generatedPrefix, []byte("DO NOT EDIT")) {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "unsupported_source", Severity: "warning", Message: "tracked generated source is excluded"})
			continue
		}
		firstLine := content
		if index := strings.IndexByte(string(firstLine), '\n'); index >= 0 {
			firstLine = firstLine[:index]
		}
		if len(firstLine) > 512 {
			firstLine = firstLine[:512]
		}
		language, err := languagepack.Detect(path, firstLine)
		if err != nil {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Category: "unsupported_source", Severity: "warning", Message: bounded(err.Error(), 256)})
			continue
		}
		if !allowed[language] {
			diagnostics = appendBounded(diagnostics, scope.MaxDiagnostics, Diagnostic{Path: path, Language: language, Category: "unsupported_source", Severity: "warning", Message: "detected language is not allowed by the code scope"})
			continue
		}
		documents = append(documents, SourceDocument{Path: filepath.ToSlash(path), Language: language, Content: content, ContentDigest: digest(content)})
	}
	sourceParts := make([]string, 0, len(documents))
	for _, document := range documents {
		sourceParts = append(sourceParts, document.Path+"\x00"+document.ContentDigest)
	}
	parserCache, err := languagepack.DefaultCacheDir()
	if err != nil {
		return nil, Snapshot{}, nil, err
	}
	parserDigest, err := languagepack.TrustDigest(parserCache, scope.Languages)
	if err != nil {
		return nil, Snapshot{}, nil, fmt.Errorf("verify parser trust: %w", err)
	}
	snapshot := Snapshot{
		Scope: scope.Name, Repository: repository, Commit: commit, Branch: branch,
		SourceDigest: digest([]byte(strings.Join(sourceParts, "\x00"))),
		ConfigurationDigest: digestJSON(struct {
			Name      string
			Root      string
			Languages []string
		}{scope.Name, repository, scope.Languages}),
		AnalyzerDigest: digest([]byte("tree-sitter-language-pack:" + languagepack.Release)),
		ParserDigest:   parserDigest,
		QueryDigest:    digest([]byte(languagepack.Release)), NormalizerDigest: digest([]byte(languagepack.NormalizerVersion)),
		SchemaDigest: digest([]byte(fmt.Sprint(SchemaVersion))),
		BoundsDigest: digestJSON([]any{scope.MaxFiles, scope.MaxFileBytes, scope.MaxRecords, scope.MaxDiagnostics, scope.MaxResults, scope.MaxTraversal}),
	}
	snapshot.ID = digestJSON(snapshot)
	return documents, snapshot, diagnostics, nil
}

func normalizedManifest(scope Scope, snapshot Snapshot, result analyzer.AnalysisResult, sourceDiagnostics []Diagnostic) (Manifest, error) {
	manifest := Manifest{Schema: SchemaVersion, Snapshot: snapshot, Provenance: result.Provenance, CreatedAt: time.Now().UTC(), Diagnostics: sourceDiagnostics}
	for index := range manifest.Diagnostics {
		diagnostic := &manifest.Diagnostics[index]
		diagnostic.ID = recordID(snapshot.ID, "diagnostic", diagnostic.Path, diagnostic.Language, diagnostic.Category, diagnostic.Severity, diagnostic.Message, fmt.Sprint(diagnostic.Span))
	}
	for _, analyzed := range result.Documents {
		document := Document{ID: recordID(snapshot.ID, "document", analyzed.Path, analyzed.Language, analyzed.ContentDigest, digestJSON(analyzed.Coverage)), Path: analyzed.Path, Language: analyzed.Language, ContentDigest: analyzed.ContentDigest, Coverage: analyzed.Coverage}
		manifest.Documents = append(manifest.Documents, document)
		for _, value := range analyzed.Symbols {
			manifest.Symbols = append(manifest.Symbols, Symbol{ID: recordID(snapshot.ID, "symbol", analyzed.Path, analyzed.Language, value.Kind, value.QualifiedName, value.Name, value.Signature, fmt.Sprint(value.Span)), DocumentID: document.ID, Path: analyzed.Path, Language: analyzed.Language, Kind: value.Kind, Name: value.Name, QualifiedName: value.QualifiedName, Signature: value.Signature, Span: value.Span})
		}
		for _, value := range analyzed.Relations {
			manifest.Relations = append(manifest.Relations, Relation{ID: recordID(snapshot.ID, "relation", analyzed.Path, value.Kind, value.Source, value.Target, fmt.Sprint(value.Span)), DocumentID: document.ID, Path: analyzed.Path, Kind: value.Kind, Source: value.Source, Target: value.Target, Candidates: value.Candidates, Evidence: value.Evidence, Resolution: value.Resolution, Span: value.Span})
		}
		for _, value := range analyzed.Diagnostics {
			manifest.Diagnostics = appendBounded(manifest.Diagnostics, scope.MaxDiagnostics, Diagnostic{ID: recordID(snapshot.ID, "diagnostic", analyzed.Path, analyzed.Language, value.Category, value.Severity, value.Message, fmt.Sprint(value.Span)), Path: analyzed.Path, Language: analyzed.Language, Category: value.Category, Severity: value.Severity, Message: value.Message, Span: value.Span})
		}
	}
	if len(manifest.Symbols)+len(manifest.Relations)+len(manifest.Diagnostics) > scope.MaxRecords {
		return Manifest{}, errors.New("code-index record bound exceeded")
	}
	resolveRelations(&manifest)
	slices.SortFunc(manifest.Documents, func(a, b Document) int { return strings.Compare(a.Path, b.Path) })
	slices.SortFunc(manifest.Symbols, func(a, b Symbol) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(manifest.Relations, func(a, b Relation) int { return strings.Compare(a.ID, b.ID) })
	slices.SortFunc(manifest.Diagnostics, func(a, b Diagnostic) int { return strings.Compare(a.ID, b.ID) })
	copy := manifest
	copy.CreatedAt = time.Time{}
	manifest.Generation = digestJSON(copy)
	return manifest, nil
}

func resolveRelations(manifest *Manifest) {
	byName := map[string][]string{}
	for _, symbol := range manifest.Symbols {
		byName[symbol.Name] = append(byName[symbol.Name], symbol.ID)
		if symbol.QualifiedName != "" && symbol.QualifiedName != symbol.Name {
			byName[symbol.QualifiedName] = append(byName[symbol.QualifiedName], symbol.ID)
		}
	}
	for index := range manifest.Relations {
		relation := &manifest.Relations[index]
		candidates := append([]string(nil), byName[relation.Target]...)
		slices.Sort(candidates)
		relation.Candidates = slices.Compact(candidates)
		switch len(relation.Candidates) {
		case 0:
			relation.Resolution = analyzer.Unresolved
		case 1:
			relation.Resolution = analyzer.Resolved
		default:
			relation.Resolution = analyzer.Ambiguous
		}
		relation.ID = recordID(manifest.Snapshot.ID, "relation", relation.Path, relation.Kind, relation.Source, relation.Target, strings.Join(relation.Candidates, "\x00"), string(relation.Evidence), string(relation.Resolution), fmt.Sprint(relation.Span))
	}
}

func publish(scopeDir string, manifest *Manifest) error {
	if err := writeGeneration(scopeDir, manifest); err != nil {
		return err
	}
	return writeCurrent(scopeDir, manifest.Generation)
}

func writeGeneration(scopeDir string, manifest *Manifest) error {
	generationDir := filepath.Join(scopeDir, "generations", manifest.Generation)
	if _, err := os.Stat(filepath.Join(generationDir, "generation.json")); err == nil {
		return nil
	}
	temporary, err := os.MkdirTemp(scopeDir, ".generation-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	var verified Manifest
	if err := json.Unmarshal(data, &verified); err != nil || !validGeneration(verified, manifest.Generation) {
		return errors.New("generated code-index manifest failed verification")
	}
	path := filepath.Join(temporary, "generation.json")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := syncDir(temporary); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(generationDir), 0o700); err != nil {
		return err
	}
	if err := os.Rename(temporary, generationDir); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	if err := syncDir(filepath.Dir(generationDir)); err != nil {
		return err
	}
	return nil
}

type liveSelection struct {
	WorkspaceEpoch    string    `json:"workspace_epoch"`
	VerifiedWatermark Watermark `json:"verified_watermark"`
}

type selector struct {
	Version      int            `json:"version"`
	GenerationID string         `json:"generation_id"`
	Live         *liveSelection `json:"live"`
}

func writeCurrent(scopeDir, generation string) error {
	return writeSelector(scopeDir, selector{Version: 2, GenerationID: generation})
}

func writeSelector(scopeDir string, selected selector) error {
	if selected.Version != 2 || !validGenerationID(selected.GenerationID) {
		return errors.New("invalid code-index selector")
	}
	temporary, err := os.CreateTemp(scopeDir, ".current-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	data, err := json.Marshal(selected)
	if err != nil {
		temporary.Close()
		return err
	}
	data = append(data, '\n')
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, filepath.Join(scopeDir, "current")); err != nil {
		return err
	}
	return syncDir(scopeDir)
}

func syncDir(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

type Reader struct {
	scope     Scope
	manifest  Manifest
	byID      map[string]Symbol
	freshness *LiveFreshness
	closed    bool
}

func Open(workspace, scopeName string) (*Reader, error) {
	scope, err := ResolveScope(workspace, scopeName)
	if err != nil {
		return nil, err
	}
	root, err := CacheRoot()
	if err != nil {
		return nil, err
	}
	scopeDir := filepath.Join(root, safeName(scope.Name))
	for attempts := 0; attempts < 2; attempts++ {
		generation, err := readCurrent(scopeDir)
		if err != nil {
			return nil, fmt.Errorf("code scope %q has no index; run `gnosis index code --scope %s`", scope.Name, scope.Name)
		}
		data, err := os.ReadFile(filepath.Join(scopeDir, "generations", generation, "generation.json"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil || !validGeneration(manifest, generation) {
			return nil, errors.New("code-index generation is corrupt")
		}
		reader := &Reader{scope: scope, manifest: manifest, byID: map[string]Symbol{}}
		for _, symbol := range manifest.Symbols {
			reader.byID[symbol.ID] = symbol
		}
		return reader, nil
	}
	return nil, errors.New("code-index generation changed during open")
}

func (reader *Reader) Close() error {
	if reader.closed {
		return errors.New("code-index reader is closed")
	}
	reader.closed = true
	reader.byID = nil
	return nil
}

func (reader *Reader) CheckCurrent(ctx context.Context) error {
	reader.requireOpen()
	_, current, _, err := readSnapshot(ctx, reader.scope)
	if err != nil {
		return err
	}
	if current.ID != reader.manifest.Snapshot.ID {
		return ErrNotCurrent
	}
	return nil
}

func (reader *Reader) Status() StatusResult {
	reader.requireOpen()
	manifest := reader.manifest
	return StatusResult{Scope: reader.scope.Name, Status: "current", Generation: manifest.Generation, Snapshot: &manifest.Snapshot, Provenance: &manifest.Provenance, Documents: len(manifest.Documents), Symbols: len(manifest.Symbols), Relations: len(manifest.Relations), Diagnostics: len(manifest.Diagnostics)}
}

func (reader *Reader) Search(query, language string, limit int) SearchResult {
	reader.requireOpen()
	limit = boundedLimit(limit, reader.scope.MaxResults)
	query = strings.ToLower(strings.TrimSpace(query))
	matches := make([]Symbol, 0)
	for _, symbol := range reader.manifest.Symbols {
		if language != "" && symbol.Language != language {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(symbol.Name), query) || strings.Contains(strings.ToLower(symbol.QualifiedName), query) {
			matches = append(matches, symbol)
		}
	}
	slices.SortFunc(matches, func(a, b Symbol) int {
		if comparison := strings.Compare(a.Name, b.Name); comparison != 0 {
			return comparison
		}
		return strings.Compare(a.ID, b.ID)
	})
	total := len(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return SearchResult{Scope: reader.scope.Name, Generation: reader.manifest.Generation, Snapshot: reader.manifest.Snapshot, Provenance: reader.manifest.Provenance, Total: total, Truncated: total > len(matches), Symbols: matches, Coverage: reader.coverage(), Freshness: reader.freshness}
}

func (reader *Reader) Symbol(id string) (Symbol, error) {
	reader.requireOpen()
	symbol, ok := reader.byID[id]
	if !ok {
		return Symbol{}, fmt.Errorf("code symbol %q was not found", id)
	}
	return symbol, nil
}

func (reader *Reader) ReadSymbol(id string) (SymbolResult, error) {
	symbol, err := reader.Symbol(id)
	if err != nil {
		return SymbolResult{}, err
	}
	return SymbolResult{Scope: reader.scope.Name, Generation: reader.manifest.Generation, Snapshot: reader.manifest.Snapshot, Provenance: reader.manifest.Provenance, Symbol: symbol, Coverage: reader.coverage(), Freshness: reader.freshness}, nil
}

func (reader *Reader) Diagnostics(path, language, category string, limit int) DiagnosticResult {
	reader.requireOpen()
	limit = boundedLimit(limit, reader.scope.MaxResults)
	matches := make([]Diagnostic, 0)
	for _, diagnostic := range reader.manifest.Diagnostics {
		if (path == "" || diagnostic.Path == path) && (language == "" || diagnostic.Language == language) && (category == "" || diagnostic.Category == category) {
			matches = append(matches, diagnostic)
		}
	}
	total := len(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return DiagnosticResult{Scope: reader.scope.Name, Generation: reader.manifest.Generation, Snapshot: reader.manifest.Snapshot, Provenance: reader.manifest.Provenance, Total: total, Truncated: total > len(matches), Diagnostics: matches, Coverage: reader.coverage(), Freshness: reader.freshness}
}

func (reader *Reader) Trace(id, direction string, limit int) (TraceResult, error) {
	reader.requireOpen()
	if direction != "incoming" && direction != "outgoing" {
		return TraceResult{}, errors.New("direction must be incoming or outgoing")
	}
	symbol, err := reader.Symbol(id)
	if err != nil {
		return TraceResult{}, err
	}
	limit = boundedLimit(limit, reader.scope.MaxTraversal)
	relations := make([]Relation, 0)
	for _, relation := range reader.manifest.Relations {
		matches := direction == "incoming" && (relation.Target == symbol.Name || relation.Target == symbol.QualifiedName)
		matches = matches || direction != "incoming" && (relation.Source == symbol.Name || relation.Source == symbol.QualifiedName || relation.DocumentID == symbol.DocumentID)
		if matches {
			relations = append(relations, relation)
		}
	}
	total := len(relations)
	if len(relations) > limit {
		relations = relations[:limit]
	}
	return reader.traceResult("relations", total, total > len(relations), relations, nil), nil
}

func (reader *Reader) Neighbors(id, direction string, limit int) (TraceResult, error) {
	reader.requireOpen()
	if direction != "incoming" && direction != "outgoing" {
		return TraceResult{}, errors.New("direction must be incoming or outgoing")
	}
	if _, err := reader.Symbol(id); err != nil {
		return TraceResult{}, err
	}
	limit = boundedLimit(limit, reader.scope.MaxTraversal)
	relations, symbols := reader.adjacent(id, direction)
	total := len(symbols)
	if len(symbols) > limit {
		symbols = symbols[:limit]
		relations = relations[:limit]
	}
	return reader.traceResult("neighbors", total, total > len(symbols), relations, symbols), nil
}

func (reader *Reader) Path(from, to, direction string, depth, limit int) (TraceResult, error) {
	reader.requireOpen()
	if direction != "incoming" && direction != "outgoing" {
		return TraceResult{}, errors.New("direction must be incoming or outgoing")
	}
	if _, err := reader.Symbol(from); err != nil {
		return TraceResult{}, err
	}
	if _, err := reader.Symbol(to); err != nil {
		return TraceResult{}, err
	}
	limit = boundedLimit(limit, reader.scope.MaxTraversal)
	if depth <= 0 || depth > reader.scope.MaxTraversal {
		depth = reader.scope.MaxTraversal
	}
	type step struct {
		id        string
		symbols   []Symbol
		relations []Relation
	}
	queue := []step{{id: from, symbols: []Symbol{reader.byID[from]}}}
	seen := map[string]bool{from: true}
	visited := 0
	for len(queue) > 0 && visited < limit {
		current := queue[0]
		queue = queue[1:]
		visited++
		if current.id == to {
			return reader.traceResult("path", len(current.relations), false, current.relations, current.symbols), nil
		}
		if len(current.relations) >= depth {
			continue
		}
		relations, neighbors := reader.adjacent(current.id, direction)
		for index, neighbor := range neighbors {
			if seen[neighbor.ID] {
				continue
			}
			seen[neighbor.ID] = true
			queue = append(queue, step{id: neighbor.ID, symbols: append(append([]Symbol(nil), current.symbols...), neighbor), relations: append(append([]Relation(nil), current.relations...), relations[index])})
		}
	}
	return reader.traceResult("path", 0, visited >= limit, nil, nil), nil
}

func (reader *Reader) adjacent(id, direction string) ([]Relation, []Symbol) {
	source := reader.byID[id]
	type pair struct {
		relation Relation
		symbol   Symbol
	}
	pairs := make([]pair, 0)
	for _, relation := range reader.manifest.Relations {
		for _, candidate := range reader.manifest.Symbols {
			outgoing := relationMatchesSource(relation, source) && relationMatchesTarget(relation, candidate)
			incoming := relationMatchesTarget(relation, source) && relationMatchesSource(relation, candidate)
			if (direction == "incoming" && incoming) || (direction != "incoming" && outgoing) {
				pairs = append(pairs, pair{relation: relation, symbol: candidate})
			}
		}
	}
	slices.SortFunc(pairs, func(a, b pair) int {
		if comparison := strings.Compare(a.symbol.ID, b.symbol.ID); comparison != 0 {
			return comparison
		}
		return strings.Compare(a.relation.ID, b.relation.ID)
	})
	relations := make([]Relation, 0, len(pairs))
	symbols := make([]Symbol, 0, len(pairs))
	seen := map[string]bool{}
	for _, pair := range pairs {
		if seen[pair.symbol.ID] {
			continue
		}
		seen[pair.symbol.ID] = true
		relations = append(relations, pair.relation)
		symbols = append(symbols, pair.symbol)
	}
	return relations, symbols
}

func relationMatchesSource(relation Relation, symbol Symbol) bool {
	return relation.Source == symbol.Name || relation.Source == symbol.QualifiedName || relation.DocumentID == symbol.DocumentID
}

func relationMatchesTarget(relation Relation, symbol Symbol) bool {
	if relation.Target == symbol.Name || relation.Target == symbol.QualifiedName {
		return true
	}
	return slices.Contains(relation.Candidates, symbol.Name) || slices.Contains(relation.Candidates, symbol.QualifiedName) || slices.Contains(relation.Candidates, symbol.ID)
}

func (reader *Reader) traceResult(mode string, total int, truncated bool, relations []Relation, symbols []Symbol) TraceResult {
	return TraceResult{Mode: mode, Scope: reader.scope.Name, Generation: reader.manifest.Generation, Snapshot: reader.manifest.Snapshot, Provenance: reader.manifest.Provenance, Total: total, Truncated: truncated, Relations: relations, Symbols: symbols, Coverage: reader.coverage(), Freshness: reader.freshness}
}

func (reader *Reader) requireOpen() {
	if reader == nil || reader.closed {
		panic("code-index reader is closed")
	}
}

func (reader *Reader) coverage() []analyzer.Coverage {
	levels := map[analyzer.Capability]analyzer.CoverageLevel{}
	for _, document := range reader.manifest.Documents {
		for _, coverage := range document.Coverage {
			current, exists := levels[coverage.Capability]
			if !exists || coverageRank(coverage.Level) < coverageRank(current) {
				levels[coverage.Capability] = coverage.Level
			}
		}
	}
	result := make([]analyzer.Coverage, 0, len(levels))
	for capability, level := range levels {
		result = append(result, analyzer.Coverage{Capability: capability, Level: level})
	}
	slices.SortFunc(result, func(a, b analyzer.Coverage) int { return strings.Compare(string(a.Capability), string(b.Capability)) })
	return result
}

func coverageRank(level analyzer.CoverageLevel) int {
	switch level {
	case analyzer.Complete:
		return 2
	case analyzer.Partial:
		return 1
	default:
		return 0
	}
}

func validGeneration(manifest Manifest, generation string) bool {
	if manifest.Generation != generation || manifest.Schema != SchemaVersion {
		return false
	}
	copy := manifest
	copy.Generation = ""
	copy.CreatedAt = time.Time{}
	return digestJSON(copy) == generation
}

func readCurrent(scopeDir string) (string, error) {
	selected, err := readSelector(scopeDir)
	return selected.GenerationID, err
}

func readSelector(scopeDir string) (selector, error) {
	data, err := os.ReadFile(filepath.Join(scopeDir, "current"))
	if err != nil {
		return selector{}, err
	}
	var selected selector
	if err := json.Unmarshal(data, &selected); err != nil {
		return selector{}, errors.New("invalid current selector")
	}
	if selected.Version != 2 {
		return selector{}, fmt.Errorf("unsupported current selector version %d", selected.Version)
	}
	if !validGenerationID(selected.GenerationID) || selected.Live != nil && (selected.Live.WorkspaceEpoch == "" || selected.Live.VerifiedWatermark == 0) {
		return selector{}, errors.New("invalid current selector")
	}
	return selected, nil
}

func validGenerationID(value string) bool {
	return len(value) == 64 && strings.Trim(value, "0123456789abcdef") == ""
}

func git(ctx context.Context, root string, arguments ...string) (string, error) {
	data, err := gitBytes(ctx, root, arguments...)
	return strings.TrimSpace(string(data)), err
}

func gitBytes(ctx context.Context, root string, arguments ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, arguments...)...)
	command.Stdin = nil
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(arguments, " "), err)
	}
	return data, nil
}

func excluded(path string) bool {
	path = "/" + filepath.ToSlash(path) + "/"
	return strings.Contains(path, "/vendor/") || strings.Contains(path, "/node_modules/") || strings.Contains(path, "/.git/") || strings.Contains(path, "/dist/") || strings.Contains(path, "/build/") || strings.HasSuffix(path, ".min.js/") || strings.Contains(path, "/generated/")
}

func appendBounded[T any](values []T, limit int, value T) []T {
	if len(values) < limit {
		return append(values, value)
	}
	return values
}

func boundedLimit(value, maximum int) int {
	if value <= 0 || value > maximum {
		return maximum
	}
	return value
}

func safeName(value string) string    { return digest([]byte(value))[:24] }
func recordID(parts ...string) string { return digest([]byte(strings.Join(parts, "\x00"))) }
func digestJSON(value any) string     { data, _ := json.Marshal(value); return digest(data) }
func digest(value []byte) string      { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
func bounded(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
