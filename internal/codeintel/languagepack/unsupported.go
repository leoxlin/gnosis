//go:build !linux || !amd64

package languagepack

import (
	"context"
	"errors"
	"runtime"

	"gnosis/internal/codeintel/analyzer"
)

const (
	Release           = "v1.13.7"
	ParserABI         = "14"
	ManifestVersion   = 1
	NormalizerVersion = "1"
)

type Manifest struct{}
type ParserStatus struct{}
type Analyzer struct{}

func unsupported() error {
	return errors.New("code intelligence is unsupported on " + runtime.GOOS + "/" + runtime.GOARCH)
}

func Platform() (string, error)        { return "", unsupported() }
func SupportedPlatforms() []string     { return []string{"linux/amd64"} }
func DefaultCacheDir() (string, error) { return "", unsupported() }
func Install(context.Context, string, []string) (Manifest, bool, error) {
	return Manifest{}, false, unsupported()
}
func Status(string, []string) ([]ParserStatus, error) { return nil, unsupported() }
func Catalog(string) ([]string, error)                { return nil, unsupported() }
func New(string, []string) (*Analyzer, error)         { return nil, unsupported() }
func Detect(string, []byte) (string, error)           { return "", unsupported() }
func (*Analyzer) Analyze(context.Context, analyzer.AnalysisRequest) (analyzer.AnalysisResult, error) {
	return analyzer.AnalysisResult{}, unsupported()
}
func (*Analyzer) Close() error { return nil }
