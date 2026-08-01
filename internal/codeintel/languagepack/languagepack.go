//go:build linux && amd64

package languagepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tspack "github.com/xberg-io/tree-sitter-language-pack/packages/go"
	"gnosis/internal/codeintel/analyzer"
)

const (
	Release               = "v1.13.7"
	ParserABI             = "14"
	ManifestVersion       = 1
	NormalizerVersion     = "1"
	ReleaseManifestSHA256 = "36aa76f647597b8498f1c1630ab6a31771ef694f6412e4e2a669502c62f4adcb"
	BundleSHA256          = "039075e27f54d2369ae8b295d52d1815d545eccd3e03603e6aec24a2e59f662d"
)

var supportedPlatforms = map[string]bool{
	"linux/amd64": true,
}

type Manifest struct {
	Version               int            `json:"version"`
	PackRelease           string         `json:"pack_release"`
	Platform              string         `json:"platform"`
	ABI                   string         `json:"abi"`
	ReleaseManifestDigest string         `json:"release_manifest_digest"`
	BundleDigest          string         `json:"bundle_digest"`
	Installed             []Installation `json:"installed"`
}

type Installation struct {
	Language      string `json:"language"`
	Library       string `json:"library"`
	LibraryDigest string `json:"library_digest"`
}

type ParserStatus struct {
	Language      string `json:"language"`
	Installed     bool   `json:"installed"`
	PackRelease   string `json:"pack_release"`
	Platform      string `json:"platform"`
	ABI           string `json:"abi"`
	LibraryDigest string `json:"library_digest,omitempty"`
}

func Platform() (string, error) {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if !supportedPlatforms[platform] {
		return "", fmt.Errorf("code intelligence is unsupported on %s", platform)
	}
	return platform, nil
}

func SupportedPlatforms() []string {
	platforms := make([]string, 0, len(supportedPlatforms))
	for platform := range supportedPlatforms {
		platforms = append(platforms, platform)
	}
	slices.Sort(platforms)
	return platforms
}

func DefaultCacheDir() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "gnosis", "parsers", strings.TrimPrefix(Release, "v")), nil
}

func Install(ctx context.Context, cacheDir string, languages []string) (Manifest, bool, error) {
	if _, err := Platform(); err != nil {
		return Manifest{}, false, err
	}
	languages, err := canonicalLanguages(languages)
	if err != nil {
		return Manifest{}, false, err
	}
	if len(languages) == 0 {
		return Manifest{}, false, errors.New("at least one language is required")
	}
	if err := safeCacheDir(cacheDir); err != nil {
		return Manifest{}, false, err
	}
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return Manifest{}, false, err
	}
	release, err := readManifest(cacheDir)
	if err == nil && releaseMatches(release, cacheDir, languages) {
		return release, false, nil
	}
	requested := append([]string(nil), languages...)
	if err == nil && verifyManifest(release, cacheDir) == nil {
		for _, installed := range release.Installed {
			requested = append(requested, installed.Language)
		}
		requested, err = canonicalLanguages(requested)
		if err != nil {
			return Manifest{}, false, err
		}
	}
	unlock, err := lock(ctx, cacheDir)
	if err != nil {
		return Manifest{}, false, err
	}
	defer unlock()

	if err := tspack.Configure(tspack.PackConfig{CacheDir: &cacheDir}); err != nil {
		return Manifest{}, false, fmt.Errorf("configure parser cache: %w", err)
	}
	if _, err := tspack.DownloadAll(); err != nil {
		return Manifest{}, false, fmt.Errorf("install requested parsers: %w", err)
	}
	installed := make([]Installation, 0, len(requested))
	keep := map[string]bool{}
	for _, language := range requested {
		library, err := findLibrary(cacheDir, language)
		if err != nil {
			return Manifest{}, false, err
		}
		digest, err := digestFile(library)
		if err != nil {
			return Manifest{}, false, err
		}
		relative, err := filepath.Rel(cacheDir, library)
		if err != nil || !filepath.IsLocal(relative) {
			return Manifest{}, false, fmt.Errorf("parser library for %q escaped the cache", language)
		}
		keep[filepath.Clean(library)] = true
		installed = append(installed, Installation{Language: language, Library: filepath.ToSlash(relative), LibraryDigest: digest})
	}
	if err := removeUnrequestedLibraries(cacheDir, keep); err != nil {
		return Manifest{}, false, err
	}
	platform, _ := Platform()
	release = Manifest{
		Version: ManifestVersion, PackRelease: Release, Platform: platform, ABI: ParserABI,
		ReleaseManifestDigest: ReleaseManifestSHA256,
		BundleDigest:          BundleSHA256,
		Installed:             installed,
	}
	if err := writeManifest(cacheDir, release); err != nil {
		return Manifest{}, false, err
	}
	return release, true, nil
}

