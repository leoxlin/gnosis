package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newPlanCommand(options *rootOptions, input io.Reader, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "plan",
		Short:   "Plan a vault resource change without side effects",
		Args:    cobra.NoArgs,
		GroupID: "basic",
		Example: "gnosis plan knowledge-change <gnosis-uri> --expected-absent --filename <file>\n" +
			"gnosis plan knowledge-change <gnosis-uri> --expected-revision <revision>",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("plan: missing resource"))
		},
	}
	command.AddCommand(newPlanKnowledgeChangeCommand(options, input, stdout))
	return command
}

func newPlanKnowledgeChangeCommand(
	options *rootOptions,
	input io.Reader,
	stdout io.Writer,
) *cobra.Command {
	var filename, expectedRevision string
	var expectedAbsent bool
	command := &cobra.Command{
		Use:   "knowledge-change <gnosis-uri> [flags]",
		Short: "Validate and diff one complete typed Markdown candidate",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis plan knowledge-change <gnosis-uri> --expected-absent --filename <file>\n" +
			"gnosis plan knowledge-change <gnosis-uri> --expected-revision <revision> < <file>",
		RunE: func(_ *cobra.Command, args []string) error {
			change, err := knowledgeChangeInput(
				input,
				filename,
				args[0],
				expectedRevision,
				expectedAbsent,
				"plan knowledge-change",
			)
			if err != nil {
				return err
			}
			plan, err := vault.PlanKnowledgeChange(options.vaultPath, change)
			if err != nil {
				return err
			}
			return writeKnowledgeChangePlan(stdout, plan)
		},
	}
	flags := command.Flags()
	flags.StringVarP(&filename, "filename", "f", "", "read complete Markdown candidate from this file")
	flags.StringVar(&expectedRevision, "expected-revision", "", "required current revision for an update or archive")
	flags.BoolVar(&expectedAbsent, "expected-absent", false, "require the target page to be absent")
	return command
}

func knowledgeChangeInput(
	input io.Reader,
	filename,
	rawURI,
	expectedRevision string,
	expectedAbsent bool,
	operation string,
) (vault.KnowledgeChangeInput, error) {
	uri := strings.TrimSpace(rawURI)
	if !vault.IsCanonicalURI(uri) {
		return vault.KnowledgeChangeInput{}, newUsageError(
			fmt.Errorf("%s: argument must be a canonical gnosis URI", operation),
		)
	}
	expectedRevision = strings.TrimSpace(expectedRevision)
	if expectedAbsent == (expectedRevision != "") {
		return vault.KnowledgeChangeInput{}, newUsageError(
			fmt.Errorf("%s: provide exactly one of --expected-absent or --expected-revision", operation),
		)
	}
	content, err := readMarkdownInput(input, filename, operation)
	if err != nil {
		return vault.KnowledgeChangeInput{}, err
	}
	return vault.KnowledgeChangeInput{
		URI:              uri,
		Candidate:        string(content),
		ExpectedRevision: expectedRevision,
		ExpectedAbsent:   expectedAbsent,
	}, nil
}

func writeKnowledgeChangePlan(output io.Writer, plan vault.KnowledgeChangePlan) error {
	findings := make([]toon.Object, 0, len(plan.Findings))
	for _, finding := range plan.Findings {
		findings = append(findings, toon.NewObject(
			toon.Field{Key: "severity", Value: finding.Severity},
			toon.Field{Key: "code", Value: finding.Code},
			toon.Field{Key: "message", Value: finding.Message},
		))
	}
	relationships := make([]toon.Object, 0, len(plan.AffectedRelationships))
	for _, relationship := range plan.AffectedRelationships {
		relationships = append(relationships, toon.NewObject(
			toon.Field{Key: "change", Value: relationship.Change},
			toon.Field{Key: "from", Value: relationship.From},
			toon.Field{Key: "relation", Value: relationship.Relation},
			toon.Field{Key: "to", Value: relationship.To},
		))
	}
	fields := []toon.Field{
		{Key: "action", Value: "plan"},
		{Key: "resource", Value: "knowledge-change"},
		{Key: "uri", Value: plan.URI},
		{Key: "operation", Value: plan.Operation},
		{Key: "applicable", Value: plan.Applicable},
		{Key: "digest", Value: plan.Digest},
	}
	if plan.ExpectedAbsent {
		fields = append(fields, toon.Field{Key: "expected_absent", Value: true})
	}
	if plan.ExpectedRevision != "" {
		fields = append(fields, toon.Field{Key: "expected_revision", Value: plan.ExpectedRevision})
	}
	if plan.BeforeRevision != "" {
		fields = append(fields, toon.Field{Key: "before_revision", Value: plan.BeforeRevision})
	}
	if plan.AfterRevision != "" {
		fields = append(fields, toon.Field{Key: "after_revision", Value: plan.AfterRevision})
	}
	fields = append(fields,
		toon.Field{Key: "findings", Value: findings},
		toon.Field{Key: "affected_relationships", Value: relationships},
		toon.Field{Key: "diff", Value: plan.Diff},
	)
	return writeTOON(output, toon.NewObject(fields...))
}

func writeKnowledgeChangeResult(output io.Writer, result vault.KnowledgeChangeResult) error {
	status := "applied"
	if !result.Changed {
		status = "no-op"
	}
	fields := []toon.Field{
		toon.Field{Key: "action", Value: "apply"},
		toon.Field{Key: "resource", Value: "knowledge-change"},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "operation", Value: result.Operation},
		toon.Field{Key: "uri", Value: result.URI},
		toon.Field{Key: "path", Value: result.Path},
		toon.Field{Key: "revision", Value: result.Revision},
		toon.Field{Key: "changed", Value: result.Changed},
	}
	if len(result.Deliveries) > 0 {
		fields = append(fields, toon.Field{Key: "deliveries", Value: hookDeliveryObjects(result.Deliveries)})
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func hookDeliveryObjects(deliveries []vault.HookDeliveryResult) []toon.Object {
	rows := make([]toon.Object, 0, len(deliveries))
	for _, delivery := range deliveries {
		fields := []toon.Field{
			{Key: "name", Value: delivery.Name},
			{Key: "kind", Value: delivery.Kind},
			{Key: "status", Value: delivery.Status},
		}
		if delivery.ExitCode != nil {
			fields = append(fields, toon.Field{Key: "exit_code", Value: *delivery.ExitCode})
		}
		if delivery.HTTPStatus != 0 {
			fields = append(fields, toon.Field{Key: "http_status", Value: delivery.HTTPStatus})
		}
		if delivery.Output != "" {
			fields = append(fields, toon.Field{Key: "output", Value: delivery.Output})
		}
		if delivery.Error != "" {
			fields = append(fields, toon.Field{Key: "error", Value: delivery.Error})
		}
		rows = append(rows, toon.NewObject(fields...))
	}
	return rows
}
