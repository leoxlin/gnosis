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

func newSearchCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "search",
		Short:   "Search vault resources",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis search knowledge \"<question>\" --backend lexical\n" +
			"gnosis search knowledge \"<question>\" --fields uri,title,score",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("search: missing resource"))
		},
	}
	command.AddCommand(newSearchKnowledgeCommand(options, stdout))
	return command
}

func newSearchKnowledgeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var backend, fields string
	var top, maxRead, depth int
	command := &cobra.Command{
		Use:   "knowledge <question> [flags]",
		Short: "Find relevant vault pages for a question",
		Args:  questionArgs("search knowledge"),
		Example: "gnosis search knowledge \"<question>\" --backend lexical\n" +
			"gnosis search knowledge \"<question>\" --fields uri,title,score",
		RunE: func(command *cobra.Command, args []string) error {
			if err := validateQueryOptions(top, maxRead, depth); err != nil {
				return newUsageError(fmt.Errorf("search knowledge: %w", err))
			}
			selector, err := parseFields(
				fields,
				[]string{"uri", "title", "type", "score"},
				[]string{"uri", "type", "title", "description", "revision", "score"},
			)
			if err != nil {
				return newUsageError(err)
			}
			queryOptions := vault.QueryOptions{Top: top, MaxRead: maxRead, MaxDepth: depth}
			var result vault.QueryResult
			switch backend {
			case "vector":
				config, configErr := vault.SemanticConfigFromEnv(options.vaultPath)
				if configErr != nil {
					return configErr
				}
				result, err = vault.QuerySemanticKnowledge(
					command.Context(), options.vaultPath, args[0], queryOptions, config,
				)
			case "lexical":
				result, err = vault.QueryKnowledge(options.vaultPath, args[0], queryOptions)
			default:
				return newUsageError(fmt.Errorf("search knowledge: unknown backend %q", backend))
			}
			if err != nil {
				return fmt.Errorf("search knowledge: %w", err)
			}
			return writeSearchResult(stdout, selector, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&backend, "backend", "vector", "retrieval backend: vector or lexical")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	flags.StringVar(
		&fields,
		"fields",
		"",
		"candidate fields: uri, type, title, description, revision, score",
	)
	return command
}

func validateQueryOptions(top, maxRead, depth int) error {
	if top <= 0 {
		return errors.New("--top must be greater than zero")
	}
	if maxRead < 0 {
		return errors.New("--max-read must be zero or greater")
	}
	if depth <= 0 {
		return errors.New("--depth must be greater than zero")
	}
	return nil
}

func writeSearchResult(output io.Writer, selector fieldSelector, result vault.QueryResult) error {
	rows := make([]toon.Object, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		rows = append(rows, selector.object(func(name string) (any, bool) {
			switch name {
			case "uri":
				return candidate.URI, true
			case "type":
				return candidate.Type, true
			case "title":
				return candidate.Title, true
			case "description":
				return candidate.Description, true
			case "revision":
				return candidate.Revision, true
			case "score":
				return candidate.Score, true
			default:
				return nil, false
			}
		}))
	}
	fields := []toon.Field{
		{Key: "answer_type", Value: string(result.AnswerType)},
		{Key: "count", Value: len(rows)},
		{Key: "candidates", Value: rows},
		{Key: "path", Value: result.Path},
		{Key: "should_read", Value: result.ShouldRead},
		{Key: "index_only", Value: result.IndexOnly},
	}
	if len(rows) == 0 {
		fields = append(fields,
			toon.Field{Key: "message", Value: "0 matching pages found"},
			toon.Field{Key: "help", Value: []string{
				"Try broader terms or `--backend lexical` for exact local matching",
			}},
		)
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func questionArgs(command string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return newUsageError(fmt.Errorf("%s: missing question", command))
		}
		if len(args) > 1 {
			return newUsageError(fmt.Errorf(
				"%s: unexpected argument(s): %s", command, strings.Join(args[1:], " "),
			))
		}
		if strings.TrimSpace(args[0]) == "" {
			return newUsageError(fmt.Errorf("%s: question must not be empty", command))
		}
		return nil
	}
}
