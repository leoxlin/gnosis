package main

import (
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"strings"
)

func newConceptsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, conceptType string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "concepts [flags]",
		Short: "List concept types or concepts of an exact type",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if jsonOutput {
				typeName := strings.TrimSpace(conceptType)
				if typeName == "" {
					catalog, err := vault.Concepts(vaultPath, "")
					if err != nil {
						return err
					}
					return writeJSON(stdout, catalog)
				}
				catalog, err := vault.ConceptRecords(vaultPath, typeName)
				if err != nil {
					return err
				}
				return writeJSON(stdout, catalog)
			}
			return vault.ListConcepts(vaultPath, conceptType, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&conceptType, "type", "", "exact concept type")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}
