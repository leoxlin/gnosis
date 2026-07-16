package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newValidateCommand(options *rootOptions, stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "validate",
		Short:   "Validate resources",
		Args:    cobra.NoArgs,
		GroupID: "workspace",
		Example: "gnosis validate vault\n" +
			"gnosis --vault <path> validate vault",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("validate: missing resource"))
		},
	}
	command.AddCommand(newValidateVaultCommand(options, stdout, stderr))
	return command
}

func newValidateVaultCommand(
	options *rootOptions,
	stdout io.Writer,
	stderr io.Writer,
) *cobra.Command {
	return &cobra.Command{
		Use:   "vault",
		Short: "Validate vault structure and links",
		Args:  cobra.NoArgs,
		Example: "gnosis validate vault\n" +
			"gnosis --vault <path> validate vault",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runValidate(options.vaultPath, stdout, stderr)
		},
	}
}

func runValidate(vaultPath string, stdout, stderr io.Writer) error {
	result, err := vault.Validate(vaultPath)
	if err != nil {
		return fmt.Errorf("validate vault: %w", err)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf(
			"validate vault: %d error(s): %s",
			len(result.Errors),
			strings.Join(result.Errors, "; "),
		)
	}
	return writeTOON(stdout, toon.NewObject(
		toon.Field{Key: "resource", Value: "vault"},
		toon.Field{Key: "status", Value: "valid"},
		toon.Field{Key: "files_checked", Value: result.FilesChecked},
		toon.Field{Key: "warnings", Value: len(result.Warnings)},
	))
}
