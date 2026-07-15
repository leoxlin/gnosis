package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"gnosis/internal/vault"
)

func newIndexCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "index",
		Short: "Index vault resources",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("index: missing resource")
		},
	}
	command.AddCommand(newIndexVaultCommand(stdout), newIndexKnowledgeCommand(stdout))
	return command
}

func newIndexVaultCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "vault [flags]",
		Short: "Generate vault indexes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runIndex(vaultPath, stdout)
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}

func newIndexKnowledgeCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var jsonOutput bool
	command := &cobra.Command{
		Use:   "knowledge [flags]",
		Short: "Synchronize the semantic knowledge index",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			config, err := vault.SemanticConfigFromEnv(vaultPath)
			if err != nil {
				return err
			}
			result, err := vault.SyncSemanticIndex(command.Context(), vaultPath, config)
			if err != nil {
				return err
			}
			return writeSemanticIndexResult(stdout, result, jsonOutput)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.BoolVar(&jsonOutput, "json", false, "emit indented machine-readable JSON")
	return command
}

func writeSemanticIndexResult(output io.Writer, result vault.SemanticIndexResult, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(output, result)
	}
	fmt.Fprintf(output, "documents: %d\n", result.Documents)
	fmt.Fprintf(output, "chunks: %d\n", result.Chunks)
	fmt.Fprintf(output, "scope: %s\n", result.Scope)
	fmt.Fprintf(output, "fingerprint: %s\n", result.Fingerprint)
	return nil
}

func runIndex(vaultPath string, stdout io.Writer) error {
	root := vaultPath
	written, enabled, err := vault.GenerateWorkspaceIndexes(root, vault.IndexOptions{Overwrite: true})
	if err != nil {
		return err
	}
	if !enabled {
		fmt.Fprintf(stdout, "ok: index disabled under %s\n", filepath.Clean(root))
		return nil
	}
	for _, path := range written {
		fmt.Fprintln(stdout, path)
	}
	fmt.Fprintf(stdout, "ok: index generated under %s\n", filepath.Clean(root))
	return nil
}
