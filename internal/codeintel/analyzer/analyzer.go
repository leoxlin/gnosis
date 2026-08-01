package analyzer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

type SnapshotID string
type AnalysisMode string
type ChangeKind string
type Capability string
type CoverageLevel string
type EvidenceLevel string
type ResolutionState string

const (
	Reset AnalysisMode = "reset"
	Delta AnalysisMode = "delta"

	Upsert ChangeKind = "upsert"
	Delete ChangeKind = "delete"

	Parse              Capability = "parse"
	Structure          Capability = "structure"
	Imports            Capability = "imports"
	Exports            Capability = "exports"
	Definitions        Capability = "definitions"
	References         Capability = "references"
	Calls              Capability = "calls"
	Injections         Capability = "injections"
	SemanticResolution Capability = "semantic-resolution"

	Complete    CoverageLevel = "complete"
	Partial     CoverageLevel = "partial"
	Unsupported CoverageLevel = "unsupported"

	Syntactic EvidenceLevel = "syntactic"
	Semantic  EvidenceLevel = "semantic"

	Resolved   ResolutionState = "resolved"
	Ambiguous  ResolutionState = "ambiguous"
	Unresolved ResolutionState = "unresolved"
)

var ErrClosed = errors.New("analyzer is closed")

type Span struct {
	StartByte   int `json:"start_byte"`
	EndByte     int `json:"end_byte"`
	StartLine   int `json:"start_line"`
	StartColumn int `json:"start_column"`
	EndLine     int `json:"end_line"`
	EndColumn   int `json:"end_column"`
}

type DocumentChange struct {
	Kind          ChangeKind `json:"kind"`
	Path          string     `json:"path"`
	Language      string     `json:"language,omitempty"`
	Content       []byte     `json:"-"`
	ContentDigest string     `json:"content_digest,omitempty"`
}

type AnalysisRequest struct {
	Snapshot     SnapshotID       `json:"snapshot"`
	Mode         AnalysisMode     `json:"mode"`
	Documents    []DocumentChange `json:"documents"`
	Capabilities []Capability     `json:"capabilities"`
}

type Coverage struct {
	Capability Capability    `json:"capability"`
	Level      CoverageLevel `json:"level"`
}

type Symbol struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name,omitempty"`
	Signature     string `json:"signature,omitempty"`
	Container     string `json:"container,omitempty"`
	Excerpt       string `json:"excerpt,omitempty"`
	Span          Span   `json:"span"`
}

type Relation struct {
	Kind       string          `json:"kind"`
	Source     string          `json:"source"`
	Target     string          `json:"target,omitempty"`
	Candidates []string        `json:"candidates,omitempty"`
	Evidence   EvidenceLevel   `json:"evidence"`
	Resolution ResolutionState `json:"resolution"`
	Span       Span            `json:"span"`
}

