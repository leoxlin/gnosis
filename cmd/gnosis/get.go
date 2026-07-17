package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newGetCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "get",
		Short:   "Get vault resources",
		Args:    cobra.NoArgs,
		GroupID: "basic",
		Example: "gnosis get vaults\n" +
			"gnosis get concepts\n" +
			"gnosis get pages <gnosis-uri>",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("get: missing resource"))
		},
	}
	command.AddCommand(
		newGetVaultsCommand(options, stdout),
		newGetConceptsCommand(options, stdout),
		newGetPagesCommand(options, stdout),
		newGetProceduresCommand(options, stdout),
		newGetDirectivesCommand(options, stdout),
	)
	return command
}

func newGetVaultsCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "vaults [flags]",
		Short: "List effective vaults",
		Args:  cobra.NoArgs,
		Example: "gnosis get vaults\n" +
			"gnosis get vaults --fields vault,kind,precedence\n" +
			"gnosis --vault <path> get vaults",
		RunE: func(_ *cobra.Command, _ []string) error {
			selector, err := parseFields(
				fields,
				[]string{"vault", "kind", "root"},
				[]string{"vault", "kind", "root", "precedence"},
			)
			if err != nil {
				return newUsageError(err)
			}
			catalog, err := vault.Vaults(options.vaultPath)
			if err != nil {
				return fmt.Errorf("get vaults: %w", err)
			}
			rows := make([]toon.Object, 0, len(catalog.Vaults))
			for _, origin := range catalog.Vaults {
				rows = append(rows, selector.object(func(name string) (any, bool) {
					switch name {
					case "vault":
						return origin.Vault, true
					case "kind":
						return string(origin.Kind), true
					case "root":
						return origin.Root, true
					case "precedence":
						return origin.Precedence, true
					default:
						return nil, false
					}
				}))
			}
			return writeTOON(stdout, listOutput(
				"vaults",
				len(rows),
				rows,
				"0 effective vaults found in the current workspace",
				[]string{
					"Run `gnosis get concepts` to list available concept types",
					"Run `gnosis get pages` to list effective pages",
				},
			))
		},
	}
	command.Flags().StringVar(
		&fields,
		"fields",
		"",
		"comma-separated fields: vault, kind, root, precedence",
	)
	return command
}

func newGetConceptsCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "concepts [type] [flags]",
		Short: "List concept types or concepts of one exact type",
		Args:  cobra.MaximumNArgs(1),
		Example: "gnosis get concepts\n" +
			"gnosis get concepts Directive\n" +
			"gnosis get concepts Directive --fields uri,title,revision",
		RunE: func(_ *cobra.Command, args []string) error {
			conceptType := ""
			if len(args) == 1 {
				conceptType = strings.TrimSpace(args[0])
				if conceptType == "" {
					return newUsageError(errors.New("get concepts: type must not be empty"))
				}
			}
			defaults := []string{"type", "description", "uri"}
			if conceptType != "" {
				defaults = []string{"uri", "title", "type"}
			}
			selector, err := parseFields(
				fields,
				defaults,
				[]string{"uri", "type", "title", "description", "revision"},
			)
			if err != nil {
				return newUsageError(err)
			}
			catalog, err := vault.Concepts(options.vaultPath, conceptType)
			if err != nil {
				return fmt.Errorf("get concepts: %w", err)
			}
			if conceptType == "" {
				return writeConceptTypes(stdout, selector, catalog.ConceptTypes)
			}
			return writeConceptList(stdout, selector, conceptType, catalog.Concepts)
		},
	}
	command.Flags().StringVar(
		&fields,
		"fields",
		"",
		"comma-separated fields: uri, type, title, description, revision",
	)
	return command
}