func Status(cacheDir string, languages []string) ([]ParserStatus, error) {
	languages, err := canonicalLanguages(languages)
	if err != nil {
		return nil, err
	}
	manifest, err := readManifest(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		return missingStatuses(languages), nil
	}
	if err != nil {
		return nil, err
	}
	if err := verifyManifest(manifest, cacheDir); err != nil {
		return nil, err
	}
	byLanguage := map[string]Installation{}
	for _, installed := range manifest.Installed {
		byLanguage[installed.Language] = installed
	}
	if len(languages) == 0 {
		for language := range byLanguage {
			languages = append(languages, language)
		}
		slices.Sort(languages)
	}
	statuses := make([]ParserStatus, 0, len(languages))
	for _, language := range languages {
		installed, ok := byLanguage[language]
		statuses = append(statuses, ParserStatus{
			Language: language, Installed: ok, PackRelease: manifest.PackRelease,
			Platform: manifest.Platform, ABI: manifest.ABI, LibraryDigest: installed.LibraryDigest,
		})
	}
	return statuses, nil
}

func TrustDigest(cacheDir string, languages []string) (string, error) {
	languages, err := canonicalLanguages(languages)
	if err != nil {
		return "", err
	}
	manifest, err := readManifest(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		return digestBytes([]byte(Release + "\x00" + ParserABI + "\x00" + BundleSHA256)), nil
	}
	if err != nil {
		return "", err
	}
	if err := verifyManifest(manifest, cacheDir); err != nil {
		return "", err
	}
	allowed := languageSet(languages)
	selected := make([]Installation, 0, len(languages))
	for _, installed := range manifest.Installed {
		if allowed[installed.Language] {
			selected = append(selected, installed)
		}
	}
	slices.SortFunc(selected, func(a, b Installation) int { return strings.Compare(a.Language, b.Language) })
	data, _ := json.Marshal(struct {
		Release       string
		ABI           string
		Manifest      string
		Bundle        string
		Installations []Installation
	}{Release, ParserABI, ReleaseManifestSHA256, BundleSHA256, selected})
	return digestBytes(data), nil
}

func Catalog(cacheDir string) ([]string, error) {
	if err := tspack.Configure(tspack.PackConfig{CacheDir: &cacheDir}); err != nil {
		return nil, err
	}
	registry := tspack.LanguageRegistryNew()
	defer registry.Free()
	languages := registry.AvailableLanguages()
	slices.Sort(languages)
	return slices.Compact(languages), nil
}

type Analyzer struct {
	mu        sync.Mutex
	manifest  Manifest
	cacheDir  string
	allow     map[string]bool
	parsers   map[string]*tree_sitter.Parser
	languages []nativeLanguage
	closed    bool
}

