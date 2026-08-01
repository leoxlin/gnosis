//go:build linux && amd64

package languagepack

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gnosis/internal/codeintel/analyzer"
)

func TestDetectCanonicalLanguages(t *testing.T) {
	cases := map[string]string{"main.go": "go", "app.ts": "typescript", "app.js": "javascript"}
	for path, want := range cases {
		got, err := Detect(path, nil)
		if err != nil || got != want {
			t.Fatalf("Detect(%q) = %q, %v; want %q", path, got, err, want)
		}
	}
	if _, err := Detect("script", []byte("#!/usr/bin/env node")); err != nil {
		t.Fatal(err)
	}
	if _, err := Detect("main.go", []byte("#!/usr/bin/env node")); err == nil {
		t.Fatal("expected ambiguous detection error")
	}
	if got, err := Detect("script", []byte("#!/usr/bin/env node")); err != nil || got != "javascript" {
		t.Fatalf("shebang detection = %q, %v", got, err)
	}
	if _, err := Detect("main.go", []byte("#!/usr/bin/env node")); err == nil {
		t.Fatal("expected ambiguous path/shebang rejection")
	}
	if _, err := Detect("LICENSE", nil); err == nil {
		t.Fatal("expected unknown language rejection")
	}
}

func TestParserCacheSafetyLockingAndCorruptManifest(t *testing.T) {
	if err := safeCacheDir("relative"); err == nil {
		t.Fatal("expected relative cache rejection")
	}
	cache := t.TempDir()
	unlock, err := lock(context.Background(), cache)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := lock(ctx, cache); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("competing lock error = %v", err)
	}
	unlock()
	if err := os.WriteFile(filepath.Join(cache, "gnosis-manifest.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(cache, []string{"go"}); err == nil {
		t.Fatal("expected corrupt manifest rejection")
	}
}

func TestManifestRejectsChangedLibrary(t *testing.T) {
	cache := t.TempDir()
	library := filepath.Join(cache, "libtree-sitter-go.so")
	if err := os.WriteFile(library, []byte("parser"), 0o600); err != nil {
		t.Fatal(err)
	}
	installation := Installation{Language: "go", Library: filepath.Base(library), LibraryDigest: digestString("parser")}
	platform, _ := Platform()
	manifest := Manifest{Version: ManifestVersion, PackRelease: Release, Platform: platform, ABI: ParserABI, ReleaseManifestDigest: ReleaseManifestSHA256, BundleDigest: BundleSHA256, Installed: []Installation{installation}}
	if err := verifyManifest(manifest, cache); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(cache, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := TrustDigest(cache, []string{"go"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(library, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyManifest(manifest, cache); err == nil {
		t.Fatal("expected digest mismatch")
	}
	if _, err := TrustDigest(cache, []string{"go"}); err == nil {
		t.Fatal("expected trust digest failure")
	}
}

func TestMissingParserReportsUnsupportedWithoutNetwork(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	provider, err := New(filepath.Join(t.TempDir(), "absent"), []string{"go"})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	source := []byte("package main\n")
	result, err := provider.Analyze(context.Background(), analyzer.AnalysisRequest{
		Snapshot: "missing", Mode: analyzer.Reset,
		Documents:    []analyzer.DocumentChange{{Kind: analyzer.Upsert, Path: "main.go", Language: "go", Content: source, ContentDigest: digestBytes(source)}},
		Capabilities: []analyzer.Capability{analyzer.Parse},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != 1 || result.Documents[0].Coverage[0].Level != analyzer.Unsupported || len(result.Documents[0].Diagnostics) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestInstalledGoAndTypeScriptAnalyzeWithoutNetwork(t *testing.T) {
	if os.Getenv("GNOSIS_TEST_PARSER_INSTALL") == "" {
		t.Skip("set GNOSIS_TEST_PARSER_INSTALL=1 for the native parser spike")
	}
	cache := t.TempDir()
	manifest, changed, err := Install(context.Background(), cache, []string{"go", "typescript", "javascript"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || len(manifest.Installed) != 3 {
		t.Fatalf("install = changed %v, manifest %+v", changed, manifest)
	}
	if _, changed, err := Install(context.Background(), cache, []string{"go", "typescript", "javascript"}); err != nil || changed {
		t.Fatalf("identical reinstall = changed %v, err %v", changed, err)
	}
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	provider, err := New(cache, []string{"go", "typescript", "javascript"})
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []struct{ path, language, source string }{
		{"main.go", "go", "package main\nimport \"fmt\"\ntype Server struct{}\nfunc (Server) Run() { fmt.Println(\"x\") }\n"},
		{"app.ts", "typescript", "import { x } from './x'; export class App { run(): void {} }\n"},
		{"app.js", "javascript", "import x from './x.js'; export function run() {}\n"},
		{"broken.go", "go", "package broken\nfunc broken(\n"},
	}
	changes := make([]analyzer.DocumentChange, 0, len(fixtures))
	for _, fixture := range fixtures {
		sum := sha256.Sum256([]byte(fixture.source))
		changes = append(changes, analyzer.DocumentChange{Kind: analyzer.Upsert, Path: fixture.path, Language: fixture.language, Content: []byte(fixture.source), ContentDigest: hex.EncodeToString(sum[:])})
	}
	request := analyzer.AnalysisRequest{Snapshot: "offline", Mode: analyzer.Reset, Documents: changes, Capabilities: []analyzer.Capability{analyzer.Parse, analyzer.Structure, analyzer.Imports, analyzer.Definitions, analyzer.Calls}}
	result, err := provider.Analyze(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != len(fixtures) {
		t.Fatalf("documents = %d, want %d", len(result.Documents), len(fixtures))
	}
	for _, document := range result.Documents {
		if document.Path == "broken.go" {
			continue
		}
		if len(document.Symbols) == 0 {
			t.Fatalf("symbols missing for %+v", document)
		}
		for _, symbol := range document.Symbols {
			if symbol.Span.EndByte <= symbol.Span.StartByte {
				t.Fatalf("invalid symbol span: %+v", symbol)
			}
		}
	}
	broken := result.Documents[0]
	for _, document := range result.Documents {
		if document.Path == "broken.go" {
			broken = document
		}
	}
	if len(broken.Diagnostics) == 0 {
		t.Fatalf("partial syntax diagnostics = %+v", broken)
	}
	again, err := provider.Analyze(context.Background(), request)
	if err != nil || !reflect.DeepEqual(result, again) {
		t.Fatalf("repeated analysis differs: %v\nfirst: %+v\nsecond: %+v", err, result, again)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Analyze(canceled, request); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatal(err)
	}
	if err := provider.Close(); !errors.Is(err, analyzer.ErrClosed) {
		t.Fatalf("second close error = %v", err)
	}
}
