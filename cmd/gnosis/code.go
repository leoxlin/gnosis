package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/codeintel"
)

func newIndexCodeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope, dispose string
	command := &cobra.Command{
		Use:     "code --scope <scope>",
		Short:   "Build one configured immutable code index",
		Args:    cobra.NoArgs,
		Example: "gnosis index code --scope app",
		RunE: func(command *cobra.Command, _ []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("index code: --scope is required"))
			}
			if dispose != "" {
				changed, err := codeintel.DisposeGeneration(options.vaultPath, scope, dispose)
				if err != nil {
					return fmt.Errorf("dispose code generation: %w", err)
				}
				return writeTOON(stdout, toon.NewObject(
					toon.Field{Key: "action", Value: "dispose"}, toon.Field{Key: "resource", Value: "code_generation"},
					toon.Field{Key: "scope", Value: scope}, toon.Field{Key: "generation", Value: dispose}, toon.Field{Key: "changed", Value: changed},
				))
			}
			result, err := codeintel.Build(command.Context(), options.vaultPath, scope)
			if err != nil {
				return fmt.Errorf("index code: %w", err)
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "action", Value: "index"}, toon.Field{Key: "resource", Value: "code"},
				toon.Field{Key: "scope", Value: result.Scope}, toon.Field{Key: "status", Value: result.Status},
				toon.Field{Key: "generation", Value: result.Generation}, toon.Field{Key: "documents", Value: result.Documents},
				toon.Field{Key: "symbols", Value: result.Symbols}, toon.Field{Key: "relations", Value: result.Relations},
				toon.Field{Key: "diagnostics", Value: result.Diagnostics},
			))
		},
	}
	command.Flags().StringVar(&scope, "scope", "", "configured code scope")
	command.Flags().StringVar(&dispose, "dispose-generation", "", "remove one non-current derived generation")
	return command
}

func newSearchCodeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope, language, fields string
	var limit int
	command := &cobra.Command{
		Use:   "code <query> --scope <scope>",
		Short: "Search symbols in one current code generation",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis search code Handler --scope app\n" +
			"gnosis search code Handler --scope app --fields id,name,kind,path,span",
		RunE: func(command *cobra.Command, args []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("search code: --scope is required"))
			}
			selector, err := parseFields(fields, []string{"id", "name", "kind", "path"}, []string{"id", "name", "qualified_name", "kind", "path", "language", "signature", "span"})
			if err != nil {
				return newUsageError(err)
			}
			result, err := codeintel.NewService(options.vaultPath).Search(command.Context(), scope, args[0], language, limit)
			if err != nil {
				return err
			}
			rows := make([]toon.Object, 0, len(result.Symbols))
			for _, symbol := range result.Symbols {
				current := symbol
				rows = append(rows, selector.object(func(name string) (any, bool) {
					switch name {
					case "id":
						return current.ID, true
					case "name":
						return current.Name, true
					case "qualified_name":
						return current.QualifiedName, true
					case "kind":
						return current.Kind, true
					case "path":
						return current.Path, true
					case "language":
						return current.Language, true
					case "signature":
						return current.Signature, true
					case "span":
						return current.Span, true
					default:
						return nil, false
					}
				}))
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "scope", Value: result.Scope}, toon.Field{Key: "generation", Value: result.Generation},
				toon.Field{Key: "snapshot", Value: result.Snapshot}, toon.Field{Key: "provenance", Value: result.Provenance}, toon.Field{Key: "coverage", Value: result.Coverage},
				toon.Field{Key: "count", Value: len(rows)}, toon.Field{Key: "total", Value: result.Total},
				toon.Field{Key: "truncated", Value: result.Truncated}, toon.Field{Key: "symbols", Value: rows},
			))
		},
	}
	command.Flags().StringVar(&scope, "scope", "", "configured code scope")
	command.Flags().StringVar(&language, "language", "", "canonical language filter")
	command.Flags().IntVar(&limit, "limit", 0, "maximum symbols to return")
	command.Flags().StringVar(&fields, "fields", "", "fields: id, name, qualified_name, kind, path, language, signature, span")
	return command
}

func newGetCodeSymbolCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope string
	command := &cobra.Command{
		Use: "code-symbol <id> --scope <scope>", Short: "Read one exact code symbol", Args: cobra.ExactArgs(1),
		Example: "gnosis get code-symbol <id> --scope app",
		RunE: func(command *cobra.Command, args []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("get code-symbol: --scope is required"))
			}
			result, err := codeintel.NewService(options.vaultPath).ReadSymbol(command.Context(), scope, args[0])
			if err != nil {
				return err
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "scope", Value: result.Scope}, toon.Field{Key: "generation", Value: result.Generation},
				toon.Field{Key: "snapshot", Value: result.Snapshot}, toon.Field{Key: "provenance", Value: result.Provenance},
				toon.Field{Key: "coverage", Value: result.Coverage}, toon.Field{Key: "symbol", Value: result.Symbol},
			))
		},
	}
	command.Flags().StringVar(&scope, "scope", "", "configured code scope")
	return command
}

func newGetCodeStatusCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope string
	command := &cobra.Command{
		Use: "code-index-status --scope <scope>", Short: "Read code-index status and provenance", Args: cobra.NoArgs,
		Example: "gnosis get code-index-status --scope app",
		RunE: func(command *cobra.Command, _ []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("get code-index-status: --scope is required"))
			}
			status, err := codeintel.NewService(options.vaultPath).Status(command.Context(), scope)
			if err != nil {
				return err
			}
			return writeTOON(stdout, toon.NewObject(toon.Field{Key: "status", Value: status}))
		},
	}
	command.Flags().StringVar(&scope, "scope", "", "configured code scope")
	return command
}

func newGetCodeDiagnosticsCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope, path, language, category string
	var limit int
	command := &cobra.Command{
		Use: "code-diagnostics --scope <scope>", Short: "Read bounded code diagnostics", Args: cobra.NoArgs,
		Example: "gnosis get code-diagnostics --scope app --language go",
		RunE: func(command *cobra.Command, _ []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("get code-diagnostics: --scope is required"))
			}
			result, err := codeintel.NewService(options.vaultPath).Diagnostics(command.Context(), scope, path, language, category, limit)
			if err != nil {
				return err
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "scope", Value: result.Scope}, toon.Field{Key: "generation", Value: result.Generation},
				toon.Field{Key: "snapshot", Value: result.Snapshot}, toon.Field{Key: "provenance", Value: result.Provenance},
				toon.Field{Key: "coverage", Value: result.Coverage}, toon.Field{Key: "count", Value: len(result.Diagnostics)},
				toon.Field{Key: "total", Value: result.Total}, toon.Field{Key: "truncated", Value: result.Truncated}, toon.Field{Key: "diagnostics", Value: result.Diagnostics},
			))
		},
	}
	flags := command.Flags()
	flags.StringVar(&scope, "scope", "", "configured code scope")
	flags.StringVar(&path, "path", "", "repository-relative path filter")
	flags.StringVar(&language, "language", "", "canonical language filter")
	flags.StringVar(&category, "category", "", "diagnostic category filter")
	flags.IntVar(&limit, "limit", 0, "maximum diagnostics to return")
	return command
}

func newGraphCodeCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var scope, direction, target string
	var limit, depth int
	var neighbors bool
	command := &cobra.Command{
		Use: "code <symbol-id> --scope <scope>", Short: "Trace bounded code relations", Args: cobra.ExactArgs(1),
		Example: "gnosis graph code <symbol-id> --scope app --direction outgoing\n" +
			"gnosis graph code <symbol-id> --scope app --neighbors\n" +
			"gnosis graph code <symbol-id> --scope app --target <symbol-id> --depth 4",
		RunE: func(command *cobra.Command, args []string) error {
			if scope == "" {
				return newUsageError(fmt.Errorf("graph code: --scope is required"))
			}
			if direction != "incoming" && direction != "outgoing" {
				return newUsageError(fmt.Errorf("graph code: --direction must be incoming or outgoing"))
			}
			service := codeintel.NewService(options.vaultPath)
			var result codeintel.TraceResult
			var err error
			if target != "" {
				result, err = service.Path(command.Context(), scope, args[0], target, direction, depth, limit)
			} else if neighbors {
				result, err = service.Neighbors(command.Context(), scope, args[0], direction, limit)
			} else {
				result, err = service.Trace(command.Context(), scope, args[0], direction, limit)
			}
			if err != nil {
				return err
			}
			return writeTOON(stdout, toon.NewObject(
				toon.Field{Key: "mode", Value: result.Mode}, toon.Field{Key: "scope", Value: result.Scope}, toon.Field{Key: "generation", Value: result.Generation},
				toon.Field{Key: "snapshot", Value: result.Snapshot}, toon.Field{Key: "provenance", Value: result.Provenance}, toon.Field{Key: "coverage", Value: result.Coverage},
				toon.Field{Key: "count", Value: len(result.Relations)}, toon.Field{Key: "total", Value: result.Total}, toon.Field{Key: "truncated", Value: result.Truncated},
				toon.Field{Key: "relations", Value: result.Relations}, toon.Field{Key: "symbols", Value: result.Symbols},
			))
		},
	}
	flags := command.Flags()
	flags.StringVar(&scope, "scope", "", "configured code scope")
	flags.StringVar(&direction, "direction", "outgoing", "relation direction: incoming or outgoing")
	flags.IntVar(&limit, "limit", 0, "maximum relations to return")
	flags.BoolVar(&neighbors, "neighbors", false, "return adjacent symbols")
	flags.StringVar(&target, "target", "", "target symbol ID for path mode")
	flags.IntVar(&depth, "depth", 0, "maximum path depth")
	return command
}