type Diagnostic struct {
	Category string `json:"category"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Span     *Span  `json:"span,omitempty"`
	Usable   bool   `json:"usable"`
}

type DocumentAnalysis struct {
	Path          string       `json:"path"`
	Language      string       `json:"language"`
	ContentDigest string       `json:"content_digest"`
	Coverage      []Coverage   `json:"coverage"`
	Symbols       []Symbol     `json:"symbols,omitempty"`
	Relations     []Relation   `json:"relations,omitempty"`
	Diagnostics   []Diagnostic `json:"diagnostics,omitempty"`
}

type AnalyzerProvenance struct {
	Implementation        string `json:"implementation"`
	ImplementationVersion string `json:"implementation_version"`
	ParserRelease         string `json:"parser_release"`
	ParserDigest          string `json:"parser_digest"`
	ABI                   string `json:"abi"`
	QueryVersion          string `json:"query_version"`
	NormalizerVersion     string `json:"normalizer_version"`
}

type AnalysisResult struct {
	Snapshot   SnapshotID         `json:"snapshot"`
	Complete   bool               `json:"complete"`
	Documents  []DocumentAnalysis `json:"documents"`
	Provenance AnalyzerProvenance `json:"provenance"`
}

type Analyzer interface {
	Analyze(context.Context, AnalysisRequest) (AnalysisResult, error)
	Close() error
}

func (request AnalysisRequest) Validate() error {
	if request.Snapshot == "" {
		return errors.New("snapshot is required")
	}
	if request.Mode != Reset && request.Mode != Delta {
		return fmt.Errorf("invalid analysis mode %q", request.Mode)
	}
	paths := map[string]bool{}
	for _, document := range request.Documents {
		if !validPath(document.Path) {
			return fmt.Errorf("invalid document path %q", document.Path)
		}
		if paths[document.Path] {
			return fmt.Errorf("duplicate document path %q", document.Path)
		}
		paths[document.Path] = true
		if document.Kind != Upsert && document.Kind != Delete {
			return fmt.Errorf("invalid change kind %q", document.Kind)
		}
		if document.Kind == Upsert && (document.Language == "" || document.ContentDigest == "") {
			return fmt.Errorf("upsert %q requires language and content digest", document.Path)
		}
	}
	if request.Mode == Reset {
		for _, document := range request.Documents {
			if document.Kind != Upsert {
				return errors.New("reset requests may contain only upserts")
			}
		}
	}
	return nil
}

// ValidateResult enforces the provider-neutral authoritative-result contract.
// snapshotDigests contains every path and digest in the requested source snapshot,
// including documents not present in a delta request.
func ValidateResult(request AnalysisRequest, result AnalysisResult, snapshotDigests map[string]string) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if result.Snapshot != request.Snapshot {
		return fmt.Errorf("result snapshot %q does not match request %q", result.Snapshot, request.Snapshot)
	}
	provenance := result.Provenance
	if provenance.Implementation == "" || provenance.ImplementationVersion == "" || provenance.ParserRelease == "" || provenance.ParserDigest == "" || provenance.ABI == "" || provenance.QueryVersion == "" || provenance.NormalizerVersion == "" {
		return errors.New("analyzer provenance is incomplete")
	}
	if request.Mode == Reset && !result.Complete {
		return errors.New("reset result must be complete")
	}
	requestedUpserts := map[string]bool{}
	requestedDeletes := map[string]bool{}
	for _, change := range request.Documents {
		if change.Kind == Upsert {
			requestedUpserts[change.Path] = true
		} else {
			requestedDeletes[change.Path] = true
		}
	}
	seen := map[string]bool{}
	for _, document := range result.Documents {
		if seen[document.Path] {
			return fmt.Errorf("duplicate result document %q", document.Path)
		}
		seen[document.Path] = true
		if requestedDeletes[document.Path] {
			return fmt.Errorf("delete %q must not have a document result", document.Path)
		}
		digest, ok := snapshotDigests[document.Path]
		if !ok {
			return fmt.Errorf("result document %q is not in the source snapshot", document.Path)
		}
		if digest != document.ContentDigest {
			return fmt.Errorf("result document %q content digest does not match source snapshot", document.Path)
		}
		if err := document.Validate(request.Capabilities); err != nil {
			return fmt.Errorf("document %q: %w", document.Path, err)
		}
	}
	if !result.Complete {
		for path := range requestedUpserts {
			if !seen[path] {
				return fmt.Errorf("delta result omitted requested upsert %q", path)
			}
		}
	} else {
		for path := range snapshotDigests {
			if !seen[path] {
				return fmt.Errorf("complete result omitted source document %q", path)
			}
		}
	}
	return nil
}

func (document DocumentAnalysis) Validate(requested []Capability) error {
	if !validPath(document.Path) || document.Language == "" || document.ContentDigest == "" {
		return errors.New("path, language, and content digest are required")
	}
	coverage := map[Capability]bool{}
	for _, item := range document.Coverage {
		if item.Capability == "" || (item.Level != Complete && item.Level != Partial && item.Level != Unsupported) {
			return fmt.Errorf("invalid coverage %+v", item)
		}
		if coverage[item.Capability] {
			return fmt.Errorf("duplicate coverage for %q", item.Capability)
		}
		coverage[item.Capability] = true
	}
	for _, capability := range requested {
		if !coverage[capability] {
			return fmt.Errorf("missing coverage for %q", capability)
		}
	}
	for _, symbol := range document.Symbols {
		if symbol.Kind == "" || symbol.Name == "" || !symbol.Span.Valid() {
			return fmt.Errorf("invalid symbol %+v", symbol)
		}
	}
	for _, relation := range document.Relations {
		if relation.Kind == "" || relation.Source == "" || !relation.Span.Valid() ||
			(relation.Evidence != Syntactic && relation.Evidence != Semantic) ||
			(relation.Resolution != Resolved && relation.Resolution != Ambiguous && relation.Resolution != Unresolved) {
			return fmt.Errorf("invalid relation %+v", relation)
		}
	}
	return nil
}

func (span Span) Valid() bool {
	return span.StartByte >= 0 && span.EndByte >= span.StartByte && span.StartLine >= 0 &&
		span.EndLine >= span.StartLine && span.StartColumn >= 0 && span.EndColumn >= 0
}

func Canonicalize(result *AnalysisResult) {
	slices.SortFunc(result.Documents, func(a, b DocumentAnalysis) int { return strings.Compare(a.Path, b.Path) })
	for i := range result.Documents {
		document := &result.Documents[i]
		slices.SortFunc(document.Coverage, func(a, b Coverage) int { return strings.Compare(string(a.Capability), string(b.Capability)) })
		slices.SortFunc(document.Symbols, func(a, b Symbol) int {
			if comparison := a.Span.StartByte - b.Span.StartByte; comparison != 0 {
				return comparison
			}
			return strings.Compare(a.Name, b.Name)
		})
		slices.SortFunc(document.Relations, func(a, b Relation) int {
			if comparison := a.Span.StartByte - b.Span.StartByte; comparison != 0 {
				return comparison
			}
			return strings.Compare(a.Kind+a.Source+a.Target, b.Kind+b.Source+b.Target)
		})
	}
}

func validPath(path string) bool {
	return path != "" && !strings.HasPrefix(path, "/") && path != "." && !strings.HasPrefix(path, "../") && !strings.Contains(path, "/../") && !strings.ContainsRune(path, '\x00')
}
