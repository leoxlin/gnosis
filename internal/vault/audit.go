package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	DefaultAuditPageLimit    = 1000
	MaxAuditPageLimit        = 10_000
	DefaultAuditFindingLimit = 100
	MaxAuditFindingLimit     = 1000
)

var ErrInvalidAuditCursor = errors.New("invalid audit cursor")

type FindingClass string

const (
	FindingOrphan                  FindingClass = "orphan"
	FindingDuplicateIdentity       FindingClass = "duplicate_identity"
	FindingAmbiguity               FindingClass = "ambiguity"
	FindingContradiction           FindingClass = "contradiction"
	FindingStale                   FindingClass = "stale"
	FindingBrokenSupersession      FindingClass = "broken_supersession"
	FindingCyclicSupersession      FindingClass = "cyclic_supersession"
	FindingNonCurrentSuccessor     FindingClass = "non_current_supersession"
	FindingTagFragmentation        FindingClass = "tag_fragmentation"
	FindingActiveReferenceRetired  FindingClass = "active_reference_retired"
	FindingAuthoredStale           FindingClass = "authored_stale"
	FindingAuthoredIncorrect       FindingClass = "authored_incorrect"
	FindingAuthoredDuplicate       FindingClass = "authored_duplicate"
	FindingBrokenMaintenanceTarget FindingClass = "broken_maintenance_target"
)

var AllFindingClasses = []FindingClass{
	FindingOrphan,
	FindingDuplicateIdentity,
	FindingAmbiguity,
	FindingContradiction,
	FindingStale,
	FindingBrokenSupersession,
	FindingCyclicSupersession,
	FindingNonCurrentSuccessor,
	FindingTagFragmentation,
	FindingActiveReferenceRetired,
	FindingAuthoredStale,
	FindingAuthoredIncorrect,
	FindingAuthoredDuplicate,
	FindingBrokenMaintenanceTarget,
}

var DefaultFindingClasses = []FindingClass{
	FindingOrphan,
	FindingDuplicateIdentity,
	FindingAmbiguity,
	FindingContradiction,
	FindingBrokenSupersession,
	FindingCyclicSupersession,
	FindingNonCurrentSuccessor,
	FindingTagFragmentation,
	FindingActiveReferenceRetired,
	FindingAuthoredStale,
	FindingAuthoredIncorrect,
	FindingAuthoredDuplicate,
	FindingBrokenMaintenanceTarget,
}

type FindingClassification string

const (
	ClassificationFact      FindingClassification = "fact"
	ClassificationCandidate FindingClassification = "candidate"
	ClassificationAuthored  FindingClassification = "authored"
)

type FindingSeverity string

const (
	SeverityHigh   FindingSeverity = "high"
	SeverityMedium FindingSeverity = "medium"
	SeverityLow    FindingSeverity = "low"
)

type FindingConfidence string

const (
	ConfidenceHigh   FindingConfidence = "high"
	ConfidenceMedium FindingConfidence = "medium"
)

type AuditEvidence struct {
	Kind      string `json:"kind"`
	URI       string `json:"uri,omitempty"`
	TargetURI string `json:"target_uri,omitempty"`
	Relation  string `json:"relation,omitempty"`
	Value     string `json:"value,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Line      int    `json:"line,omitempty"`
	Column    int    `json:"column,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Author    string `json:"author,omitempty"`
}

type KnowledgeFinding struct {
	ID             string                `json:"id"`
	Class          FindingClass          `json:"class"`
	Classification FindingClassification `json:"classification"`
	Severity       FindingSeverity       `json:"severity"`
	Confidence     FindingConfidence     `json:"confidence"`
	URIs           []string              `json:"uris"`
	Evidence       []AuditEvidence       `json:"evidence"`
	Procedure      string                `json:"procedure,omitempty"`
	AuthorDecision bool                  `json:"author_decision,omitempty"`
}

