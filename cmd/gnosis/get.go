package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"gnosis/internal/vault"
)

func newGetCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Get vault resources",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newGetVaultsCommand(stdout), newGetConceptsCommand(stdout))
	return command
}

func newGetVaultsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "vaults [flags]",
		Short: "List effective vaults",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			catalog, err := vault.Vaults(vaultPath)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(stdout, catalog)
			}
			for _, entry := range catalog.Vaults {
				fmt.Fprintf(stdout, "Vault: %s\nKind: %s\n", entry.Vault, entry.Kind)
				if entry.Root != "" {
					fmt.Fprintf(stdout, "Root: %s\n", entry.Root)
				}
				fmt.Fprintln(stdout)
			}
			return nil
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}

func newGetConceptsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "concepts [type] [flags]",
		Short: "List concept types or concepts of an exact type",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			conceptType := ""
			if len(args) == 1 {
				conceptType = strings.TrimSpace(args[0])
			}
			if !jsonOutput {
				return vault.ListConcepts(vaultPath, conceptType, stdout)
			}
			if conceptType == "" {
				catalog, err := vault.Concepts(vaultPath, "")
				if err != nil {
					return err
				}
				return writeJSON(stdout, catalog)
			}
			catalog, err := vault.ConceptRecords(vaultPath, conceptType)
			if err != nil {
				return err
			}
			return writeJSON(stdout, catalog)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}
