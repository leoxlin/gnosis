package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"path/filepath"
)

func newIndexCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "index [flags]",
		Short: "Generate vault indexes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runIndex(vaultPath, stdout)
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
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