type KnowledgeAuditRequest struct {
	Classes      []FindingClass `json:"classes,omitempty"`
	PageLimit    int            `json:"page_limit,omitempty"`
	FindingLimit int            `json:"finding_limit,omitempty"`
	Types        []string       `json:"types,omitempty"`
	Tiers        []string       `json:"tiers,omitempty"`
	StaleAfter   string         `json:"stale_after,omitempty"`
	Cursor       string         `json:"cursor,omitempty"`
}

type AuditBound struct {
	PageLimit    int  `json:"page_limit"`
	FindingLimit int  `json:"finding_limit"`
	Truncated    bool `json:"truncated"`
}

type AuditOmission struct {
	Class  FindingClass `json:"class"`
	URI    string       `json:"uri"`
	Reason string       `json:"reason"`
}

type KnowledgeAuditResult struct {
	Classes      []FindingClass     `json:"classes"`
	PagesScanned int                `json:"pages_scanned"`
	Findings     []KnowledgeFinding `json:"findings"`
	Omissions    []AuditOmission    `json:"omissions,omitempty"`
	NextCursor   string             `json:"next_cursor,omitempty"`
	Bound        AuditBound         `json:"bound"`
}

type auditCursor struct {
	Version  int    `json:"v"`
	Kind     string `json:"kind"`
	Snapshot string `json:"snapshot"`
	Request  string `json:"request"`
	Offset   int    `json:"offset"`
}

type normalizedAuditRequest struct {
	classes      []FindingClass
	pageLimit    int
	findingLimit int
	types        map[string]struct{}
	tiers        map[string]struct{}
	staleAfter   time.Duration
	cursor       string
}

const (
	maintainVaultProcedure = "gnosis://core/procedures/maintain-vault.md"
	linkPagesProcedure     = "gnosis://core/procedures/link-pages.md"
)

func AuditKnowledge(root string, request KnowledgeAuditRequest) (KnowledgeAuditResult, error) {
	options, err := normalizeAuditRequest(request)
	if err != nil {
		return KnowledgeAuditResult{}, err
	}
	effective, err := loadEffectiveVault(root)
	if err != nil {
		return KnowledgeAuditResult{}, err
	}
	allPages, err := effective.resolvedPages()
	if err != nil {
		return KnowledgeAuditResult{}, err
	}
	sort.Slice(allPages, func(i, j int) bool {
		return allPages[i].document.URI < allPages[j].document.URI
	})
	pages := filterAuditPages(allPages, options)
	if len(pages) > options.pageLimit {
		return KnowledgeAuditResult{}, fmt.Errorf(
			"audit knowledge: %d pages match; page_limit is %d; narrow types or tiers, or raise page_limit to at most %d",
			len(pages), options.pageLimit, MaxAuditPageLimit,
		)
	}

	selected := make(map[FindingClass]bool, len(options.classes))
	for _, class := range options.classes {
		selected[class] = true
	}
	findings, omissions := detectKnowledgeFindings(
		effective, allPages, pages, selected, options.staleAfter,
	)
	sort.Slice(findings, func(i, j int) bool {
		if severityRank(findings[i].Severity) != severityRank(findings[j].Severity) {
			return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
		}
		left, right := strings.Join(findings[i].URIs, "\x00"), strings.Join(findings[j].URIs, "\x00")
		if left != right {
			return left < right
		}
		if findings[i].Class != findings[j].Class {
			return findings[i].Class < findings[j].Class
		}
		return findings[i].ID < findings[j].ID
	})
	sort.Slice(omissions, func(i, j int) bool {
		if omissions[i].URI != omissions[j].URI {
			return omissions[i].URI < omissions[j].URI
		}
		return omissions[i].Class < omissions[j].Class
	})

	snapshot := auditSnapshot(allPages)
	requestDigest := auditRequestDigest(options)
	offset := 0
	if options.cursor != "" {
		var cursor auditCursor
		if err := decodeCursor(options.cursor, &cursor); err != nil ||
			cursor.Version != 1 || cursor.Kind != "knowledge-audit" ||
			cursor.Snapshot != snapshot || cursor.Request != requestDigest ||
			cursor.Offset < 0 || cursor.Offset > len(findings) {
			return KnowledgeAuditResult{}, ErrInvalidAuditCursor
		}
		offset = cursor.Offset
	}
	end := min(offset+options.findingLimit, len(findings))
	result := KnowledgeAuditResult{
		Classes:      options.classes,
		PagesScanned: len(pages),
		Findings:     findings[offset:end],
		Omissions:    omissions,
		Bound: AuditBound{
			PageLimit: options.pageLimit, FindingLimit: options.findingLimit,
			Truncated: end < len(findings),
		},
	}
	if result.Bound.Truncated {
		result.NextCursor, err = encodeCursor(auditCursor{
			Version: 1, Kind: "knowledge-audit", Snapshot: snapshot,
			Request: requestDigest, Offset: end,
		})
		if err != nil {
			return KnowledgeAuditResult{}, err
		}
	}
	return result, nil
}

