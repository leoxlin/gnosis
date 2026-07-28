package main

import (
	"errors"
	"io"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newGetHistoryCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var cursor string
	var limit int
	command := &cobra.Command{
		Use:   "history <gnosis-uri> [flags]",
		Short: "Read bounded committed and working page history",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis get history gnosis://team/notes/example.md\n" +
			"gnosis get history gnosis://team/notes/example.md --limit 10\n" +
			"gnosis get history gnosis://team/notes/example.md --cursor <cursor>",
		RunE: func(_ *cobra.Command, args []string) error {
			result, err := vault.ReadPageHistory(options.vaultPath, args[0], cursor, limit)
			if err != nil {
				return err
			}
			return writeHistoryTOON(stdout, result)
		},
	}
	command.Flags().StringVar(&cursor, "cursor", "", "opaque continuation cursor")
	command.Flags().IntVar(&limit, "limit", vault.DefaultHistoryLimit, "maximum committed entries")
	return command
}

func newGetDiffCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var fromRevision string
	var toRevision string
	var limit int
	command := &cobra.Command{
		Use:   "diff <gnosis-uri> --from <revision> --to <revision> [flags]",
		Short: "Diff two exact revisions of one page",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis get diff gnosis://team/notes/example.md --from sha256:<old> --to sha256:<new>\n" +
			"gnosis get diff gnosis://team/notes/example.md --from sha256:<old> --to sha256:<new> --limit 5000",
		RunE: func(_ *cobra.Command, args []string) error {
			if fromRevision == "" || toRevision == "" {
				return newUsageError(errors.New("get diff: --from and --to are required"))
			}
			result, err := vault.DiffPage(
				options.vaultPath, args[0], fromRevision, toRevision, limit,
			)
			if err != nil {
				return err
			}
			return writeDiffTOON(stdout, result)
		},
	}
	command.Flags().StringVar(&fromRevision, "from", "", "exact earlier content revision")
	command.Flags().StringVar(&toRevision, "to", "", "exact later content revision")
	command.Flags().IntVar(&limit, "limit", vault.DefaultDiffLimit, "maximum diff characters")
	return command
}

func newGetChangesCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var cursor string
	var limit int
	command := &cobra.Command{
		Use:   "changes [flags]",
		Short: "Read committed effective-vault changes after a cursor",
		Args:  cobra.NoArgs,
		Example: "gnosis get changes\n" +
			"gnosis get changes --cursor <cursor>\n" +
			"gnosis get changes --cursor <cursor> --limit 10",
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := vault.ChangesSince(options.vaultPath, cursor, limit)
			if err != nil {
				return err
			}
			return writeChangesTOON(stdout, result)
		},
	}
	command.Flags().StringVar(&cursor, "cursor", "", "opaque committed effective-view cursor")
	command.Flags().IntVar(&limit, "limit", vault.DefaultHistoryLimit, "maximum changes")
	return command
}

func writeHistoryTOON(output io.Writer, result vault.PageHistoryResult) error {
	rows := make([]toon.Object, 0, len(result.Entries))
	for _, entry := range result.Entries {
		fields := []toon.Field{
			{Key: "revision", Value: entry.Revision},
			{Key: "classification", Value: string(entry.Classification)},
			{Key: "working", Value: entry.Working},
			{Key: "origin", Value: historyOriginTOON(entry.Origin)},
		}
		if entry.Commit != "" {
			fields = append(fields, toon.Field{Key: "commit", Value: entry.Commit})
		}
		if entry.Timestamp != "" {
			fields = append(fields, toon.Field{Key: "timestamp", Value: entry.Timestamp})
		}
		if entry.Actor != "" {
			fields = append(fields, toon.Field{Key: "actor", Value: entry.Actor})
		}
		rows = append(rows, toon.NewObject(fields...))
	}
	fields := []toon.Field{
		{Key: "uri", Value: result.URI},
		{Key: "current", Value: result.Current},
		{Key: "count", Value: len(rows)},
		{Key: "entries", Value: rows},
		{Key: "bound", Value: historyBoundTOON(result.Bound)},
	}
	if result.NextCursor != "" {
		fields = append(fields,
			toon.Field{Key: "next_cursor", Value: result.NextCursor},
			toon.Field{Key: "help", Value: []string{
				"Pass next_cursor to `gnosis get history <uri> --cursor <cursor>`.",
			}},
		)
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func writeDiffTOON(output io.Writer, result vault.PageDiffResult) error {
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "uri", Value: result.URI},
		toon.Field{Key: "from_revision", Value: result.FromRevision},
		toon.Field{Key: "to_revision", Value: result.ToRevision},
		toon.Field{Key: "classification", Value: string(result.Classification)},
		toon.Field{Key: "characters", Value: result.Characters},
		toon.Field{Key: "bound", Value: historyBoundTOON(result.Bound)},
		toon.Field{Key: "diff", Value: result.Diff},
	))
}

func writeChangesTOON(output io.Writer, result vault.ChangeFeedResult) error {
	rows := make([]toon.Object, 0, len(result.Changes))
	for _, change := range result.Changes {
		fields := []toon.Field{
			{Key: "uri", Value: change.URI},
			{Key: "classification", Value: string(change.Classification)},
		}
		if change.PreviousURI != "" {
			fields = append(fields, toon.Field{Key: "previous_uri", Value: change.PreviousURI})
		}
		if change.BeforeRevision != "" {
			fields = append(fields, toon.Field{Key: "before_revision", Value: change.BeforeRevision})
		}
		if change.AfterRevision != "" {
			fields = append(fields, toon.Field{Key: "after_revision", Value: change.AfterRevision})
		}
		rows = append(rows, toon.NewObject(fields...))
	}
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "count", Value: len(rows)},
		toon.Field{Key: "changes", Value: rows},
		toon.Field{Key: "next_cursor", Value: result.NextCursor},
		toon.Field{Key: "bound", Value: historyBoundTOON(result.Bound)},
		toon.Field{Key: "help", Value: []string{
			"Save next_cursor and pass it to `gnosis get changes --cursor <cursor>`.",
		}},
	))
}

func historyOriginTOON(origin vault.Origin) toon.Object {
	return toon.NewObject(
		toon.Field{Key: "vault", Value: origin.Vault},
		toon.Field{Key: "kind", Value: string(origin.Kind)},
		toon.Field{Key: "root", Value: origin.Root},
		toon.Field{Key: "path", Value: origin.Path},
		toon.Field{Key: "precedence", Value: origin.Precedence},
	)
}

func historyBoundTOON(bound vault.ResultBound) toon.Object {
	return toon.NewObject(
		toon.Field{Key: "limit", Value: bound.Limit},
		toon.Field{Key: "truncated", Value: bound.Truncated},
	)
}