func New(cacheDir string, languages []string) (*Analyzer, error) {
	languages, err := canonicalLanguages(languages)
	if err != nil {
		return nil, err
	}
	manifest, err := readManifest(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		platform, _ := Platform()
		manifest = Manifest{Version: ManifestVersion, PackRelease: Release, Platform: platform, ABI: ParserABI, ReleaseManifestDigest: ReleaseManifestSHA256, BundleDigest: BundleSHA256}
	} else if err != nil {
		return nil, fmt.Errorf("read parser manifest: %w", err)
	} else if err := verifyManifest(manifest, cacheDir); err != nil {
		return nil, err
	}
	installed := map[string]bool{}
	for _, parser := range manifest.Installed {
		installed[parser.Language] = true
	}
	adapter := &Analyzer{manifest: manifest, cacheDir: cacheDir, allow: languageSet(languages), parsers: map[string]*tree_sitter.Parser{}}
	byLanguage := map[string]Installation{}
	for _, parser := range manifest.Installed {
		byLanguage[parser.Language] = parser
	}
	for _, name := range languages {
		if !installed[name] {
			continue
		}
		installed := byLanguage[name]
		loaded, err := openNativeLanguage(filepath.Join(cacheDir, filepath.FromSlash(installed.Library)), name)
		if err != nil {
			adapter.Close()
			return nil, err
		}
		parser := tree_sitter.NewParser()
		if err := parser.SetLanguage(loaded.language); err != nil {
			loaded.close()
			adapter.Close()
			return nil, fmt.Errorf("parser ABI mismatch for %q: %w", name, err)
		}
		adapter.languages = append(adapter.languages, loaded)
		adapter.parsers[name] = parser
	}
	return adapter, nil
}

func (adapter *Analyzer) Analyze(ctx context.Context, request analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return analyzer.AnalysisResult{}, analyzer.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return analyzer.AnalysisResult{}, err
	}
	if err := request.Validate(); err != nil {
		return analyzer.AnalysisResult{}, err
	}
	result := analyzer.AnalysisResult{
		Snapshot: request.Snapshot, Complete: request.Mode == analyzer.Reset,
		Provenance: analyzer.AnalyzerProvenance{
			Implementation: "tree-sitter-language-pack", ImplementationVersion: Release,
			ParserRelease: Release, ParserDigest: adapter.manifest.BundleDigest, ABI: ParserABI,
			QueryVersion: Release, NormalizerVersion: NormalizerVersion,
		},
	}
	for _, change := range request.Documents {
		if change.Kind == analyzer.Delete {
			continue
		}
		if !adapter.allow[change.Language] {
			result.Documents = append(result.Documents, unsupportedDocument(change, request.Capabilities, "language is not allowed by the code scope"))
			continue
		}
		if err := ctx.Err(); err != nil {
			return analyzer.AnalysisResult{}, err
		}
		parser := adapter.parsers[change.Language]
		if parser == nil {
			result.Documents = append(result.Documents, unsupportedDocument(change, request.Capabilities, "parser is not installed"))
			continue
		}
		tree := parser.ParseCtx(ctx, change.Content, nil)
		if tree == nil {
			return analyzer.AnalysisResult{}, fmt.Errorf("analyze %s: parsing returned no tree", change.Path)
		}
		result.Documents = append(result.Documents, normalizeTree(change, request.Capabilities, tree.RootNode(), change.Content))
		tree.Close()
	}
	analyzer.Canonicalize(&result)
	return result, nil
}

func (adapter *Analyzer) Close() error {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	if adapter.closed {
		return analyzer.ErrClosed
	}
	adapter.closed = true
	for _, parser := range adapter.parsers {
		parser.Close()
	}
	for _, language := range adapter.languages {
		language.close()
	}
	return nil
}