func normalizeAuditRequest(request KnowledgeAuditRequest) (normalizedAuditRequest, error) {
	result := normalizedAuditRequest{
		classes:      append([]FindingClass{}, request.Classes...),
		pageLimit:    request.PageLimit,
		findingLimit: request.FindingLimit,
		types:        stringSet(request.Types),
		tiers:        stringSet(request.Tiers),
		cursor:       strings.TrimSpace(request.Cursor),
	}
	if len(result.classes) == 0 {
		result.classes = append([]FindingClass{}, DefaultFindingClasses...)
	}
	known := make(map[FindingClass]bool, len(AllFindingClasses))
	for _, class := range AllFindingClasses {
		known[class] = true
	}
	seen := map[FindingClass]bool{}
	for _, class := range result.classes {
		if !known[class] {
			return normalizedAuditRequest{}, fmt.Errorf(
				"audit knowledge: unknown class %q; valid classes: %s",
				class, findingClassNames(),
			)
		}
		if seen[class] {
			return normalizedAuditRequest{}, fmt.Errorf("audit knowledge: duplicate class %q", class)
		}
		seen[class] = true
	}
	sort.Slice(result.classes, func(i, j int) bool { return result.classes[i] < result.classes[j] })
	if result.pageLimit == 0 {
		result.pageLimit = DefaultAuditPageLimit
	}
	if result.pageLimit < 1 || result.pageLimit > MaxAuditPageLimit {
		return normalizedAuditRequest{}, fmt.Errorf(
			"audit knowledge: page_limit must be between 1 and %d", MaxAuditPageLimit,
		)
	}
	if result.findingLimit == 0 {
		result.findingLimit = DefaultAuditFindingLimit
	}
	if result.findingLimit < 1 || result.findingLimit > MaxAuditFindingLimit {
		return normalizedAuditRequest{}, fmt.Errorf(
			"audit knowledge: finding_limit must be between 1 and %d", MaxAuditFindingLimit,
		)
	}
	if seen[FindingStale] {
		if strings.TrimSpace(request.StaleAfter) == "" {
			return normalizedAuditRequest{}, errors.New(
				"audit knowledge: stale_after is required when class \"stale\" is selected",
			)
		}
		result.staleAfter, _ = time.ParseDuration(request.StaleAfter)
		if result.staleAfter <= 0 {
			return normalizedAuditRequest{}, errors.New(
				"audit knowledge: stale_after must be a positive duration",
			)
		}
		if len(result.types) == 0 && len(result.tiers) == 0 {
			return normalizedAuditRequest{}, errors.New(
				"audit knowledge: stale audits require at least one type or tier filter",
			)
		}
	} else if strings.TrimSpace(request.StaleAfter) != "" {
		return normalizedAuditRequest{}, errors.New(
			"audit knowledge: stale_after requires class \"stale\"",
		)
	}
	return result, nil
}

