package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gnosis/internal/vault"
)

func newSearchCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "search",
		Short: "Search vault resources",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newSearchKnowledgeCommand(stdout))
	return command
}

func newSearchKnowledgeCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, backend string
	var top, maxRead, depth int
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "knowledge [flags] <question>",
		Short: "Find relevant vault pages for a question",
		Args:  questionArgs("search knowledge"),
		RunE: func(command *cobra.Command, args []string) error {
			if err := validateQueryOptions(top, maxRead, depth); err != nil {
				return fmt.Errorf("search knowledge: %w", err)
			}
			options := vault.QueryOptions{Top: top, MaxRead: maxRead, MaxDepth: depth}
			var result vault.QueryResult
			var err error
			switch backend {
			case "vector":
				config, configErr := vault.SemanticConfigFromEnv(vaultPath)
				if configErr != nil {
					return configErr
				}
				result, err = vault.QuerySemanticKnowledge(command.Context(), vaultPath, args[0], options, config)
			case "lexical":
				result, err = vault.QueryKnowledge(vaultPath, args[0], options)
			default:
				return fmt.Errorf("search knowledge: unknown backend %q", backend)
			}
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(stdout, result)
			}
			writeQueryText(stdout, result)
			return nil
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&backend, "backend", "vector", "retrieval backend: vector or lexical")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
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

func writeQueryText(output io.Writer, result vault.QueryResult) {
	fmt.Fprintf(output, "answer_type: %s\n", result.AnswerType)
	fmt.Fprintf(output, "index_only: %t\n", result.IndexOnly)
	if len(result.Candidates) == 0 {
		fmt.Fprintln(output, "no matches")
		return
	}
	fmt.Fprintln(output, "candidates:")
	for _, candidate := range result.Candidates {
		fmt.Fprintf(output, "- %s (%s)", candidate.Title, candidate.URI)
		if candidate.Description != "" {
			fmt.Fprintf(output, " - %s", candidate.Description)
		}
		fmt.Fprintln(output)
	}
	if len(result.Path) > 0 {
		fmt.Fprintln(output, "path:")
		fmt.Fprintln(output, strings.Join(result.Path, " -> "))
	}
	if len(result.ShouldRead) > 0 {
		fmt.Fprintln(output, "should_read:")
		for _, page := range result.ShouldRead {
			fmt.Fprintf(output, "- %s\n", page)
		}
	}
}

func questionArgs(command string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("%s: missing question", command)
		}
		if len(args) > 1 {
			return fmt.Errorf("%s: unexpected argument(s): %s", command, strings.Join(args[1:], " "))
		}
		if strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("%s: question must not be empty", command)
		}
		return nil
	}
}
