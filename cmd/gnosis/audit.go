package main

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newAuditCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "audit",
		Short:   "Audit knowledge health without changing the vault",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis audit knowledge\n" +
			"gnosis audit knowledge --class orphan --type Note",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("audit: missing subject"))
		},
	}
	command.AddCommand(newAuditKnowledgeCommand(options, stdout))
	return command
}

func newAuditKnowledgeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var classes []string
	var types []string
	var tiers []string
	var pageLimit int
	var findingLimit int
	var staleAfter string
	var cursor string
	command := &cobra.Command{
		Use:   "knowledge [flags]",
		Short: "Report bounded deterministic knowledge-health findings",
		Args:  cobra.NoArgs,
		Example: "gnosis audit knowledge\n" +
			"gnosis audit knowledge --class orphan,duplicate_identity --type Note\n" +
			"gnosis audit knowledge --class stale --tier core --stale-after 720h",
		RunE: func(_ *cobra.Command, _ []string) error {
			selected := make([]vault.FindingClass, len(classes))
			for index, class := range classes {
				selected[index] = vault.FindingClass(strings.TrimSpace(class))
			}
			result, err := vault.AuditKnowledge(options.vaultPath, vault.KnowledgeAuditRequest{
				Classes: selected, PageLimit: pageLimit, FindingLimit: findingLimit,
				Types: types, Tiers: tiers, StaleAfter: staleAfter, Cursor: cursor,
			})
			if err != nil {
				if strings.HasPrefix(err.Error(), "audit knowledge:") ||
					errors.Is(err, vault.ErrInvalidAuditCursor) {
					return newUsageError(err)
				}
				return err
			}
			return writeAuditTOON(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringSliceVar(&classes, "class", nil, "finding classes")
	flags.StringSliceVar(&types, "type", nil, "exact concept type filters")
	flags.StringSliceVar(&tiers, "tier", nil, "exact knowledge tier filters")
	flags.IntVar(&pageLimit, "page-limit", vault.DefaultAuditPageLimit, "maximum matching pages")
	flags.IntVar(&findingLimit, "finding-limit", vault.DefaultAuditFindingLimit, "maximum findings")
	flags.StringVar(&staleAfter, "stale-after", "", "required positive duration for stale findings")
	flags.StringVar(&cursor, "cursor", "", "opaque finding continuation cursor")
	return command
}

func writeAuditTOON(output io.Writer, result vault.KnowledgeAuditResult) error {
	findings := make([]toon.Object, 0, len(result.Findings))
	for _, finding := range result.Findings {
		evidence := make([]toon.Object, 0, len(finding.Evidence))
		for _, item := range finding.Evidence {
			fields := []toon.Field{{Key: "kind", Value: item.Kind}}
			for _, field := range []struct {
				key   string
				value string
			}{
				{"uri", item.URI},
				{"target_uri", item.TargetURI},
				{"relation", item.Relation},
				{"value", item.Value},
				{"timestamp", item.Timestamp},
			} {
				if field.value != "" {
					fields = append(fields, toon.Field{Key: field.key, Value: field.value})
				}
			}
			if item.Line != 0 {
				fields = append(fields,
					toon.Field{Key: "line", Value: item.Line},
					toon.Field{Key: "column", Value: item.Column},
				)
			}
			evidence = append(evidence, toon.NewObject(fields...))
		}
		findings = append(findings, toon.NewObject(
			toon.Field{Key: "id", Value: finding.ID},
			toon.Field{Key: "class", Value: string(finding.Class)},
			toon.Field{Key: "classification", Value: string(finding.Classification)},
			toon.Field{Key: "severity", Value: string(finding.Severity)},
			toon.Field{Key: "confidence", Value: string(finding.Confidence)},
			toon.Field{Key: "uris", Value: finding.URIs},
			toon.Field{Key: "evidence", Value: evidence},
			toon.Field{Key: "procedure", Value: finding.Procedure},
			toon.Field{Key: "author_decision", Value: finding.AuthorDecision},
		))
	}
	omissions := make([]toon.Object, 0, len(result.Omissions))
	for _, omission := range result.Omissions {
		omissions = append(omissions, toon.NewObject(
			toon.Field{Key: "class", Value: string(omission.Class)},
			toon.Field{Key: "uri", Value: omission.URI},
			toon.Field{Key: "reason", Value: omission.Reason},
		))
	}
	classes := make([]string, len(result.Classes))
	for index, class := range result.Classes {
		classes[index] = string(class)
	}
	fields := []toon.Field{
		{Key: "classes", Value: classes},
		{Key: "pages_scanned", Value: result.PagesScanned},
		{Key: "count", Value: len(findings)},
		{Key: "findings", Value: findings},
		{Key: "omissions", Value: omissions},
		{Key: "bound", Value: toon.NewObject(
			toon.Field{Key: "page_limit", Value: result.Bound.PageLimit},
			toon.Field{Key: "finding_limit", Value: result.Bound.FindingLimit},
			toon.Field{Key: "truncated", Value: result.Bound.Truncated},
		)},
	}
	if result.NextCursor != "" {
		fields = append(fields,
			toon.Field{Key: "next_cursor", Value: result.NextCursor},
			toon.Field{Key: "help", Value: []string{
				"Pass next_cursor to `gnosis audit knowledge --cursor <cursor>` with the same filters.",
			}},
		)
	}
	return writeTOON(output, toon.NewObject(fields...))
}