func detectKnowledgeFindings(
	effective *effectiveVault,
	allPages []*effectivePage,
	pages []*effectivePage,
	selected map[FindingClass]bool,
	staleAfter time.Duration,
) ([]KnowledgeFinding, []AuditOmission) {
	graph := newDocumentGraph(allPages)
	byURI := make(map[string]*effectivePage, len(allPages))
	for _, page := range allPages {
		byURI[page.document.URI] = page
	}
	entryPoints := configuredEntryPoints(effective)
	findings := []KnowledgeFinding{}
	omissions := []AuditOmission{}

	if selected[FindingAuthoredStale] || selected[FindingAuthoredIncorrect] ||
		selected[FindingAuthoredDuplicate] || selected[FindingBrokenMaintenanceTarget] {
		for _, page := range pages {
			for _, annotation := range page.document.Trust.Maintenance {
				class := FindingAuthoredStale
				severity := SeverityMedium
				switch annotation.Kind {
				case "incorrect":
					class, severity = FindingAuthoredIncorrect, SeverityHigh
				case "duplicate":
					class = FindingAuthoredDuplicate
				}
				if selected[class] {
					evidence := AuditEvidence{
						Kind: annotation.Kind, URI: page.document.URI,
						Reason: annotation.Reason, Author: annotation.Author,
						Timestamp: annotation.ObservedAt,
					}
					uris := []string{page.document.URI}
					if annotation.Target != nil {
						evidence.Value = annotation.Target.Authored
						evidence.TargetURI = annotation.Target.URI
						if annotation.Target.URI != "" {
							uris = append(uris, annotation.Target.URI)
						}
					}
					findings = append(findings, newFinding(
						class, ClassificationAuthored, severity, ConfidenceHigh,
						uris, []AuditEvidence{evidence}, maintainVaultProcedure, false,
					))
				}
				if annotation.Target != nil && annotation.Target.URI == "" &&
					selected[FindingBrokenMaintenanceTarget] {
					findings = append(findings, newFinding(
						FindingBrokenMaintenanceTarget, ClassificationFact,
						SeverityHigh, ConfidenceHigh, []string{page.document.URI},
						[]AuditEvidence{{
							Kind: "maintenance_target", URI: page.document.URI,
							Value: annotation.Target.Authored, Reason: annotation.Reason,
							Author: annotation.Author, Timestamp: annotation.ObservedAt,
						}},
						maintainVaultProcedure, false,
					))
				}
			}
		}
	}

	if selected[FindingOrphan] {
		for _, page := range pages {
			if page.document.Type == "Concept" || entryPoints[page.document.URI] ||
				len(graph.incoming[page.document.URI]) != 0 {
				continue
			}
			findings = append(findings, newFinding(
				FindingOrphan, ClassificationFact, SeverityMedium, ConfidenceHigh,
				[]string{page.document.URI},
				[]AuditEvidence{{Kind: "inbound_link_count", URI: page.document.URI, Value: "0"}},
				linkPagesProcedure, false,
			))
		}
	}

	if selected[FindingBrokenSupersession] || selected[FindingCyclicSupersession] ||
		selected[FindingNonCurrentSuccessor] {
		seenCycles := map[string]bool{}
		for _, page := range pages {
			successor := page.document.Trust.SupersededBy
			if successor == nil {
				continue
			}
			if successor.URI == "" {
				if selected[FindingBrokenSupersession] {
					findings = append(findings, newFinding(
						FindingBrokenSupersession, ClassificationFact, SeverityHigh, ConfidenceHigh,
						[]string{page.document.URI},
						[]AuditEvidence{{
							Kind: "superseded_by", URI: page.document.URI,
							Relation: "superseded_by", Value: successor.Authored,
						}},
						maintainVaultProcedure, false,
					))
				}
				continue
			}
			resolution := resolveCurrent(allPages, page)
			if resolution.Status == CurrentCycle && selected[FindingCyclicSupersession] {
				cycle := canonicalCycle(resolution.Chain)
				key := strings.Join(cycle, "\x00")
				if !seenCycles[key] {
					seenCycles[key] = true
					findings = append(findings, newFinding(
						FindingCyclicSupersession, ClassificationFact, SeverityHigh, ConfidenceHigh,
						cycle,
						[]AuditEvidence{{
							Kind: "supersession_cycle", URI: page.document.URI,
							Relation: "superseded_by", Value: strings.Join(resolution.Chain, " -> "),
						}},
						maintainVaultProcedure, false,
					))
				}
				continue
			}
			target := byURI[successor.URI]
			if selected[FindingNonCurrentSuccessor] && target != nil &&
				(isRetired(target.document.Trust.Status) || target.document.Trust.SupersededBy != nil) {
				findings = append(findings, newFinding(
					FindingNonCurrentSuccessor, ClassificationFact, SeverityHigh, ConfidenceHigh,
					[]string{page.document.URI, target.document.URI},
					[]AuditEvidence{{
						Kind: "non_current_successor", URI: page.document.URI,
						TargetURI: target.document.URI, Relation: "superseded_by",
						Value: target.document.Trust.Status,
					}},
					maintainVaultProcedure, false,
				))
			}
		}
	}

	if selected[FindingActiveReferenceRetired] {
		for _, page := range pages {
			if isRetired(page.document.Trust.Status) {
				continue
			}
			for _, edge := range graph.outgoing[page.document.URI] {
				target := byURI[edge.To.URI]
				if target == nil || !isRetired(target.document.Trust.Status) ||
					edge.Relation == "superseded_by" {
					continue
				}
				findings = append(findings, newFinding(
					FindingActiveReferenceRetired, ClassificationFact, SeverityHigh, ConfidenceHigh,
					[]string{page.document.URI, target.document.URI},
					[]AuditEvidence{{
						Kind: "active_reference", URI: page.document.URI,
						TargetURI: target.document.URI, Relation: edge.Relation,
						Value: target.document.Trust.Status,
					}},
					maintainVaultProcedure, false,
				))
			}
		}
	}

	if selected[FindingDuplicateIdentity] {
		type identityUse struct {
			pages    map[string]*effectivePage
			evidence map[string]AuditEvidence
		}
		groups := map[string]*identityUse{}
		for _, page := range pages {
			values := append([]string{page.document.Title}, page.document.Aliases...)
			for _, value := range values {
				identity := normalizeIdentity(value)
				if identity == "" {
					continue
				}
				key := page.document.Type + "\x00" + identity
				if groups[key] == nil {
					groups[key] = &identityUse{
						pages: map[string]*effectivePage{}, evidence: map[string]AuditEvidence{},
					}
				}
				groups[key].pages[page.document.URI] = page
				groups[key].evidence[page.document.URI] = AuditEvidence{
					Kind: "title_or_alias", URI: page.document.URI, Value: value,
				}
			}
		}
		for key, group := range groups {
			if len(group.pages) < 2 {
				continue
			}
			authoredDuplicate := false
			for _, page := range group.pages {
				for _, annotation := range page.document.Trust.Maintenance {
					if annotation.Kind == "duplicate" && annotation.Target != nil &&
						group.pages[annotation.Target.URI] != nil {
						authoredDuplicate = true
					}
				}
			}
			if authoredDuplicate {
				continue
			}
			parts := strings.SplitN(key, "\x00", 2)
			uris := mapKeys(group.pages)
			evidence := []AuditEvidence{{Kind: "normalized_identity", Value: parts[1]}}
			for _, uri := range uris {
				evidence = append(evidence, group.evidence[uri])
			}
			findings = append(findings, newFinding(
				FindingDuplicateIdentity, ClassificationCandidate, SeverityLow, ConfidenceMedium,
				uris,
				evidence,
				maintainVaultProcedure, true,
			))
		}
	}

	if selected[FindingAmbiguity] || selected[FindingContradiction] {
		seenContradictions := map[string]bool{}
		for _, page := range pages {
			if selected[FindingAmbiguity] {
				evidence := []AuditEvidence{}
				for _, signal := range page.document.Trust.Claims {
					if signal.Kind == "ambiguous" {
						evidence = append(evidence, AuditEvidence{
							Kind: signal.Kind, URI: page.document.URI,
							Line: signal.Line, Column: signal.Column,
						})
					}
				}
				if len(evidence) > 0 {
					findings = append(findings, newFinding(
						FindingAmbiguity, ClassificationCandidate, SeverityMedium, ConfidenceHigh,
						[]string{page.document.URI}, evidence,
						maintainVaultProcedure, true,
					))
				}
			}
			if selected[FindingContradiction] {
				for _, contradiction := range page.document.Trust.Contradictions {
					uris := []string{page.document.URI, contradiction.URI}
					sort.Strings(uris)
					key := strings.Join(uris, "\x00")
					if seenContradictions[key] {
						continue
					}
					seenContradictions[key] = true
					findings = append(findings, newFinding(
						FindingContradiction, ClassificationCandidate, SeverityHigh, ConfidenceHigh,
						uris,
						[]AuditEvidence{{
							Kind: "authored_relationship", URI: page.document.URI,
							TargetURI: contradiction.URI, Relation: contradiction.Relation,
						}},
						maintainVaultProcedure, true,
					))
				}
			}
		}
	}

	if selected[FindingStale] {
		now := time.Now().UTC()
		for _, page := range pages {
			timestamp, source := auditTimestamp(page)
			if timestamp.IsZero() {
				omissions = append(omissions, AuditOmission{
					Class: FindingStale, URI: page.document.URI, Reason: "timestamp unavailable",
				})
				continue
			}
			if now.Sub(timestamp) <= staleAfter {
				continue
			}
			findings = append(findings, newFinding(
				FindingStale, ClassificationCandidate, SeverityMedium, ConfidenceHigh,
				[]string{page.document.URI},
				[]AuditEvidence{{
					Kind: source, URI: page.document.URI, Timestamp: timestamp.Format(time.RFC3339),
				}},
				maintainVaultProcedure, true,
			))
		}
	}

	if selected[FindingTagFragmentation] {
		type tagUse struct {
			variants map[string]bool
			uris     map[string]bool
		}
		groups := map[string]*tagUse{}
		for _, page := range pages {
			for _, tag := range page.document.Tags {
				normalized := normalizeIdentity(tag)
				if normalized == "" {
					continue
				}
				if groups[normalized] == nil {
					groups[normalized] = &tagUse{variants: map[string]bool{}, uris: map[string]bool{}}
				}
				groups[normalized].variants[tag] = true
				groups[normalized].uris[page.document.URI] = true
			}
		}
		for normalized, group := range groups {
			if len(group.variants) < 2 {
				continue
			}
			variants := boolMapKeys(group.variants)
			findings = append(findings, newFinding(
				FindingTagFragmentation, ClassificationCandidate, SeverityLow, ConfidenceHigh,
				boolMapKeys(group.uris),
				[]AuditEvidence{{
					Kind: "tag_variants", Value: normalized + ": " + strings.Join(variants, ", "),
				}},
				maintainVaultProcedure, true,
			))
		}
	}
	return findings, omissions
}

