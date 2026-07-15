package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

const defaultVault = "."

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runContext(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	return runContext(context.Background(), args, stdout, stderr)
}

func runContext(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		command := newRootCommand(stdout, stderr)
		command.SetOut(stderr)
		if err := command.Usage(); err != nil {
			return err
		}
		return errors.New("missing command")
	}

	command := newRootCommand(stdout, stderr)
	command.SetArgs(args)
	return command.ExecuteContext(ctx)
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:           "gnosis",
		Short:         "Manage an OKF-compatible Obsidian vault",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.AddCommand(
		newScaffoldCommand(stdout),
		newSetupCommand(stdout),
		newGetCommand(stdout),
		newIndexCommand(stdout),
		newReadCommand(stdout),
		newWriteCommand(os.Stdin, stdout),
		newValidateCommand(stdout, stderr),
		newSearchCommand(stdout),
		newServeCommand(),
		newProcedureCommand(stdout),
		newGraphCommand(stdout),
	)
	command.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the gnosis version",
		Args:  cobra.NoArgs,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(stdout, "gnosis 0.1.0")
		},
	})
	return command
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
