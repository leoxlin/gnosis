package vault

import (
	"bytes"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const maxSupersessionDepth = 32

// TrustProjection is the shared agent-facing evidence and lifecycle envelope.
type TrustProjection struct {
	Origin         Origin                  `json:"origin"`
	Revision       string                  `json:"revision"`
	Status         string                  `json:"status,omitempty"`
	Confidence     *float64                `json:"confidence,omitempty"`
	Source         string                  `json:"source,omitempty"`
	ValidFrom      string                  `json:"valid_from,omitempty"`
	ValidUntil     string                  `json:"valid_until,omitempty"`
	ObservedAt     string                  `json:"observed_at,omitempty"`
	OccurredAt     string                  `json:"occurred_at,omitempty"`
	Tier           string                  `json:"tier,omitempty"`
	SupersededBy   *Supersession           `json:"superseded_by,omitempty"`
	Current        *bool                   `json:"current,omitempty"`
	Claims         []ClaimSignal           `json:"claims,omitempty"`
	Contradictions []Contradiction         `json:"contradictions,omitempty"`
	Maintenance    []MaintenanceAnnotation `json:"maintenance,omitempty"`
}

// MaintenanceAnnotation is one authored maintenance judgment.
type MaintenanceAnnotation struct {
	Kind       string             `json:"kind"`
	Reason     string             `json:"reason"`
	ObservedAt string             `json:"observed_at"`
	Author     string             `json:"author,omitempty"`
	Target     *MaintenanceTarget `json:"target,omitempty"`
}

// MaintenanceTarget preserves an authored duplicate target and its effective identity.
type MaintenanceTarget struct {
	Authored string `json:"authored"`
	URI      string `json:"uri,omitempty"`
}

// Supersession preserves the authored target and its effective identity.
type Supersession struct {
	Authored string `json:"authored"`
	URI      string `json:"uri,omitempty"`
}

// ClaimSignal locates one authored inference or ambiguity marker in the body.
type ClaimSignal struct {
	Kind   string `json:"kind"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Contradiction identifies one explicitly related contradictory page.
type Contradiction struct {
	URI      string `json:"uri"`
	Relation string `json:"relation"`
}

// CurrentResolutionStatus explains bounded supersession resolution.
type CurrentResolutionStatus string

const (
	CurrentResolved      CurrentResolutionStatus = "current"
	CurrentMissingTarget CurrentResolutionStatus = "missing_target"
	CurrentCycle         CurrentResolutionStatus = "cycle"
	CurrentDepthExceeded CurrentResolutionStatus = "depth_exceeded"
)

// CurrentResolution retains the requested identity and every followed successor.
type CurrentResolution struct {
	Status  CurrentResolutionStatus `json:"status"`
	Current string                  `json:"current,omitempty"`
	Chain   []string                `json:"chain"`
}

func initialTrust(document Document) TrustProjection {
	trust := TrustProjection{
		Origin:      document.Origin,
		Revision:    document.Revision,
		Status:      metadataScalar(document.Metadata, "status"),
		Source:      metadataScalar(document.Metadata, "source"),
		ValidFrom:   metadataTime(document.Metadata, "valid_from"),
		ValidUntil:  metadataTime(document.Metadata, "valid_until"),
		ObservedAt:  metadataTime(document.Metadata, "observed_at"),
		OccurredAt:  metadataTime(document.Metadata, "occurred_at"),
		Tier:        metadataScalar(document.Metadata, "tier"),
		Claims:      claimSignals(document.Body),
		Maintenance: copyMaintenance(document.Maintenance),
	}
	if confidence, ok := metadataNumber(document.Metadata, "confidence"); ok {
		trust.Confidence = &confidence
	}
	return trust
}

func copyMaintenance(source []MaintenanceAnnotation) []MaintenanceAnnotation {
	result := make([]MaintenanceAnnotation, len(source))
	for index, annotation := range source {
		result[index] = annotation
		if annotation.Target != nil {
			target := *annotation.Target
			result[index].Target = &target
		}
	}
	return result
}

func metadataScalar(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func metadataNumber(metadata map[string]any, key string) (float64, bool) {
	var result float64
	switch value := metadata[key].(type) {
	case float64:
		result = value
	case float32:
		result = float64(value)
	case int:
		result = float64(value)
	case int64:
		result = float64(value)
	default:
		return 0, false
	}
	return result, !math.IsNaN(result) && !math.IsInf(result, 0)
}

func metadataTime(metadata map[string]any, key string) string {
	switch value := metadata[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case time.Time:
		if value.Hour() == 0 && value.Minute() == 0 && value.Second() == 0 && value.Nanosecond() == 0 {
			return value.Format(time.DateOnly)
		}
		return value.Format(time.RFC3339Nano)
	default:
		return ""
	}
}

func projectTrust(pages []*effectivePage) error {
	resolver := newDocumentResolver(pages)
	for _, page := range pages {
		trust := initialTrust(page.document)
		raw := metadataScalar(page.document.Metadata, "superseded_by")
		if raw != "" {
			resolution, include, err := resolver.resolvePage(page, raw)
			if err != nil {
				return err
			}
			trust.SupersededBy = &Supersession{Authored: raw}
			if include && resolution.document && resolution.uri != "" {
				trust.SupersededBy.URI = resolution.uri
				current := false
				trust.Current = &current
				addDocumentEdge(&page.document, Edge{
					To:       resolution.uri,
					Relation: "superseded_by",
					Raw:      raw,
					Source:   "frontmatter.superseded_by",
				})
			}
		}
		for index := range trust.Maintenance {
			annotation := &trust.Maintenance[index]
			if annotation.Target == nil {
				continue
			}
			resolution, include, err := resolver.resolvePage(page, annotation.Target.Authored)
			if err != nil {
				return err
			}
			if include && resolution.document {
				annotation.Target.URI = resolution.uri
			}
		}
		page.document.Trust = trust
	}

	byURI := make(map[string]*effectivePage, len(pages))
	for _, page := range pages {
		byURI[page.document.URI] = page
	}
	for _, page := range pages {
		for _, edge := range page.document.Edges {
			if edge.Relation != "contradicts" {
				continue
			}
			addContradiction(&page.document.Trust, edge.To)
			if target := byURI[edge.To]; target != nil {
				addContradiction(&target.document.Trust, page.document.URI)
			}
		}
	}
	return nil
}

func addDocumentEdge(document *Document, edge Edge) {
	for _, existing := range document.Edges {
		if existing.To == edge.To && existing.Relation == edge.Relation {
			return
		}
	}
	document.Edges = append(document.Edges, edge)
	sort.Slice(document.Edges, func(i, j int) bool {
		if document.Edges[i].To != document.Edges[j].To {
			return document.Edges[i].To < document.Edges[j].To
		}
		return document.Edges[i].Relation < document.Edges[j].Relation
	})
	if !containsString(document.Links, edge.To) {
		document.Links = append(document.Links, edge.To)
		sort.Strings(document.Links)
	}
}

func addContradiction(trust *TrustProjection, uri string) {
	for _, existing := range trust.Contradictions {
		if existing.URI == uri {
			return
		}
	}
	trust.Contradictions = append(trust.Contradictions, Contradiction{
		URI:      uri,
		Relation: "contradicts",
	})
	sort.Slice(trust.Contradictions, func(i, j int) bool {
		return trust.Contradictions[i].URI < trust.Contradictions[j].URI
	})
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func resolveCurrent(pages []*effectivePage, start *effectivePage) CurrentResolution {
	result := CurrentResolution{
		Status: CurrentResolved,
		Chain:  []string{start.document.URI},
	}
	byURI := make(map[string]*effectivePage, len(pages))
	for _, page := range pages {
		byURI[page.document.URI] = page
	}
	seen := map[string]struct{}{start.document.URI: {}}
	current := start
	for depth := 0; depth < maxSupersessionDepth; depth++ {
		successor := current.document.Trust.SupersededBy
		if successor == nil {
			result.Current = current.document.URI
			return result
		}
		if successor.URI == "" {
			result.Status = CurrentMissingTarget
			return result
		}
		result.Chain = append(result.Chain, successor.URI)
		if _, exists := seen[successor.URI]; exists {
			result.Status = CurrentCycle
			return result
		}
		seen[successor.URI] = struct{}{}
		next := byURI[successor.URI]
		if next == nil {
			result.Status = CurrentMissingTarget
			return result
		}
		current = next
	}
	result.Status = CurrentDepthExceeded
	return result
}

func claimSignals(body string) []ClaimSignal {
	source := []byte(body)
	signals := []ClaimSignal{}
	fence := byte(0)
	for lineStart := 0; lineStart < len(source); {
		lineEnd := bytes.IndexByte(source[lineStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(source)
		} else {
			lineEnd += lineStart
		}
		line := source[lineStart:lineEnd]
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) >= 3 &&
			(bytes.HasPrefix(trimmed, []byte("```")) || bytes.HasPrefix(trimmed, []byte("~~~"))) {
			marker := trimmed[0]
			if fence == 0 {
				fence = marker
			} else if fence == marker {
				fence = 0
			}
			lineStart = lineEnd + 1
			continue
		}
		if fence != 0 {
			lineStart = lineEnd + 1
			continue
		}
		for _, marker := range []struct {
			token string
			kind  string
		}{
			{token: "^[inferred]", kind: "inferred"},
			{token: "^[ambiguous]", kind: "ambiguous"},
		} {
			for offset := 0; offset < len(line); {
				index := bytes.Index(line[offset:], []byte(marker.token))
				if index < 0 {
					break
				}
				index += offset
				if insideInlineCode(line, index) {
					offset = index + len(marker.token)
					continue
				}
				absolute := lineStart + index
				line, column := sourceLocation(source, absolute)
				signals = append(signals, ClaimSignal{
					Kind: marker.kind, Line: line, Column: column,
				})
				offset = index + len(marker.token)
			}
		}
		lineStart = lineEnd + 1
	}
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].Line != signals[j].Line {
			return signals[i].Line < signals[j].Line
		}
		if signals[i].Column != signals[j].Column {
			return signals[i].Column < signals[j].Column
		}
		return signals[i].Kind < signals[j].Kind
	})
	return signals
}

func insideInlineCode(line []byte, offset int) bool {
	for cursor := 0; cursor < len(line); {
		open := bytes.IndexByte(line[cursor:], '`')
		if open < 0 {
			return false
		}
		open += cursor
		size := backtickRun(line, open)
		close := open + size
		for close < len(line) {
			next := bytes.IndexByte(line[close:], '`')
			if next < 0 {
				return false
			}
			next += close
			nextSize := backtickRun(line, next)
			if nextSize == size {
				if offset >= open+size && offset < next {
					return true
				}
				cursor = next + size
				break
			}
			close = next + nextSize
		}
		if close >= len(line) {
			return false
		}
	}
	return false
}

func backtickRun(line []byte, start int) int {
	end := start
	for end < len(line) && line[end] == '`' {
		end++
	}
	return end - start
}

func sourceLocation(source []byte, offset int) (int, int) {
	line := bytes.Count(source[:offset], []byte("\n")) + 1
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	return line, utf8.RuneCount(source[start:offset]) + 1
}
