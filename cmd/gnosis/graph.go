package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"strings"
)

func newGraphCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "graph",
		Short: "Traverse exact directed vault links",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newGraphNeighborsCommand(stdout), newGraphPathCommand(stdout))
	return command
}

func newGraphNeighborsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, uri, direction string
	var relations []string
	command := &cobra.Command{
		Use:   "neighbors [flags]",
		Short: "List typed links adjacent to one exact page",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			uri = strings.TrimSpace(uri)
			if !strings.HasPrefix(uri, "gnosis://") {
				return errors.New("graph neighbors: --uri must be a gnosis URI")
			}
			result, err := vault.TraceNeighbors(vaultPath, uri, vault.Direction(direction), relations)
			if err != nil {
				return fmt.Errorf("graph neighbors: %w", err)
			}
			return writeJSON(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&uri, "uri", "", "exact page gnosis URI")
	flags.StringVar(&direction, "direction", string(vault.DirectionBoth), "edge direction: out, in, or both")
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	return command
}

func newGraphPathCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, from, to, direction string
	var relations []string
	var depth int
	command := &cobra.Command{
		Use:   "path [flags]",
		Short: "Find a typed path between exact pages",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			from = strings.TrimSpace(from)
			to = strings.TrimSpace(to)
			if !strings.HasPrefix(from, "gnosis://") {
				return errors.New("graph path: --from must be a gnosis URI")
			}
			if !strings.HasPrefix(to, "gnosis://") {
				return errors.New("graph path: --to must be a gnosis URI")
			}
			if depth < 0 {
				return errors.New("graph path: --depth must be zero or greater")
			}
			result, err := vault.TracePath(vaultPath, from, to, vault.Direction(direction), relations, depth)
			if err != nil {
				return fmt.Errorf("graph path: %w", err)
			}
			return writeJSON(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&from, "from", "", "source page gnosis URI")
	flags.StringVar(&to, "to", "", "target page gnosis URI")
	flags.StringVar(&direction, "direction", string(vault.DirectionBoth), "edge direction: out, in, or both")
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	flags.IntVar(&depth, "depth", 3, "maximum traversal depth")
	return command
}
