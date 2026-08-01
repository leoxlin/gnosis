package main

import (
	"fmt"
	"io"
	"slices"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/codeintel"
	"gnosis/internal/codeintel/languagepack"
)

func newParsersCommand(options *rootOptions, stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "parsers",
		Short:   "Manage explicitly installed code parsers",
		Args:    cobra.NoArgs,
		GroupID: "workspace",
		Example: "gnosis parsers list\n" +
			"gnosis parsers install go typescript --scope <scope>\n" +
			"gnosis parsers status go typescript",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(fmt.Errorf("parsers: missing action"))
		},
	}
	command.AddCommand(newParsersListCommand(stdout), newParsersInstallCommand(options, stdout, stderr), newParsersStatusCommand(stdout))
	return command
}

func newParsersListCommand(stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "list",
		Short: "List languages in the pinned parser catalog",
		Args:  cobra.NoArgs,
		Example: "gnosis parsers list\n" +
			"gnosis parsers list --fields language,installed,release,abi",
		RunE: func(_ *cobra.Command, _ []string) error {
			selector, err := parseFields(fields, []string{"language", "installed"}, []string{"language", "installed", "release", "platform", "abi", "library_digest"})
			if err != nil {
				return newUsageError(err)
			}
			cache, err := languagepack.DefaultCacheDir()
			if err != nil {
				return err
			}
			languages, err := languagepack.Catalog(cache)
			if err != nil {
				return fmt.Errorf("list parser catalog: %w", err)
			}
			statuses, err := languagepack.Status(cache, languages)
			if err != nil {
				return fmt.Errorf("verify parser catalog status: %w", err)
			}
			return writeParserStatuses(stdout, selector, statuses, "no parser languages are available")
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "fields: language, installed, release, platform, abi, library_digest")
	return command
}

func newParsersInstallCommand(options *rootOptions, stdout, stderr io.Writer) *cobra.Command {
	var scopeName string
	command := &cobra.Command{
		Use:     "install <language>... --scope <scope>",
		Short:   "Install verified native parsers for a configured code scope",
		Args:    cobra.MinimumNArgs(1),
		Example: "gnosis parsers install go typescript --scope app",
		RunE: func(command *cobra.Command, languages []string) error {
			if scopeName == "" {
				return newUsageError(fmt.Errorf("parsers install: --scope is required"))
			}
			scope, err := codeintel.ResolveScope(options.vaultPath, scopeName)
			if err != nil {
				return err
			}
			for _, language := range languages {
				if !slices.Contains(scope.Languages, language) {
					return newUsageError(fmt.Errorf("language %q is not allowed by code scope %q", language, scopeName))
				}
			}
			cache, err := languagepack.DefaultCacheDir()
			if err != nil {
				return err
			}
			fmt.Fprintf(stderr, "Installing verified parsers for %s...\n", scopeName)
			manifest, changed, err := languagepack.Install(command.Context(), cache, languages)
			if err != nil {
				return fmt.Errorf("install parsers: %w", err)
			}
			status := "installed"
			if !changed {
				status = "already installed"
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "action", Value: "install"}, toon.Field{Key: "resource", Value: "parsers"},
				toon.Field{Key: "status", Value: status}, toon.Field{Key: "changed", Value: changed},
				toon.Field{Key: "languages", Value: languages}, toon.Field{Key: "release", Value: manifest.PackRelease},
				toon.Field{Key: "platform", Value: manifest.Platform}, toon.Field{Key: "abi", Value: manifest.ABI},
			))
		},
	}
	command.Flags().StringVar(&scopeName, "scope", "", "configured code scope")
	return command
}

func newParsersStatusCommand(stdout io.Writer) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "status [language]...",
		Short: "Verify installed parser state",
		Args:  cobra.ArbitraryArgs,
		Example: "gnosis parsers status\n" +
			"gnosis parsers status go typescript --fields language,installed,library_digest",
		RunE: func(_ *cobra.Command, languages []string) error {
			selector, err := parseFields(fields, []string{"language", "installed"}, []string{"language", "installed", "release", "platform", "abi", "library_digest"})
			if err != nil {
				return newUsageError(err)
			}
			cache, err := languagepack.DefaultCacheDir()
			if err != nil {
				return err
			}
			statuses, err := languagepack.Status(cache, languages)
			if err != nil {
				return fmt.Errorf("verify parser status: %w", err)
			}
			return writeParserStatuses(stdout, selector, statuses, "no parsers are installed")
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "fields: language, installed, release, platform, abi, library_digest")
	return command
}

func writeParserStatuses(output io.Writer, selector fieldSelector, statuses []languagepack.ParserStatus, empty string) error {
	rows := make([]toon.Object, 0, len(statuses))
	for _, status := range statuses {
		current := status
		rows = append(rows, selector.object(func(name string) (any, bool) {
			switch name {
			case "language":
				return current.Language, true
			case "installed":
				return current.Installed, true
			case "release":
				return current.PackRelease, true
			case "platform":
				return current.Platform, true
			case "abi":
				return current.ABI, true
			case "library_digest":
				return current.LibraryDigest, true
			default:
				return nil, false
			}
		}))
	}
	fields := []toon.Field{
		{Key: "count", Value: len(rows)}, {Key: "total", Value: len(statuses)}, {Key: "parsers", Value: rows},
	}
	if len(rows) == 0 {
		fields = append(fields, toon.Field{Key: "message", Value: empty})
	}
	return writeTOON(output, toon.NewObject(fields...))
}
