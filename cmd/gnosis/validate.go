package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
)

func newValidateCommand(stdout, stderr io.Writer) *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "validate [flags]",
		Short: "Validate vault structure and links",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runValidate(vaultPath, stdout, stderr)
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}

func runValidate(vaultPath string, stdout, stderr io.Writer) error {
	result, err := vault.Validate(vaultPath)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	if len(result.Errors) > 0 {
		for _, validationErr := range result.Errors {
			fmt.Fprintf(stderr, "error: %s\n", validationErr)
		}
		return fmt.Errorf("validation failed: %d error(s)", len(result.Errors))
	}
	fmt.Fprintf(stdout, "ok: %d markdown file(s) validated\n", result.FilesChecked)
	return nil
}
