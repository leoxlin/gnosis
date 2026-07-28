package evidencecontext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/search"
)

func TestResolveAppliesExactConstraints(t *testing.T) {
	root := contextTestVault(t)
	minConfidence := 0.8
	tests := []struct {
		name        string
		constraints Constraints
		reason      string
	}{
		{name: "type", constraints: Constraints{Type: "Policy"}, reason: "type"},
		{name: "status", constraints: Constraints{Status: "active"}, reason: "status"},
		{name: "tag", constraints: Constraints{Tags: []string{"security"}}, reason: "tag"},
		{name: "source", constraints: Constraints{Source: "handbook"}, reason: "source"},
		{name: "time", constraints: Constraints{AsOf: "2026-06-01T00:00:00Z"}, reason: "time_undetermined"},
		{name: "confidence", constraints: Constraints{MinConfidence: &minConfidence}, reason: "confidence"},
		{name: "tier", constraints: Constraints{Tier: "core"}, reason: "tier"},
		{
			name: "relationship",
			constraints: Constraints{Relationship: &RelationshipConstraint{
				Type: "applies_to", Target: "gnosis://test/product.md",
			}},
			reason: "relationship",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Resolve(context.Background(), root, testRequest("retention", test.constraints))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Evidence) != 1 ||
				result.Evidence[0].Citation.URI != "gnosis://test/policy.md" {
				t.Fatalf("evidence = %+v", result.Evidence)
			}
			if !hasOmission(result.Omissions, "gnosis://test/other.md", test.reason) {
				t.Fatalf("omissions = %+v, want %q", result.Omissions, test.reason)
			}
		})
	}
}

