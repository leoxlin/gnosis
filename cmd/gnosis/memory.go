package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	agentmemory "gnosis/internal/memory"
)

func newAddCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "add",
		Short:   "Add a vault resource",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis add memory \"<text>\"\n" +
			"gnosis --vault <path> add memory \"<text>\"",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("add: missing resource"))
		},
	}
	command.AddCommand(newAddMemoryCommand(options, stdout))
	return command
}

func newAddMemoryCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "memory <text>",
		Short: "Store one scoped durable memory",
		Args:  memoryArgs("add memory", "text"),
		Example: "gnosis add memory \"I prefer concise answers\"\n" +
			"gnosis --vault <path> add memory \"Use UTC timestamps\"",
		RunE: func(command *cobra.Command, args []string) error {
			return runAddMemory(command.Context(), options.vaultPath, args[0], stdout)
		},
	}
}

func newSearchMemoryCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var limit int
	command := &cobra.Command{
		Use:   "memory <query> [flags]",
		Short: "Search scoped durable memories",
		Args:  memoryArgs("search memory", "query"),
		Example: "gnosis search memory \"formatting preference\"\n" +
			"gnosis search memory \"deployment\" --limit 10",
		RunE: func(command *cobra.Command, args []string) error {
			if limit < 1 || limit > agentmemory.MaxSearchLimit {
				return newUsageError(fmt.Errorf(
					"search memory: --limit must be between 1 and %d",
					agentmemory.MaxSearchLimit,
				))
			}
			return runSearchMemory(command.Context(), options.vaultPath, args[0], limit, stdout)
		},
	}
	command.Flags().IntVar(
		&limit,
		"limit",
		agentmemory.DefaultSearchLimit,
		"maximum memories to return, from 1 through 20",
	)
	return command
}

func runAddMemory(ctx context.Context, vaultPath, text string, output io.Writer) error {
	service, err := agentmemory.NewFromEnv(vaultPath)
	if err != nil {
		return err
	}
	result, err := service.Add(ctx, text)
	if err != nil {
		return err
	}
	return writeMemoryResult(output, result)
}

func runSearchMemory(ctx context.Context, vaultPath, query string, limit int, output io.Writer) error {
	service, err := agentmemory.NewFromEnv(vaultPath)
	if err != nil {
		return err
	}
	result, err := service.Search(ctx, query, &limit)
	if err != nil {
		return err
	}
	return writeMemoryResult(output, result)
}

func writeMemoryResult(output io.Writer, result agentmemory.Result) error {
	rows := make([]toon.Object, 0, len(result.Memories))
	for _, record := range result.Memories {
		fields := []toon.Field{
			{Key: "id", Value: record.ID},
			{Key: "text", Value: record.Text},
			{Key: "backend", Value: record.Backend},
		}
		var score any
		if record.Score != nil {
			score = *record.Score
		}
		var origin any
		if record.Origin != nil {
			origin = toon.NewObject(
				toon.Field{Key: "vault", Value: record.Origin.Vault},
				toon.Field{Key: "kind", Value: string(record.Origin.Kind)},
				toon.Field{Key: "root", Value: record.Origin.Root},
				toon.Field{Key: "path", Value: record.Origin.Path},
				toon.Field{Key: "precedence", Value: record.Origin.Precedence},
			)
		}
		for _, optional := range []struct {
			key     string
			value   any
			include bool
		}{
			{key: "event", value: record.Event, include: record.Event != ""},
			{key: "score", value: score, include: record.Score != nil},
			{key: "metadata", value: record.Metadata, include: len(record.Metadata) > 0},
			{key: "created_at", value: record.CreatedAt, include: record.CreatedAt != ""},
			{key: "updated_at", value: record.UpdatedAt, include: record.UpdatedAt != ""},
			{key: "origin", value: origin, include: record.Origin != nil},
		} {
			if optional.include {
				fields = append(fields, toon.Field{Key: optional.key, Value: optional.value})
			}
		}
		rows = append(rows, toon.NewObject(fields...))
	}
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "count", Value: result.Count},
		toon.Field{Key: "memories", Value: rows},
	))
}

func memoryArgs(command, argument string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return newUsageError(fmt.Errorf("%s: missing %s", command, argument))
		}
		if len(args) > 1 {
			return newUsageError(fmt.Errorf(
				"%s: unexpected argument(s): %s", command, strings.Join(args[1:], " "),
			))
		}
		if strings.TrimSpace(args[0]) == "" {
			return newUsageError(fmt.Errorf("%s: %s must not be empty", command, argument))
		}
		return nil
	}
}
