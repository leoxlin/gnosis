package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"gnosis/internal/vault"
)

func newVaultsCommand(stdout io.Writer) *cobra.Command {
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
