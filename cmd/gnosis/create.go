package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
		Example: "gnosis create vault --name <name>\n" +
			"gnosis create vault --vault <path> --concepts",
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
		Example: "gnosis create vault --name <name>\n" +
			"gnosis create vault --vault <path> --name <name>\n" +
			"gnosis create vault --vault <path> --concepts",
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
	root := filepath.Clean(vaultPath)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("create vault: make root: %w", err)
	}
	created, err := vault.Scaffold(root, vault.ScaffoldOptions{
		Force: isForce,
		Name:  vaultName,
	})
	if err != nil {
		return fmt.Errorf("create vault: %w", err)
	}
	if hasConcepts {
		conceptPaths, err := writeBundledConcepts(root, isForce)
		if err != nil {
			return fmt.Errorf("create vault: concepts: %w", err)
		}
		created = append(created, conceptPaths...)
		if len(conceptPaths) > 0 {
			indexPaths, _, err := vault.GenerateWorkspaceIndexes(
				root,
				vault.IndexOptions{Overwrite: true},
			)
			if err != nil {
				return fmt.Errorf("create vault: indexes: %w", err)
			}
			created = append(created, indexPaths...)
		}
	}
	status := "created"
	if len(created) == 0 {
		status = "no-op"
	}
	return writeTOON(stdout, toon.NewObject(
		toon.Field{Key: "action", Value: "create"},
		toon.Field{Key: "resource", Value: "vault"},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "path", Value: root},
		toon.Field{Key: "changed", Value: len(created) > 0},
		toon.Field{Key: "files", Value: created},
	))
}

func writeBundledConcepts(root string, force bool) ([]string, error) {
	documents, err := vault.BundledConcepts()
	if err != nil {
		return nil, err
	}

	created := make([]string, 0, len(documents))
	for _, document := range documents {
		path := filepath.Join(root, document.Path)
		changed, err := vault.WriteGeneratedFile(path, document.Data, force)
		if err != nil {
			return created, err
		}
		if changed {
			created = append(created, path)
		}
	}
	return created, nil
}
