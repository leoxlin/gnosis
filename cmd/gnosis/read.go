package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"strings"
)

func newReadCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "read [gnosis-uri] [flags]",
		Short: "Print one exact vault document",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			uri := strings.TrimSpace(args[0])
			if !strings.HasPrefix(uri, "gnosis://") {
				return errors.New("read: argument must be a gnosis URI")
			}
			return runRead(vaultPath, uri, jsonOutput, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}

func runRead(vaultPath, uri string, jsonOutput bool, stdout io.Writer) error {
	page, err := vault.ReadPage(vaultPath, uri)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if jsonOutput {
		return writeJSON(stdout, page)
	}
	_, err = io.WriteString(stdout, page.Markdown)
	return err
}
