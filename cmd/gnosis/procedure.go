package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"strings"
)

func newProcedureCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "procedure",
		Short: "Discover and load executable vault procedures",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("procedure: missing subcommand")
		},
	}
	command.AddCommand(newProcedureDiscoveryCommand(stdout), newProcedureInvokeCommand(stdout))
	return command
}

func newProcedureDiscoveryCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var tags []string
	command := &cobra.Command{
		Use:   "discovery [flags]",
		Short: "List all model-invocable processes for agent selection",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			result, err := vault.DiscoverProcesses(vaultPath, tags)
			if err != nil {
				return fmt.Errorf("procedure discovery: %w", err)
			}
			return writeJSON(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringSliceVar(&tags, "tags", nil, "require all procedure tags")
	return command
}

func newProcedureInvokeCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, uri string
	command := &cobra.Command{
		Use:   "invoke [flags]",
		Short: "Load one exact procedure execution contract",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			uri = strings.TrimSpace(uri)
			if !strings.HasPrefix(uri, "gnosis://") {
				return errors.New("procedure invoke: --uri must be a gnosis URI")
			}
			result, err := vault.InvokeProcess(vaultPath, uri)
			if err != nil {
				return fmt.Errorf("procedure invoke: %w", err)
			}
			return writeJSON(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&uri, "uri", "", "exact process gnosis URI")
	return command
}
