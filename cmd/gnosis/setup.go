package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"os"
	"path/filepath"
)

func newSetupCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var imports []string
	var force bool
	command := &cobra.Command{
		Use:   "setup [flags]",
		Short: "Configure a workspace to import gnosis vaults",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSetup(vaultPath, imports, force, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "directory for gnosis.toml")
	flags.StringSliceVar(&imports, "import", nil, "path or URL of a vault to import")
	flags.BoolVar(&force, "force", false, "overwrite an existing gnosis.toml")
	return command
}

func runSetup(vaultPath string, imports []string, force bool, stdout io.Writer) error {
	if len(imports) == 0 {
		return errors.New("setup: at least one --import is required")
	}
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		return err
	}
	changed, err := vault.WriteWorkspaceConfig(vaultPath, imports, force)
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintln(stdout, filepath.Join(vaultPath, "gnosis.toml"))
	}
	fmt.Fprintf(stdout, "ok: workspace configured under %s\n", filepath.Clean(vaultPath))
	return nil
}
