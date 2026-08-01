package main

import (
	"errors"
	"io"
	"time"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	githubsource "gnosis/internal/github"
)

func newIngestCommand(options *rootOptions, output io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "ingest",
		Short:   "Ingest configured source evidence",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("ingest: missing source"))
		},
	}
	command.AddCommand(newIngestGitHubCommand(options, output))
	return command
}

func newIngestGitHubCommand(options *rootOptions, output io.Writer) *cobra.Command {
	var since string
	var maxItems int
	var reconcile bool
	command := &cobra.Command{
		Use:   "github <owner/repository>",
		Short: "Synchronize one configured GitHub repository into immutable evidence",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis ingest github owner/repository\n" +
			"gnosis --vault <name> ingest github owner/repository --since 2026-07-01T00:00:00Z\n" +
			"gnosis ingest github owner/repository --reconcile",
		RunE: func(command *cobra.Command, args []string) error {
			var sinceTime time.Time
			if since != "" {
				parsed, err := time.Parse(time.RFC3339, since)
				if err != nil {
					return newUsageError(err)
				}
				sinceTime = parsed
			}
			if maxItems < 0 {
				return newUsageError(errors.New("--max-items must not be negative"))
			}
			client, _, err := githubsource.NewConfigured(options.vaultPath, args[0])
			if err != nil {
				return err
			}
			result, err := client.Sync(command.Context(), githubsource.Options{
				Since: sinceTime, MaxItems: maxItems, Reconcile: reconcile,
			})
			if err != nil {
				return err
			}
			return writeGitHubResult(output, args[0], result)
		},
	}
	command.Flags().StringVar(&since, "since", "", "oldest upstream timestamp to retain (RFC3339)")
	command.Flags().IntVar(&maxItems, "max-items", 0, "maximum objects to process (0 uses configured page bounds)")
	command.Flags().BoolVar(&reconcile, "reconcile", false, "complete listing and append tombstones for confirmed absences")
	return command
}

func writeGitHubResult(output io.Writer, repository string, result githubsource.Result) error {
	status := "synchronized"
	if result.RateLimited {
		status = "rate-limited"
	}
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "action", Value: "ingest"},
		toon.Field{Key: "source", Value: "github"},
		toon.Field{Key: "repository", Value: repository},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "created", Value: result.Created},
		toon.Field{Key: "unchanged", Value: result.Unchanged},
		toon.Field{Key: "tombstoned", Value: result.Tombstoned},
		toon.Field{Key: "rejected", Value: result.Rejected},
		toon.Field{Key: "rate_limited", Value: result.RateLimited},
		toon.Field{Key: "rate_reset", Value: result.RateReset},
		toon.Field{Key: "cursor", Value: result.Cursor.Value},
	))
}
