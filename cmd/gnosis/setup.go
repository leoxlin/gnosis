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
	var vaultPath, githubWiki, vaultName string
	var imports []string
	var force bool
	command := &cobra.Command{
		Use:   "setup [flags]",
		Short: "Configure a gnosis workspace",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSetup(vaultPath, imports, githubWiki, vaultName, force, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "directory for gnosis.toml")
	flags.StringSliceVar(&imports, "import", nil, "path of a vault to import")
	flags.StringVar(&githubWiki, "github-wiki", "", "GitHub OWNER/REPOSITORY to use as the primary vault")
	flags.StringVar(&vaultName, "name", "", "canonical name for the primary vault")
	flags.BoolVar(&force, "force", false, "overwrite an existing gnosis.toml")
	return command
}

func runSetup(vaultPath string, imports []string, githubWiki, vaultName string, force bool, stdout io.Writer) error {
	if githubWiki != "" && len(imports) > 0 {
		return errors.New("setup: --github-wiki cannot be combined with --import")
	}
	if githubWiki != "" && vaultName == "" {
		return errors.New("setup: --name is required with --github-wiki")
	}
	if githubWiki == "" && vaultName != "" {
		return errors.New("setup: --name requires --github-wiki")
	}
	if githubWiki == "" && len(imports) == 0 {
		return errors.New("setup: at least one --import is required")
	}
	if err := os.MkdirAll(vaultPath, 0o755); err != nil {
		return err
	}
	var changed bool
	var err error
	if githubWiki != "" {
		changed, err = vault.WriteGitHubWikiConfig(vaultPath, vaultName, githubWiki, force)
	} else {
		changed, err = vault.WriteWorkspaceConfig(vaultPath, imports, force)
	}
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintln(stdout, filepath.Join(vaultPath, "gnosis.toml"))
	}
	fmt.Fprintf(stdout, "ok: workspace configured under %s\n", filepath.Clean(vaultPath))
	return nil
}
