package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const defaultVault = "."

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
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
	return command.Execute()
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
	command.AddCommand(newScaffoldCommand(stdout), newSetupCommand(stdout), newVaultsCommand(stdout), newIndexCommand(stdout), newReadCommand(stdout), newWriteCommand(os.Stdin, stdout), newValidateCommand(stdout, stderr), newQueryCommand(stdout), newConceptsCommand(stdout), newProcedureCommand(stdout), newGraphCommand(stdout))
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