func filterAuditPages(pages []*effectivePage, options normalizedAuditRequest) []*effectivePage {
	result := make([]*effectivePage, 0, len(pages))
	for _, page := range pages {
		if len(options.types) > 0 {
			if _, ok := options.types[page.document.Type]; !ok {
				continue
			}
		}
		if len(options.tiers) > 0 {
			if _, ok := options.tiers[page.document.Trust.Tier]; !ok {
				continue
			}
		}
		result = append(result, page)
	}
	return result
}

func newFinding(
	class FindingClass,
	classification FindingClassification,
	severity FindingSeverity,
	confidence FindingConfidence,
	uris []string,
	evidence []AuditEvidence,
	procedure string,
	authorDecision bool,
) KnowledgeFinding {
	uris = append([]string{}, uris...)
	sort.Strings(uris)
	identity, _ := json.Marshal(struct {
		Class    FindingClass
		URIs     []string
		Evidence []AuditEvidence
	}{class, uris, evidence})
	return KnowledgeFinding{
		ID: digestString(string(identity)), Class: class, Classification: classification,
		Severity: severity, Confidence: confidence, URIs: uris, Evidence: evidence,
		Procedure: procedure, AuthorDecision: authorDecision,
	}
}

func configuredEntryPoints(effective *effectiveVault) map[string]bool {
	result := map[string]bool{}
	for _, source := range effective.sources {
		for _, uri := range source.config.Vault.EntryPoints {
			result[strings.TrimSpace(uri)] = true
		}
	}
	return result
}

