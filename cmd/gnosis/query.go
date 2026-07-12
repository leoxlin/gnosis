package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"strings"
)

func newQueryCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "query",
		Short: "Query the vault",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newSearchQueryCommand(stdout), newGraphQueryCommand(stdout))
	return command
}

func newSearchQueryCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var top, maxRead, depth int
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "search [flags] <question>",
		Short: "Find relevant vault pages for a question",
		Args:  questionArgs("query search"),
		RunE: func(_ *cobra.Command, args []string) error {
			return runQuery(vaultPath, top, maxRead, depth, jsonOutput, args[0], stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}

func newGraphQueryCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var top, maxRead, depth int
	command := &cobra.Command{
		Use:   "graph [flags] <question>",
		Short: "Query the vault and emit graph-aware JSON",
		Args:  questionArgs("query graph"),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGraphQuery(vaultPath, top, maxRead, depth, args[0], stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	return command
}

func runQuery(vaultPath string, top, maxRead, depth int, jsonOutput bool, question string, stdout io.Writer) error {
	if err := validateQueryOptions(top, maxRead, depth); err != nil {
		return fmt.Errorf("query search: %w", err)
	}

	result, err := vault.QueryKnowledge(vaultPath, question, vault.QueryOptions{
		Top:      top,
		MaxRead:  maxRead,
		MaxDepth: depth,
	})
	if err != nil {
		return err
	}
	if jsonOutput {
		return writeJSON(stdout, result)
	}
	writeQueryText(stdout, result)
	return nil
}

func runGraphQuery(vaultPath string, top, maxRead, depth int, question string, stdout io.Writer) error {
	if err := validateQueryOptions(top, maxRead, depth); err != nil {
		return fmt.Errorf("query graph: %w", err)
	}

	result, err := vault.QueryKnowledge(vaultPath, question, vault.QueryOptions{
		Top:      top,
		MaxRead:  maxRead,
		MaxDepth: depth,
	})
	if err != nil {
		return err
	}
	return writeJSON(stdout, result)
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