func normalizeTree(change analyzer.DocumentChange, requested []analyzer.Capability, root *tree_sitter.Node, source []byte) analyzer.DocumentAnalysis {
	document := analyzer.DocumentAnalysis{Path: change.Path, Language: change.Language, ContentDigest: change.ContentDigest}
	for _, capability := range requested {
		level := analyzer.Partial
		if capability == analyzer.Parse {
			level = analyzer.Complete
		} else if capability == analyzer.SemanticResolution {
			level = analyzer.Unsupported
		}
		document.Coverage = append(document.Coverage, analyzer.Coverage{Capability: capability, Level: level})
	}
	var visit func(*tree_sitter.Node)
	visit = func(node *tree_sitter.Node) {
		kind := node.Kind()
		if node.IsError() || node.IsMissing() {
			document.Diagnostics = append(document.Diagnostics, analyzer.Diagnostic{Category: "syntax", Severity: "error", Message: "syntax error", Span: treeSpan(node), Usable: true})
		}
		if isSymbolKind(kind) {
			if name := node.ChildByFieldName("name"); name != nil {
				document.Symbols = append(document.Symbols, analyzer.Symbol{Kind: kind, Name: bounded(name.Utf8Text(source), 256), Span: *treeSpan(node)})
			}
		}
		if strings.Contains(kind, "import") {
			document.Relations = append(document.Relations, analyzer.Relation{Kind: "import", Source: change.Path, Target: bounded(node.Utf8Text(source), 512), Evidence: analyzer.Syntactic, Resolution: analyzer.Unresolved, Span: *treeSpan(node)})
		}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(root)
	if root.HasError() {
		if len(document.Diagnostics) == 0 {
			document.Diagnostics = append(document.Diagnostics, analyzer.Diagnostic{Category: "syntax", Severity: "error", Message: "syntax tree contains errors", Span: treeSpan(root), Usable: true})
		}
		for index := range document.Coverage {
			if document.Coverage[index].Level == analyzer.Complete {
				document.Coverage[index].Level = analyzer.Partial
			}
		}
	}
	return document
}

func isSymbolKind(kind string) bool {
	return strings.Contains(kind, "declaration") || strings.Contains(kind, "definition") || strings.Contains(kind, "method") || strings.Contains(kind, "type_spec")
}

func treeSpan(node *tree_sitter.Node) *analyzer.Span {
	start, end := node.StartPosition(), node.EndPosition()
	value := analyzer.Span{StartByte: int(node.StartByte()), EndByte: int(node.EndByte()), StartLine: int(start.Row), StartColumn: int(start.Column), EndLine: int(end.Row), EndColumn: int(end.Column)}
	return &value
}

func Detect(path string, firstLine []byte) (string, error) {
	pathLanguage := tspack.DetectLanguageFromPath(filepath.ToSlash(path))
	contentLanguage := tspack.DetectLanguageFromContent(string(firstLine))
	if pathLanguage != nil && contentLanguage != nil && *pathLanguage != *contentLanguage {
		return "", fmt.Errorf("ambiguous language detection for %q", path)
	}
	if pathLanguage != nil {
		return canonicalLanguage(*pathLanguage)
	}
	if contentLanguage != nil {
		return canonicalLanguage(*contentLanguage)
	}
	return "", fmt.Errorf("language is not recognized for %q", path)
}

func unsupportedDocument(change analyzer.DocumentChange, capabilities []analyzer.Capability, message string) analyzer.DocumentAnalysis {
	document := analyzer.DocumentAnalysis{Path: change.Path, Language: change.Language, ContentDigest: change.ContentDigest}
	for _, capability := range capabilities {
		document.Coverage = append(document.Coverage, analyzer.Coverage{Capability: capability, Level: analyzer.Unsupported})
	}
	document.Diagnostics = []analyzer.Diagnostic{{Category: "unsupported", Severity: "warning", Message: message, Usable: true}}
	return document
}

func canonicalLanguages(languages []string) ([]string, error) {
	result := make([]string, 0, len(languages))
	for _, language := range languages {
		canonical, err := canonicalLanguage(language)
		if err != nil {
			return nil, err
		}
		result = append(result, canonical)
	}
	slices.Sort(result)
	return slices.Compact(result), nil
}

func canonicalLanguage(language string) (string, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	switch language {
	case "ts", "tsx", "typescriptreact":
		language = "typescript"
	case "js", "jsx", "node":
		language = "javascript"
	case "golang":
		language = "go"
	}
	if language == "" || strings.ContainsAny(language, "/\\.\x00") {
		return "", fmt.Errorf("invalid language %q", language)
	}
	return language, nil
}

func verifyManifest(manifest Manifest, cacheDir string) error {
	platform, err := Platform()
	if err != nil {
		return err
	}
	if manifest.Version != ManifestVersion || manifest.PackRelease != Release || manifest.Platform != platform || manifest.ABI != ParserABI {
		return errors.New("parser manifest is incompatible; reinstall the requested parsers")
	}
	if manifest.ReleaseManifestDigest != ReleaseManifestSHA256 || manifest.BundleDigest != BundleSHA256 {
		return errors.New("parser manifest digest mismatch; reinstall the requested parsers")
	}
	for _, installed := range manifest.Installed {
		library, err := filepath.Localize(installed.Library)
		if _, languageErr := canonicalLanguage(installed.Language); languageErr != nil || err != nil || !filepath.IsLocal(library) {
			return errors.New("parser manifest contains an unsafe entry")
		}
		digest, err := digestFile(filepath.Join(cacheDir, library))
		if err != nil || digest != installed.LibraryDigest {
			return fmt.Errorf("parser library %q digest mismatch; reinstall it", installed.Language)
		}
	}
	return nil
}

func readManifest(cacheDir string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, "gnosis-manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse parser manifest: %w", err)
	}
	return manifest, nil
}

