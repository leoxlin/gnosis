package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

const defaultVault = "."

type rootOptions struct {
	vaultPath string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := runContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	if err == nil {
		return
	}
	if writeErr := writeCommandError(os.Stdout, err); writeErr != nil {
		fmt.Fprintln(os.Stderr, writeErr)
		os.Exit(1)
	}
	os.Exit(exitCode(err))
}

func run(args []string, stdout, stderr io.Writer) error {
	return runContext(context.Background(), args, stdout, stderr)
}

func runContext(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	command := newRootCommand(stdout, stderr)
	command.SetArgs(args)
	executed, err := command.ExecuteContextC(ctx)
	if executed == nil {
		executed = command
	}
	return wrapCommandError(executed, err)
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	options := &rootOptions{vaultPath: defaultVault}
	command := &cobra.Command{
		Use:           "gnosis",
		Short:         "Manage OKF-compatible knowledge in the current workspace",
		Args:          cobra.NoArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		Example: "gnosis\n" +
			"gnosis get pages\n" +
			"gnosis search knowledge \"<question>\" --backend lexical",
		RunE: func(_ *cobra.Command, _ []string) error {
			return writeHome(stdout, options)
		},
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return newUsageError(err)
	})
	command.PersistentFlags().StringVar(
		&options.vaultPath,
		"vault",
		defaultVault,
		"path to the OKF vault",
	)
	command.AddGroup(
		&cobra.Group{ID: "basic", Title: "Basic Commands"},
		&cobra.Group{ID: "knowledge", Title: "Knowledge Commands"},
		&cobra.Group{ID: "workspace", Title: "Workspace Commands"},
		&cobra.Group{ID: "other", Title: "Other Commands"},
	)
	command.AddCommand(
		newCreateCommand(options, stdout),
		newGetCommand(options, stdout),
		newApplyCommand(options, os.Stdin, stdout),
		newSearchCommand(options, stdout),
		newGraphCommand(options, stdout),
		newIndexCommand(options, stdout),
		newValidateCommand(options, stdout, stderr),
		newServeCommand(options),
	)
	command.AddCommand(&cobra.Command{
		Use:     "version",
		Short:   "Print the gnosis version",
		Args:    cobra.NoArgs,
		GroupID: "other",
		Example: "gnosis version\n" +
			"gnosis version --help",
		RunE: func(_ *cobra.Command, _ []string) error {
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "version", Value: "0.1.0"},
			))
		},
	})
	configureCompletion(command)
	setCommandHelp(command)
	return command
}

func configureCompletion(root *cobra.Command) {
	root.InitDefaultCompletionCmd()
	completion, _, err := root.Find([]string{"completion"})
	if err != nil || completion == nil {
		return
	}
	completion.GroupID = "other"
	completion.Example = "gnosis completion bash\n" +
		"gnosis completion zsh"
	for _, shell := range completion.Commands() {
		shell.Example = "gnosis completion " + shell.Name() + "\n" +
			"gnosis completion " + shell.Name() + " --help"
	}
}

func writeHome(output io.Writer, options *rootOptions) error {
	vaultCatalog, err := vault.Vaults(options.vaultPath)
	if err != nil {
		return fmt.Errorf("home: list vaults: %w", err)
	}
	conceptCatalog, err := vault.Concepts(options.vaultPath, "")
	if err != nil {
		return fmt.Errorf("home: list concept types: %w", err)
	}
	pages, err := vault.ListPages(options.vaultPath)
	if err != nil {
		return fmt.Errorf("home: list pages: %w", err)
	}

	vaultRows := make([]toon.Object, 0, len(vaultCatalog.Vaults))
	for _, origin := range vaultCatalog.Vaults {
		vaultRows = append(vaultRows, toon.NewObject(
			toon.Field{Key: "vault", Value: origin.Vault},
			toon.Field{Key: "kind", Value: string(origin.Kind)},
			toon.Field{Key: "root", Value: origin.Root},
		))
	}
	typeRows := make([]toon.Object, 0, len(conceptCatalog.ConceptTypes))
	for _, conceptType := range conceptCatalog.ConceptTypes {
		typeRows = append(typeRows, toon.NewObject(
			toon.Field{Key: "type", Value: conceptType.Type},
			toon.Field{Key: "description", Value: conceptType.Description},
		))
	}
	workspace, err := filepath.Abs(options.vaultPath)
	if err != nil {
		workspace = filepath.Clean(options.vaultPath)
	}
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "bin", Value: executablePath()},
		toon.Field{
			Key:   "description",
			Value: "Manage OKF-compatible knowledge in the current workspace",
		},
		toon.Field{Key: "workspace", Value: workspace},
		toon.Field{Key: "counts", Value: toon.NewObject(
			toon.Field{Key: "vaults", Value: len(vaultRows)},
			toon.Field{Key: "pages", Value: len(pages)},
			toon.Field{Key: "concept_types", Value: len(typeRows)},
		)},
		toon.Field{Key: "vaults", Value: vaultRows},
		toon.Field{Key: "concept_types", Value: typeRows},
		toon.Field{Key: "help", Value: []string{
			"Run `gnosis get pages` to list available pages",
			"Run `gnosis search knowledge \"<question>\" --backend lexical` to search live knowledge",
			"Run `gnosis --help` for the command reference",
		}},
	))
}