func TestResolveBudgetsExcerptsAndReportsOmissions(t *testing.T) {
	root := contextTestVault(t)
	maxEvidence, maxChars, maxDepth := 1, 24, 2
	result, err := Resolve(context.Background(), root, Request{
		Question:    "retention",
		Strategy:    StrategyLexical,
		MaxEvidence: &maxEvidence,
		MaxChars:    &maxChars,
		MaxDepth:    &maxDepth,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 1 || result.Budget.UsedEvidence != 1 ||
		result.Budget.UsedChars > maxChars || !result.Budget.Truncated ||
		!result.Evidence[0].Excerpt.Truncated {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(strings.ToLower(result.Evidence[0].Excerpt.Section), "retention") {
		t.Fatalf("excerpt = %+v", result.Evidence[0].Excerpt)
	}
	if !hasReason(result.Omissions, "evidence_limit") {
		t.Fatalf("omissions = %+v", result.Omissions)
	}
}

func TestResolveStrategiesAreExplicitAndDeduplicateURIs(t *testing.T) {
	root := contextTestVault(t)
	semantic := func(
		_ context.Context,
		_, _ string,
		_ search.QueryOptions,
	) (search.QueryResult, error) {
		return search.QueryResult{Candidates: []search.Candidate{
			{URI: "gnosis://test/other.md"},
			{URI: "gnosis://test/policy.md"},
		}}, nil
	}
	request := testRequest("retention", Constraints{})
	request.Strategy = StrategyHybrid
	result, err := ResolveWithSemantic(context.Background(), root, request, semantic)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Passes) != 2 || result.Passes[0].Backend != "lexical" ||
		result.Passes[1].Backend != "vector" || len(result.Evidence) != 3 {
		t.Fatalf("result = %+v", result)
	}
	seen := map[string]bool{}
	for _, evidence := range result.Evidence {
		if seen[evidence.Citation.URI] {
			t.Fatalf("evidence was not de-duplicated: %+v", result.Evidence)
		}
		seen[evidence.Citation.URI] = true
	}

	request.Strategy = StrategyVector
	result, err = ResolveWithSemantic(context.Background(), root, request, semantic)
	if err != nil {
		t.Fatal(err)
	}
	if result.Evidence[0].Citation.URI != "gnosis://test/other.md" {
		t.Fatalf("vector evidence = %+v", result.Evidence)
	}

	_, err = ResolveWithSemantic(
		context.Background(), root, request,
		func(context.Context, string, string, search.QueryOptions) (search.QueryResult, error) {
			return search.QueryResult{}, errors.New("semantic unavailable")
		},
	)
	if err == nil || !strings.Contains(err.Error(), "vector pass failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveReturnsTypedPathsAndExplicitGaps(t *testing.T) {
	root := contextTestVault(t)
	result, err := Resolve(context.Background(), root, testRequest("retention product", Constraints{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Paths) == 0 || result.Paths[0].Status != "found" ||
		len(result.Paths[0].Edges) != 1 ||
		result.Paths[0].Edges[0].Relation != "applies_to" {
		t.Fatalf("paths = %+v; gaps = %+v", result.Paths, result.Gaps)
	}

	result, err = Resolve(context.Background(), root, testRequest("zyxwv unmatched", Constraints{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 0 || len(result.Gaps) != 1 ||
		result.Gaps[0].Kind != "no_match" {
		t.Fatalf("result = %+v", result)
	}
}

func TestValidateRejectsInvalidContextRequests(t *testing.T) {
	zero, one := 0, 1
	tests := []Request{
		Defaults(Request{Question: " "}),
		{Question: "q", Strategy: "other", MaxEvidence: &one, MaxChars: &one, MaxDepth: &one},
		{Question: "q", Strategy: StrategyLexical, MaxEvidence: &zero, MaxChars: &one, MaxDepth: &one},
		{Question: "q", Strategy: StrategyLexical, MaxEvidence: &one, MaxChars: &zero, MaxDepth: &one},
		{Question: "q", Strategy: StrategyLexical, MaxEvidence: &one, MaxChars: &one, MaxDepth: &zero},
		Defaults(Request{Question: "q", Constraints: Constraints{Tags: []string{""}}}),
		Defaults(Request{Question: "q", Constraints: Constraints{AsOf: "today"}}),
		Defaults(Request{Question: "q", Constraints: Constraints{
			Relationship: &RelationshipConstraint{Type: "uses"},
		}}),
	}
	for _, request := range tests {
		if err := Validate(request); err == nil {
			t.Fatalf("Validate(%+v) succeeded", request)
		}
	}
}

func testRequest(question string, constraints Constraints) Request {
	maxEvidence, maxChars, maxDepth := 5, 6_000, 2
	return Request{
		Question: question, Strategy: StrategyLexical, Constraints: constraints,
		MaxEvidence: &maxEvidence, MaxChars: &maxChars, MaxDepth: &maxDepth,
	}
}

func hasOmission(omissions []Omission, uri, reason string) bool {
	for _, omission := range omissions {
		if omission.URI == uri && omission.Reason == reason {
			return true
		}
	}
	return false
}

func hasReason(omissions []Omission, reason string) bool {
	for _, omission := range omissions {
		if omission.Reason == reason {
			return true
		}
	}
	return false
}

func contextTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeContextFile(t, root, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	writeContextFile(t, root, "policy.md", `---
type: Policy
title: Retention policy
description: Retention requirements for product data.
status: active
tags: [security]
source: handbook
valid_from: "2026-01-01T00:00:00Z"
valid_until: "2026-12-31T00:00:00Z"
confidence: 0.9
tier: core
relationships:
  - type: applies_to
    target: product.md
---

# Retention policy

Retain product records for the required period.

## Exceptions

Delete test data earlier.
`)
	writeContextFile(t, root, "other.md", `---
type: Note
title: Other retention note
description: A lower-priority retention note.
status: archived
tags: [other]
source: archive
confidence: 0.2
tier: edge
---

# Other retention note

Retention is mentioned without an applicable policy.
`)
	writeContextFile(t, root, "product.md", `---
type: Product
title: Product records
description: Product evidence linked from the retention policy.
---

# Product records

Product records are governed by the retention policy.
`)
	return root
}

func writeContextFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