func writeConceptTypes(
	stdout io.Writer,
	selector fieldSelector,
	conceptTypes []vault.ConceptTypeSummary,
) error {
	rows := make([]toon.Object, 0, len(conceptTypes))
	for _, conceptType := range conceptTypes {
		rows = append(rows, selector.object(func(name string) (any, bool) {
			switch name {
			case "type":
				return conceptType.Type, true
			case "description":
				return conceptType.Description, true
			case "uri":
				return conceptType.URI, true
			case "title", "revision":
				return "", true
			default:
				return nil, false
			}
		}))
	}
	return writeTOON(stdout, listOutput(
		"concept_types",
		len(rows),
		rows,
		"0 concept types found in the current vault",
		[]string{
			"Run `gnosis get concepts <type>` to list concepts of one type",
			"Run `gnosis get pages <uri>` to inspect one concept page",
		},
	))
}

func writeConceptList(
	stdout io.Writer,
	selector fieldSelector,
	conceptType string,
	concepts []vault.DocumentRef,
) error {
	rows := documentRows(selector, concepts)
	return writeTOON(stdout, listOutput(
		"concepts",
		len(rows),
		rows,
		fmt.Sprintf("0 concepts with type %q found in the current vault", conceptType),
		[]string{"Run `gnosis get pages <uri>` to inspect one concept page"},
	))
}

func newGetPagesCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	var isFull bool
	command := &cobra.Command{
		Use:   "pages [gnosis-uri] [flags]",
		Short: "List effective pages or read one exact page",
		Args:  cobra.MaximumNArgs(1),
		Example: "gnosis get pages\n" +
			"gnosis get pages --fields uri,title,revision\n" +
			"gnosis get pages <gnosis-uri> --full",
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 0 {
				if command.Flags().Changed("full") {
					return newUsageError(errors.New("get pages: --full requires a gnosis uri"))
				}
				return listPages(options.vaultPath, fields, stdout)
			}
			if command.Flags().Changed("fields") {
				return newUsageError(errors.New("get pages: --fields is available only when listing pages"))
			}
			uri := strings.TrimSpace(args[0])
			if !vault.IsCanonicalURI(uri) {
				return newUsageError(errors.New("get pages: argument must be a gnosis uri"))
			}
			return getPage(options.vaultPath, uri, isFull, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(
		&fields,
		"fields",
		"",
		"list fields: uri, type, title, description, revision",
	)
	flags.BoolVar(&isFull, "full", false, "include complete page Markdown")
	return command
}

func listPages(vaultPath, fields string, stdout io.Writer) error {
	selector, err := parseFields(
		fields,
		[]string{"uri", "title", "type"},
		[]string{"uri", "type", "title", "description", "revision"},
	)
	if err != nil {
		return newUsageError(err)
	}
	pages, err := vault.ListPages(vaultPath)
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}
	rows := documentRows(selector, pages)
	return writeTOON(stdout, listOutput(
		"pages",
		len(rows),
		rows,
		"0 pages found in the current vault",
		[]string{
			"Run `gnosis get pages <uri>` to inspect one page",
			"Run `gnosis search knowledge \"<question>\" --backend lexical` to search pages",
		},
	))
}

func getPage(vaultPath, uri string, isFull bool, stdout io.Writer) error {
	page, err := vault.ReadPage(vaultPath, uri)
	if err != nil {
		return fmt.Errorf("get pages: %w", err)
	}
	markdown, total, isTruncated := truncate(page.Markdown, isFull)
	fields := []toon.Field{
		{Key: "page", Value: toon.NewObject(
			toon.Field{Key: "document", Value: documentObject(page.Document)},
			toon.Field{Key: "markdown", Value: markdown},
		)},
		{Key: "content_chars", Value: total},
		{Key: "truncated", Value: isTruncated},
	}
	if isTruncated {
		fields = append(fields, toon.Field{
			Key:   "help",
			Value: []string{"Run `gnosis get pages <gnosis-uri> --full` for complete Markdown"},
		})
	}
	return writeTOON(stdout, toon.NewObject(fields...))
}

func newGetProceduresCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	var tags []string
	var isFull bool
	command := &cobra.Command{
		Use:   "procedures [gnosis-uri] [flags]",
		Short: "List executable procedures or read one execution contract",
		Args:  cobra.MaximumNArgs(1),
		Example: "gnosis get procedures --tags gnosis,development\n" +
			"gnosis get procedures --fields uri,title,tags\n" +
			"gnosis get procedures <gnosis-uri> --full",
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 0 {
				if command.Flags().Changed("full") {
					return newUsageError(errors.New("get procedures: --full requires a gnosis uri"))
				}
				return listProcedures(options.vaultPath, tags, fields, stdout)
			}
			if command.Flags().Changed("fields") || command.Flags().Changed("tags") {
				return newUsageError(errors.New(
					"get procedures: --fields and --tags are available only when listing procedures",
				))
			}
			uri := strings.TrimSpace(args[0])
			if !vault.IsCanonicalURI(uri) {
				return newUsageError(errors.New("get procedures: argument must be a gnosis uri"))
			}
			return getProcedure(options.vaultPath, uri, isFull, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(
		&fields,
		"fields",
		"",
		"list fields: uri, type, title, description, revision, invocation, tags",
	)
	flags.StringSliceVar(&tags, "tags", nil, "require all procedure tags")
	flags.BoolVar(&isFull, "full", false, "include the complete execution contract")
	return command
}

func listProcedures(vaultPath string, tags []string, fields string, stdout io.Writer) error {
	selector, err := parseFields(
		fields,
		[]string{"uri", "title", "description"},
		[]string{"uri", "type", "title", "description", "revision", "invocation", "tags"},
	)
	if err != nil {
		return newUsageError(err)
	}
	catalog, err := vault.DiscoverProcesses(vaultPath, tags)
	if err != nil {
		return fmt.Errorf("get procedures: %w", err)
	}
	records := catalog["procedures"]
	rows := make([]toon.Object, 0, len(records))
	for _, record := range records {
		rows = append(rows, selector.object(func(name string) (any, bool) {
			value, ok := record[name]
			if ok {
				return value, true
			}
			if name == "tags" {
				return []string{}, true
			}
			return "", true
		}))
	}
	return writeTOON(stdout, listOutput(
		"procedures",
		len(rows),
		rows,
		"0 executable procedures found for the requested tags",
		[]string{"Run `gnosis get procedures <uri> --full` to load one execution contract"},
	))
}

func getProcedure(vaultPath, uri string, isFull bool, stdout io.Writer) error {
	invocation, err := vault.InvokeProcess(vaultPath, uri)
	if err != nil {
		return fmt.Errorf("get procedures: %w", err)
	}
	procedure, total, isTruncated := procedureObject(invocation, isFull)
	fields := []toon.Field{
		{Key: "procedure", Value: procedure},
		{Key: "content_chars", Value: total},
		{Key: "truncated", Value: isTruncated},
	}
	if isTruncated {
		fields = append(fields, toon.Field{
			Key:   "help",
			Value: []string{"Run `gnosis get procedures <gnosis-uri> --full` for the complete contract"},
		})
	}
	return writeTOON(stdout, toon.NewObject(fields...))
}

func procedureObject(invocation vault.ProcessInvocation, isFull bool) (toon.Object, int, bool) {
	total := procedureCharacterCount(invocation)
	remaining := detailPreviewLimit
	clip := func(content string) string {
		if isFull {
			return content
		}
		runes := []rune(content)
		if len(runes) <= remaining {
			remaining -= len(runes)
			return content
		}
		if remaining == 0 {
			return ""
		}
		preview := string(runes[:remaining])
		remaining = 0
		return preview
	}
	sections := sectionsObject(invocation.Sections, clip)
	steps := make([]toon.Object, 0, len(invocation.Steps))
	for _, step := range invocation.Steps {
		steps = append(steps, toon.NewObject(
			toon.Field{Key: "number", Value: step.Number},
			toon.Field{Key: "name", Value: step.Name},
			toon.Field{Key: "sections", Value: sectionsObject(step.Sections, clip)},
		))
	}
	return toon.NewObject(
		toon.Field{Key: "process", Value: toon.NewObject(
			toon.Field{Key: "document", Value: documentObject(invocation.Process.DocumentRef)},
			toon.Field{Key: "invocation", Value: invocation.Process.Invocation},
			toon.Field{Key: "tags", Value: invocation.Process.Tags},
		)},
		toon.Field{Key: "sections", Value: sections},
		toon.Field{Key: "steps", Value: steps},
	), total, !isFull && total > detailPreviewLimit
}

func sectionsObject(
	sections vault.ProcessSections,
	transform func(string) string,
) toon.Object {
	return toon.NewObject(
		toon.Field{Key: "knowledge_inputs", Value: transform(sections.KnowledgeInputs)},
		toon.Field{Key: "process", Value: transform(sections.Process)},
		toon.Field{Key: "completion", Value: transform(sections.Completion)},
	)
}

func procedureCharacterCount(invocation vault.ProcessInvocation) int {
	total := utf8.RuneCountInString(invocation.Sections.KnowledgeInputs) +
		utf8.RuneCountInString(invocation.Sections.Process) +
		utf8.RuneCountInString(invocation.Sections.Completion)
	for _, step := range invocation.Steps {
		total += utf8.RuneCountInString(step.Sections.KnowledgeInputs)
		total += utf8.RuneCountInString(step.Sections.Process)
		total += utf8.RuneCountInString(step.Sections.Completion)
	}
	return total
}

func documentRows(selector fieldSelector, documents []vault.DocumentRef) []toon.Object {
	rows := make([]toon.Object, 0, len(documents))
	for _, document := range documents {
		rows = append(rows, selector.object(func(name string) (any, bool) {
			switch name {
			case "uri":
				return document.URI, true
			case "type":
				return document.Type, true
			case "title":
				return document.Title, true
			case "description":
				return document.Description, true
			case "revision":
				return document.Revision, true
			default:
				return nil, false
			}
		}))
	}
	return rows
}

func documentObject(document vault.DocumentRef) toon.Object {
	return toon.NewObject(
		toon.Field{Key: "uri", Value: document.URI},
		toon.Field{Key: "type", Value: document.Type},
		toon.Field{Key: "title", Value: document.Title},
		toon.Field{Key: "description", Value: document.Description},
		toon.Field{Key: "origin", Value: toon.NewObject(
			toon.Field{Key: "vault", Value: document.Origin.Vault},
			toon.Field{Key: "kind", Value: string(document.Origin.Kind)},
			toon.Field{Key: "root", Value: document.Origin.Root},
			toon.Field{Key: "path", Value: document.Origin.Path},
			toon.Field{Key: "precedence", Value: document.Origin.Precedence},
		)},
		toon.Field{Key: "revision", Value: document.Revision},
	)
}

func newGetDirectivesCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "directives [flags]",
		Short: "List directives with derived task progress",
		Args:  cobra.NoArgs,
		Example: "gnosis get directives\n" +
			"gnosis get directives --fields uri,status,tasks_done,tasks_total",
		RunE: func(_ *cobra.Command, _ []string) error {
			selector, err := parseFields(
				fields,
				[]string{"uri", "title", "status", "tasks_done", "tasks_total"},
				[]string{"uri", "title", "status", "tasks_done", "tasks_total", "revision"},
			)
			if err != nil {
				return newUsageError(err)
			}
			directives, err := vault.Directives(options.vaultPath)
			if err != nil {
				return fmt.Errorf("get directives: %w", err)
			}
			rows := make([]toon.Object, 0, len(directives))
			for _, directive := range directives {
				rows = append(rows, selector.object(func(name string) (any, bool) {
					switch name {
					case "uri":
						return directive.URI, true
					case "title":
						return directive.Title, true
					case "status":
						return directive.Status, true
					case "tasks_done":
						return directive.TasksDone, true
					case "tasks_total":
						return directive.TasksTotal, true
					case "revision":
						return directive.Revision, true
					default:
						return nil, false
					}
				}))
			}
			return writeTOON(stdout, listOutput(
				"directives",
				len(rows),
				rows,
				"0 directives found in the current vault",
				[]string{"Run `gnosis get pages <uri> --full` to read one directive"},
			))
		},
	}
	command.Flags().StringVar(
		&fields,
		"fields",
		"",
		"comma-separated fields: uri, title, status, tasks_done, tasks_total, revision",
	)
	return command
}