func canonicalCycle(chain []string) []string {
	if len(chain) > 1 && chain[0] == chain[len(chain)-1] {
		chain = chain[:len(chain)-1]
	}
	result := append([]string{}, chain...)
	sort.Strings(result)
	return result
}

func normalizeIdentity(value string) string {
	var normalized strings.Builder
	space := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && normalized.Len() > 0 {
				normalized.WriteByte(' ')
			}
			space = false
			normalized.WriteRune(r)
		} else {
			space = true
		}
	}
	return normalized.String()
}

func auditTimestamp(page *effectivePage) (time.Time, string) {
	for _, candidate := range []struct {
		value  string
		source string
	}{
		{page.document.Trust.ObservedAt, "observed_at"},
		{page.document.Trust.ValidUntil, "valid_until"},
		{page.document.Trust.ValidFrom, "valid_from"},
		{page.document.Trust.OccurredAt, "occurred_at"},
	} {
		if parsed, ok := parseAuditTime(candidate.value); ok {
			return parsed, candidate.source
		}
	}
	repository, ok := repositoryAt(filepath.Dir(page.path))
	if !ok {
		return time.Time{}, ""
	}
	relative, err := filepathRelSlash(repository.root, page.path)
	if err != nil {
		return time.Time{}, ""
	}
	records, err := pageGitHistory(repository.root, relative)
	if err != nil || len(records) == 0 {
		return time.Time{}, ""
	}
	parsed, ok := parseAuditTime(records[0].timestamp)
	if !ok {
		return time.Time{}, ""
	}
	return parsed, "committed_at"
}

func parseAuditTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.DateOnly} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func filepathRelSlash(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

func isRetired(status string) bool {
	return status == "archived" || status == "retired"
}

func severityRank(severity FindingSeverity) int {
	switch severity {
	case SeverityHigh:
		return 0
	case SeverityMedium:
		return 1
	default:
		return 2
	}
}

func auditSnapshot(pages []*effectivePage) string {
	parts := make([]string, 0, len(pages))
	for _, page := range pages {
		parts = append(parts, page.document.URI+"\x00"+page.document.Revision)
	}
	return digestString(strings.Join(parts, "\n"))
}

func auditRequestDigest(options normalizedAuditRequest) string {
	types, tiers := setKeys(options.types), setKeys(options.tiers)
	parts := []string{
		findingClassesString(options.classes),
		strconv.Itoa(options.pageLimit),
		strconv.Itoa(options.findingLimit),
		strings.Join(types, "\x00"),
		strings.Join(tiers, "\x00"),
		options.staleAfter.String(),
	}
	return digestString(strings.Join(parts, "\n"))
}

func findingClassNames() string {
	return findingClassesString(AllFindingClasses)
}

func findingClassesString(classes []FindingClass) string {
	names := make([]string, len(classes))
	for i, class := range classes {
		names[i] = string(class)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func stringSet(values []string) map[string]struct{} {
	result := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func setKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func mapKeys(values map[string]*effectivePage) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func boolMapKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