func writeManifest(cacheDir string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(cacheDir, ".manifest-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
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
	return os.Rename(temporaryPath, filepath.Join(cacheDir, "gnosis-manifest.json"))
}

func lock(ctx context.Context, cacheDir string) (func(), error) {
	path := filepath.Join(cacheDir, ".gnosis-install.lock")
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			file.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func findLibrary(cacheDir, language string) (string, error) {
	needle := strings.ReplaceAll(language, "-", "_")
	var found string
	entries := []string{}
	err := filepath.WalkDir(cacheDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := strings.ToLower(entry.Name())
		if !entry.IsDir() && len(entries) < 20 {
			entries = append(entries, filepath.ToSlash(path))
		}
		base := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(name, ".dylib"), ".dll"), ".so")
		if !entry.IsDir() && (base == "libtree_sitter_"+needle || base == "tree_sitter_"+needle) && isLibrary(name) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found != "" {
		return found, nil
	}
	return "", fmt.Errorf("installed parser library for %q was not found in %v", language, entries)
}

func removeUnrequestedLibraries(cacheDir string, keep map[string]bool) error {
	return filepath.WalkDir(cacheDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && isLibrary(strings.ToLower(entry.Name())) && !keep[filepath.Clean(path)] {
			return os.Remove(path)
		}
		return nil
	})
}

func isLibrary(name string) bool {
	return strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".dylib") || strings.HasSuffix(name, ".dll")
}

func digestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func digestString(value string) string { return digestBytes([]byte(value)) }
func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func safeCacheDir(cacheDir string) error {
	if cacheDir == "" || !filepath.IsAbs(cacheDir) || filepath.Clean(cacheDir) != cacheDir || cacheDir == string(filepath.Separator) {
		return errors.New("parser cache must be a canonical absolute directory")
	}
	return nil
}

func releaseMatches(manifest Manifest, cacheDir string, languages []string) bool {
	if verifyManifest(manifest, cacheDir) != nil {
		return false
	}
	installed := map[string]bool{}
	for _, parser := range manifest.Installed {
		installed[parser.Language] = true
	}
	for _, language := range languages {
		if !installed[language] {
			return false
		}
	}
	return true
}

func missingStatuses(languages []string) []ParserStatus {
	statuses := make([]ParserStatus, 0, len(languages))
	for _, language := range languages {
		statuses = append(statuses, ParserStatus{Language: language})
	}
	return statuses
}

func languageSet(languages []string) map[string]bool {
	result := map[string]bool{}
	for _, language := range languages {
		result[language] = true
	}
	return result
}

func bounded(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
