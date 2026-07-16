package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newIndexCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "index",
		Short:   "Build derived indexes",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis index vault\n" +
			"gnosis index knowledge",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("index: missing resource"))
		},
	}
	command.AddCommand(
		newIndexVaultCommand(options, stdout),
		newIndexKnowledgeCommand(options, stdout),
	)
	return command
}

func newIndexVaultCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "vault",
		Short: "Generate vault indexes",
		Args:  cobra.NoArgs,
		Example: "gnosis index vault\n" +
			"gnosis --vault <path> index vault",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runIndex(options.vaultPath, stdout)
		},
	}
}

func newIndexKnowledgeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "knowledge",
		Short: "Synchronize the semantic knowledge index",
		Args:  cobra.NoArgs,
		Example: "gnosis index knowledge\n" +
			"gnosis --vault <path> index knowledge",
		RunE: func(command *cobra.Command, _ []string) error {
			config, err := vault.SemanticConfigFromEnv(options.vaultPath)
			if err != nil {
				return err
			}
			result, err := vault.SyncSemanticIndex(command.Context(), options.vaultPath, config)
			if err != nil {
				return err
			}
			return writeSemanticIndexResult(stdout, result)
		},
	}
}

func writeSemanticIndexResult(output io.Writer, result vault.SemanticIndexResult) error {
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "action", Value: "index"},
		toon.Field{Key: "resource", Value: "knowledge"},
		toon.Field{Key: "status", Value: "synchronized"},
		toon.Field{Key: "documents", Value: result.Documents},
		toon.Field{Key: "chunks", Value: result.Chunks},
		toon.Field{Key: "scope", Value: result.Scope},
		toon.Field{Key: "fingerprint", Value: result.Fingerprint},
	))
}

func runIndex(vaultPath string, stdout io.Writer) error {
	root := filepath.Clean(vaultPath)
	written, enabled, err := vault.GenerateWorkspaceIndexes(
		root,
		vault.IndexOptions{Overwrite: true},
	)
	if err != nil {
		return fmt.Errorf("index vault: %w", err)
	}
	status := "generated"
	if !enabled {
		status = "disabled"
	} else if len(written) == 0 {
		status = "no-op"
	}
	return writeTOON(stdout, toon.NewObject(
		toon.Field{Key: "action", Value: "index"},
		toon.Field{Key: "resource", Value: "vault"},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "path", Value: root},
		toon.Field{Key: "changed", Value: len(written) > 0},
		toon.Field{Key: "files", Value: written},
	))
}
