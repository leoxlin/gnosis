package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"os"
	"path/filepath"
)

func newScaffoldCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, vaultName string
	var force, includeConcepts bool
	command := &cobra.Command{
		Use:   "scaffold [flags]",
		Short: "Create an OKF-compatible gnosis vault",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runScaffold(vaultPath, vaultName, force, includeConcepts, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the new gnosis vault")
	flags.StringVar(&vaultName, "name", "", "name for the new vault")
	flags.BoolVar(&force, "force", false, "overwrite existing files")
	flags.BoolVar(&includeConcepts, "concepts", false, "include reusable project concept definitions")
	return command
}

func runScaffold(vaultPath, vaultName string, force, includeConcepts bool, stdout io.Writer) error {
	root := vaultPath
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	created, err := vault.Scaffold(root, vault.ScaffoldOptions{Force: force, Name: vaultName})
	if err != nil {
		return err
	}
	if includeConcepts {
		conceptPaths, err := writeConcepts(root, force)
		if err != nil {
			return err
		}
		created = append(created, conceptPaths...)
		if len(conceptPaths) > 0 {
			indexPaths, _, err := vault.GenerateWorkspaceIndexes(root, vault.IndexOptions{Overwrite: true})
			if err != nil {
				return err
			}
			if len(indexPaths) > 0 {
				created = append(created, indexPaths...)
			}
		}
	}
	for _, path := range created {
		fmt.Fprintln(stdout, path)
	}
	fmt.Fprintf(stdout, "ok: vault scaffolded under %s\n", filepath.Clean(root))
	return nil
}

func writeConcepts(root string, force bool) ([]string, error) {
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
