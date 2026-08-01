package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newCreateCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "create",
		Short:   "Create vault resources",
		Args:    cobra.NoArgs,
		GroupID: "basic",
		Example: "mkdir <directory> && cd <directory>\n" +
			"gnosis create vault --name <name> --concepts",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("create: missing resource"))
		},
	}
	command.AddCommand(newCreateVaultCommand(options, stdout))
	return command
}

func newCreateVaultCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var vaultName string
	var isForce, hasConcepts bool
	command := &cobra.Command{
		Use:   "vault [flags]",
		Short: "Create an OKF-compatible gnosis vault",
		Args:  cobra.NoArgs,
		Example: "mkdir <directory> && cd <directory>\n" +
			"gnosis create vault --name <name>\n" +
			"gnosis create vault --name <name> --concepts",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCreateVault(
				options.vaultPath,
				vaultName,
				isForce,
				hasConcepts,
				stdout,
			)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultName, "name", "", "name for the new vault")
	flags.BoolVar(&isForce, "force", false, "overwrite existing files")
	flags.BoolVar(&hasConcepts, "concepts", false, "include project concept definitions")
	return command
}

func runCreateVault(
	vaultPath string,
	vaultName string,
	isForce bool,
	hasConcepts bool,
	stdout io.Writer,
) error {
	created, err := vault.Scaffold(vaultPath, vault.ScaffoldOptions{
		Force:    isForce,
		Name:     vaultName,
		Concepts: hasConcepts,
	})
	if err != nil {
		return fmt.Errorf("create vault: %w", err)
	}
	status := "created"
	if len(created) == 0 {
		status = "no-op"
	}
	return writeTOON(stdout, toon.NewObject(
		toon.Field{Key: "action", Value: "create"},
		toon.Field{Key: "resource", Value: "vault"},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "path", Value: vaultPath},
		toon.Field{Key: "changed", Value: len(created) > 0},
		toon.Field{Key: "files", Value: created},
	))
}
