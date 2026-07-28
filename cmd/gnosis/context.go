package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	evidencecontext "gnosis/internal/evidencecontext"
	"gnosis/internal/vault"
)

func newContextCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "context",
		Short:   "Resolve bounded agent context",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis context knowledge \"<question>\"\n" +
			"gnosis context knowledge \"<question>\" --strategy hybrid",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(fmt.Errorf("context: missing resource"))
		},
	}
	command.AddCommand(newContextKnowledgeCommand(options, stdout))
	return command
}

func newContextKnowledgeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var strategy, conceptType, status, source, asOf, tier, relationship, relationshipTarget string
	var tags []string
	var minConfidence float64
	var maxEvidence, maxChars, maxDepth int
	command := &cobra.Command{
		Use:   "knowledge <question> [flags]",
		Short: "Resolve a bounded evidence packet without generating an answer",
		Args:  questionArgs("context knowledge"),
		Example: "gnosis context knowledge \"<question>\"\n" +
			"gnosis context knowledge \"<question>\" --type Policy --status active",
		RunE: func(command *cobra.Command, args []string) error {
			request := evidencecontext.Request{
				Question:    args[0],
				Strategy:    evidencecontext.Strategy(strategy),
				MaxEvidence: intPointer(maxEvidence),
				MaxChars:    intPointer(maxChars),
				MaxDepth:    intPointer(maxDepth),
				Constraints: evidencecontext.Constraints{
					Type:   strings.TrimSpace(conceptType),
					Status: strings.TrimSpace(status),
					Tags:   tags,
					Source: strings.TrimSpace(source),
					AsOf:   strings.TrimSpace(asOf),
					Tier:   strings.TrimSpace(tier),
				},
			}
			if command.Flags().Changed("min-confidence") {
				request.Constraints.MinConfidence = &minConfidence
			}
			if relationship != "" || relationshipTarget != "" {
				request.Constraints.Relationship = &evidencecontext.RelationshipConstraint{
					Type: strings.TrimSpace(relationship), Target: strings.TrimSpace(relationshipTarget),
				}
			}
			if err := evidencecontext.Validate(request); err != nil {
				return newUsageError(fmt.Errorf("context knowledge: %w", err))
			}
			result, err := evidencecontext.Resolve(command.Context(), options.vaultPath, request)
			if err != nil {
				return err
			}
			return writeContextResult(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&strategy, "strategy", string(evidencecontext.StrategyLexical), "retrieval strategy: lexical, vector, or hybrid")
	flags.StringVar(&conceptType, "type", "", "exact concept type")
	flags.StringVar(&status, "status", "", "exact lifecycle status")
	flags.StringSliceVar(&tags, "tag", nil, "required tag")
	flags.StringVar(&source, "source", "", "exact source")
	flags.StringVar(&asOf, "as-of", "", "RFC3339 validity or observation time")
	flags.Float64Var(&minConfidence, "min-confidence", 0, "minimum confidence from 0 through 1")
	flags.StringVar(&tier, "tier", "", "exact evidence tier")
	flags.StringVar(&relationship, "relationship", "", "exact relationship type")
	flags.StringVar(&relationshipTarget, "relationship-target", "", "exact relationship target URI or authored value")
	flags.IntVar(&maxEvidence, "max-evidence", evidencecontext.DefaultMaxEvidence, "maximum evidence excerpts")
	flags.IntVar(&maxChars, "max-chars", evidencecontext.DefaultMaxChars, "maximum excerpt characters")
	flags.IntVar(&maxDepth, "depth", evidencecontext.DefaultMaxDepth, "maximum typed-path depth")
	return command
}

func writeContextResult(output io.Writer, result evidencecontext.Result) error {
	passes := make([]toon.Object, 0, len(result.Passes))
	for _, pass := range result.Passes {
		passes = append(passes, toon.NewObject(
			toon.Field{Key: "backend", Value: pass.Backend},
			toon.Field{Key: "candidates", Value: pass.Candidates},
		))
	}
	evidence := make([]toon.Object, 0, len(result.Evidence))
	for _, item := range result.Evidence {
		confidence := 0.0
		hasConfidence := item.Trust.Confidence != nil
		if hasConfidence {
			confidence = *item.Trust.Confidence
		}
		evidence = append(evidence, toon.NewObject(
			toon.Field{Key: "uri", Value: item.Citation.URI},
			toon.Field{Key: "revision", Value: item.Citation.Revision},
			toon.Field{Key: "origin_vault", Value: item.Citation.Origin.Vault},
			toon.Field{Key: "origin_kind", Value: string(item.Citation.Origin.Kind)},
			toon.Field{Key: "origin_root", Value: item.Citation.Origin.Root},
			toon.Field{Key: "origin_path", Value: item.Citation.Origin.Path},
			toon.Field{Key: "origin_precedence", Value: item.Citation.Origin.Precedence},
			toon.Field{Key: "type", Value: item.Type},
			toon.Field{Key: "title", Value: item.Title},
			toon.Field{Key: "score", Value: item.Score},
			toon.Field{Key: "status", Value: item.Trust.Status},
			toon.Field{Key: "source", Value: item.Trust.Source},
			toon.Field{Key: "tier", Value: item.Trust.Tier},
			toon.Field{Key: "has_confidence", Value: hasConfidence},
			toon.Field{Key: "confidence", Value: confidence},
			toon.Field{Key: "section", Value: item.Excerpt.Section},
			toon.Field{Key: "excerpt", Value: item.Excerpt.Content},
			toon.Field{Key: "truncated", Value: item.Excerpt.Truncated},
		))
	}
	paths := make([]toon.Object, 0, len(result.Paths))
	for _, path := range result.Paths {
		relations := make([]string, 0, len(path.Edges))
		for _, edge := range path.Edges {
			relations = append(relations, edge.Relation)
		}
		paths = append(paths, toon.NewObject(
			toon.Field{Key: "status", Value: string(path.Status)},
			toon.Field{Key: "from", Value: path.From.URI},
			toon.Field{Key: "to", Value: path.To.URI},
			toon.Field{Key: "max_depth", Value: path.MaxDepth},
			toon.Field{Key: "nodes", Value: strings.Join(pathURIs(path.Nodes), " -> ")},
			toon.Field{Key: "relations", Value: strings.Join(relations, " -> ")},
		))
	}
	gaps := make([]toon.Object, 0, len(result.Gaps))
	for _, gap := range result.Gaps {
		gaps = append(gaps, toon.NewObject(
			toon.Field{Key: "kind", Value: gap.Kind},
			toon.Field{Key: "message", Value: gap.Message},
			toon.Field{Key: "from", Value: gap.From},
			toon.Field{Key: "to", Value: gap.To},
		))
	}
	omissions := make([]toon.Object, 0, len(result.Omissions))
	for _, omission := range result.Omissions {
		omissions = append(omissions, toon.NewObject(
			toon.Field{Key: "uri", Value: omission.URI},
			toon.Field{Key: "reason", Value: omission.Reason},
		))
	}
	fields := []toon.Field{
		{Key: "question", Value: result.Question},
		{Key: "strategy", Value: string(result.Strategy)},
		{Key: "passes", Value: passes},
		{Key: "evidence", Value: evidence},
		{Key: "paths", Value: paths},
		{Key: "gaps", Value: gaps},
		{Key: "omissions", Value: omissions},
		{Key: "budget", Value: toon.NewObject(
			toon.Field{Key: "max_evidence", Value: result.Budget.MaxEvidence},
			toon.Field{Key: "max_chars", Value: result.Budget.MaxChars},
			toon.Field{Key: "max_depth", Value: result.Budget.MaxDepth},
			toon.Field{Key: "used_evidence", Value: result.Budget.UsedEvidence},
			toon.Field{Key: "used_chars", Value: result.Budget.UsedChars},
			toon.Field{Key: "truncated", Value: result.Budget.Truncated},
		)},
	}
	if result.Budget.Truncated {
		fields = append(fields, toon.Field{Key: "help", Value: result.Continuation})
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func pathURIs(documents []vault.DocumentRef) []string {
	result := make([]string, 0, len(documents))
	for _, document := range documents {
		result = append(result, document.URI)
	}
	return result
}

func intPointer(value int) *int {
	return &value
}
