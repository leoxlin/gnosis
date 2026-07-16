package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newGraphCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "graph",
		Short:   "Traverse exact directed vault links",
		Args:    cobra.NoArgs,
		GroupID: "knowledge",
		Example: "gnosis graph neighbors <gnosis-uri>\n" +
			"gnosis graph path <from-uri> <to-uri>",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("graph: missing operation"))
		},
	}
	command.AddCommand(
		newGraphNeighborsCommand(options, stdout),
		newGraphPathCommand(options, stdout),
	)
	return command
}

func newGraphNeighborsCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var direction string
	var relations []string
	command := &cobra.Command{
		Use:   "neighbors <gnosis-uri> [flags]",
		Short: "List typed links adjacent to one exact page",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis graph neighbors <gnosis-uri>\n" +
			"gnosis graph neighbors <gnosis-uri> --direction out --relation supports",
		RunE: func(_ *cobra.Command, args []string) error {
			uri := strings.TrimSpace(args[0])
			if !isGnosisURI(uri) {
				return newUsageError(errors.New("graph neighbors: argument must be a gnosis uri"))
			}
			if err := validateDirection(direction); err != nil {
				return newUsageError(fmt.Errorf("graph neighbors: %w", err))
			}
			result, err := vault.TraceNeighbors(
				options.vaultPath, uri, vault.Direction(direction), relations,
			)
			if err != nil {
				return fmt.Errorf("graph neighbors: %w", err)
			}
			return writeGraphNeighbors(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(
		&direction,
		"direction",
		string(vault.DirectionBoth),
		"edge direction: out, in, or both",
	)
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	return command
}

func newGraphPathCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var direction string
	var relations []string
	var depth int
	command := &cobra.Command{
		Use:   "path <from-uri> <to-uri> [flags]",
		Short: "Find a typed path between exact pages",
		Args:  cobra.ExactArgs(2),
		Example: "gnosis graph path <from-uri> <to-uri>\n" +
			"gnosis graph path <from-uri> <to-uri> --depth 5 --direction out",
		RunE: func(_ *cobra.Command, args []string) error {
			from := strings.TrimSpace(args[0])
			to := strings.TrimSpace(args[1])
			if !isGnosisURI(from) {
				return newUsageError(errors.New("graph path: from argument must be a gnosis uri"))
			}
			if !isGnosisURI(to) {
				return newUsageError(errors.New("graph path: to argument must be a gnosis uri"))
			}
			if depth < 0 {
				return newUsageError(errors.New("graph path: --depth must be zero or greater"))
			}
			if err := validateDirection(direction); err != nil {
				return newUsageError(fmt.Errorf("graph path: %w", err))
			}
			result, err := vault.TracePath(
				options.vaultPath,
				from,
				to,
				vault.Direction(direction),
				relations,
				depth,
			)
			if err != nil {
				return fmt.Errorf("graph path: %w", err)
			}
			return writeGraphPath(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVar(
		&direction,
		"direction",
		string(vault.DirectionBoth),
		"edge direction: out, in, or both",
	)
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	flags.IntVar(&depth, "depth", 3, "maximum traversal depth")
	return command
}

func isGnosisURI(value string) bool {
	return vault.IsCanonicalURI(value)
}

func validateDirection(value string) error {
	switch vault.Direction(value) {
	case vault.DirectionOut, vault.DirectionIn, vault.DirectionBoth:
		return nil
	default:
		return errors.New("--direction must be out, in, or both")
	}
}

func writeGraphNeighbors(output io.Writer, result vault.GraphNeighbors) error {
	edges := graphEdges(result.Edges)
	fields := []toon.Field{
		{Key: "node", Value: documentObject(result.Node)},
		{Key: "direction", Value: string(result.Direction)},
		{Key: "count", Value: len(edges)},
		{Key: "edges", Value: edges},
	}
	if len(edges) == 0 {
		fields = append(fields, toon.Field{Key: "message", Value: "0 adjacent links found"})
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func writeGraphPath(output io.Writer, result vault.GraphPath) error {
	nodes := make([]toon.Object, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		nodes = append(nodes, documentObject(node))
	}
	from := any(nil)
	if result.From != nil {
		from = documentObject(*result.From)
	}
	to := any(nil)
	if result.To != nil {
		to = documentObject(*result.To)
	}
	return writeTOON(output, toon.NewObject(
		toon.Field{Key: "status", Value: string(result.Status)},
		toon.Field{Key: "from", Value: from},
		toon.Field{Key: "to", Value: to},
		toon.Field{Key: "direction", Value: string(result.Direction)},
		toon.Field{Key: "max_depth", Value: result.MaxDepth},
		toon.Field{Key: "node_count", Value: len(nodes)},
		toon.Field{Key: "nodes", Value: nodes},
		toon.Field{Key: "edge_count", Value: len(result.Edges)},
		toon.Field{Key: "edges", Value: graphEdges(result.Edges)},
	))
}

func graphEdges(source []vault.GraphEdge) []toon.Object {
	edges := make([]toon.Object, 0, len(source))
	for _, edge := range source {
		edges = append(edges, toon.NewObject(
			toon.Field{Key: "from", Value: documentObject(edge.From)},
			toon.Field{Key: "to", Value: documentObject(edge.To)},
			toon.Field{Key: "relation", Value: edge.Relation},
			toon.Field{Key: "raw", Value: edge.Raw},
			toon.Field{Key: "source", Value: edge.Source},
		))
	}
	return edges
}
