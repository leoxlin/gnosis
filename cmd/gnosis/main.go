package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"gnosis/internal/mcpserver"
	"gnosis/internal/vault"
)

const defaultVault = "."

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		command := newRootCommand(stdout, stderr)
		command.SetOut(stderr)
		if err := command.Usage(); err != nil {
			return err
		}
		return errors.New("missing command")
	}

	command := newRootCommand(stdout, stderr)
	command.SetArgs(normalizeLegacyLongFlags(args))
	return command.Execute()
}

func newRootCommand(stdout, stderr io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:           "gnosis",
		Short:         "Manage an OKF-compatible Obsidian vault",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.AddCommand(newScaffoldCommand(stdout), newSetupCommand(stdout), newIndexCommand(stdout), newReadCommand(stdout), newWriteCommand(os.Stdin, stdout), newValidateCommand(stdout, stderr), newQueryCommand(stdout), newConceptsCommand(stdout), newProcessCommand(stdout), newGraphCommand(stdout), newMCPCommand())
	command.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the gnosis version",
		Args:  noArgs("version"),
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintln(stdout, "gnosis 0.1.0")
		},
	})
	return command
}

func newConceptsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, conceptType string
	var jsonOutput, pretty bool
	command := &cobra.Command{
		Use:   "concepts [flags]",
		Short: "List concept types or concepts of an exact type",
		Args:  noArgs("concepts"),
		RunE: func(_ *cobra.Command, _ []string) error {
			if jsonOutput || pretty {
				catalog, err := vault.Concepts(vaultPath, conceptType)
				if err != nil {
					return err
				}
				return writeJSON(stdout, catalog, pretty)
			}
			return vault.ListConcepts(vaultPath, conceptType, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&conceptType, "type", "", "exact concept type")
	flags.BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output (implies --json)")
	return command
}

func newReadCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, conceptType, title, id string
	var jsonOutput, pretty bool
	command := &cobra.Command{
		Use:   "read [gnosis-uri] [flags]",
		Short: "Print one exact vault document",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) <= 1 {
				return nil
			}
			return fmt.Errorf("read: unexpected argument(s): %s", strings.Join(args[1:], " "))
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 1 {
				if strings.TrimSpace(id) != "" {
					return errors.New("read: positional gnosis URI cannot be combined with --id")
				}
				if !strings.HasPrefix(strings.TrimSpace(args[0]), "gnosis://") {
					return errors.New("read: positional argument must be a gnosis URI")
				}
				id = args[0]
			}
			return runRead(vaultPath, id, conceptType, title, jsonOutput, pretty, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&id, "id", "", "exact effective page ID or gnosis URI")
	flags.StringVar(&conceptType, "type", "", "exact document type")
	flags.StringVar(&title, "title", "", "exact document title")
	flags.BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON (requires --id)")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output (implies --json)")
	return command
}

func runRead(vaultPath, id, conceptType, title string, jsonOutput, pretty bool, stdout io.Writer) error {
	id = strings.TrimSpace(id)
	conceptType = strings.TrimSpace(conceptType)
	title = strings.TrimSpace(title)
	if id != "" {
		if conceptType != "" || title != "" {
			return errors.New("read: --id cannot be combined with --type or --title")
		}
		page, err := vault.ReadPage(vaultPath, id)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		if jsonOutput || pretty {
			return writeJSON(stdout, page, pretty)
		}
		_, err = io.WriteString(stdout, page.Markdown)
		return err
	}
	if jsonOutput || pretty {
		return errors.New("read: --json and --pretty require --id")
	}
	if conceptType == "" {
		return errors.New("read: --type must not be empty when --id is not used")
	}
	if title == "" {
		return errors.New("read: --title must not be empty when --id is not used")
	}

	document, err := vault.Read(vaultPath, conceptType, title)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	_, err = stdout.Write(document)
	return err
}

func newWriteCommand(input io.Reader, stdout io.Writer) *cobra.Command {
	var conceptType, title string
	var overwrite bool
	command := &cobra.Command{
		Use:   "write [filename]",
		Short: "Write a typed Markdown document into the current vault",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) <= 1 {
				return nil
			}
			return fmt.Errorf("write: unexpected argument(s): %s", strings.Join(args[1:], " "))
		},
		RunE: func(_ *cobra.Command, args []string) error {
			var content []byte
			var err error
			if len(args) == 1 {
				content, err = os.ReadFile(args[0])
				if err != nil {
					return fmt.Errorf("write: read %s: %w", args[0], err)
				}
			} else {
				content, err = io.ReadAll(input)
				if err != nil {
					return fmt.Errorf("write: read standard input: %w", err)
				}
			}
			path, err := vault.WriteDocument(defaultVault, conceptType, title, content, overwrite)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, path)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&conceptType, "type", "", "exact concept type")
	flags.StringVar(&title, "title", "", "exact document title")
	flags.BoolVar(&overwrite, "overwrite", false, "allow overriding imported or built-in documents")
	return command
}

func newQueryCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "query",
		Short: "Query the vault",
		Args:  noArgs("query"),
	}
	command.AddCommand(newSearchQueryCommand(stdout), newGraphQueryCommand(stdout))
	return command
}

func newProcessCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "process",
		Short: "Discover and invoke executable vault processes",
		Args:  noArgs("process"),
	}
	command.AddCommand(newProcessDiscoverCommand(stdout), newProcessInvokeCommand(stdout))
	return command
}

func newProcessDiscoverCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var processTypes []string
	var top int
	var pretty bool
	command := &cobra.Command{
		Use:   "discover [flags] <request>",
		Short: "Rank executable processes for an agent request",
		Args:  questionArgs("process discover"),
		RunE: func(_ *cobra.Command, args []string) error {
			if top <= 0 {
				return errors.New("process discover: --top must be greater than zero")
			}
			result, err := vault.DiscoverProcesses(vaultPath, args[0], processTypes, top)
			if err != nil {
				return fmt.Errorf("process discover: %w", err)
			}
			return writeJSON(stdout, result, pretty)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringSliceVar(&processTypes, "type", nil, "executable process type (Gnosis Process)")
	flags.IntVar(&top, "top", 5, "number of process candidates to return")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	return command
}

func newProcessInvokeCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, id string
	var pretty bool
	command := &cobra.Command{
		Use:   "invoke [flags]",
		Short: "Load one exact process execution contract",
		Args:  noArgs("process invoke"),
		RunE: func(_ *cobra.Command, _ []string) error {
			id = strings.TrimSpace(id)
			if id == "" {
				return errors.New("process invoke: --id must not be empty")
			}
			result, err := vault.InvokeProcess(vaultPath, id)
			if err != nil {
				return fmt.Errorf("process invoke: %w", err)
			}
			return writeJSON(stdout, result, pretty)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&id, "id", "", "exact process ID or gnosis URI")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	return command
}

func newGraphCommand(stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:   "graph",
		Short: "Traverse exact directed vault links",
		Args:  noArgs("graph"),
	}
	command.AddCommand(newGraphNeighborsCommand(stdout), newGraphPathCommand(stdout))
	return command
}

func newGraphNeighborsCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, id, direction string
	var relations []string
	var pretty bool
	command := &cobra.Command{
		Use:   "neighbors [flags]",
		Short: "List typed links adjacent to one exact page",
		Args:  noArgs("graph neighbors"),
		RunE: func(_ *cobra.Command, _ []string) error {
			id = strings.TrimSpace(id)
			if id == "" {
				return errors.New("graph neighbors: --id must not be empty")
			}
			result, err := vault.TraceNeighbors(vaultPath, id, vault.Direction(direction), relations)
			if err != nil {
				return fmt.Errorf("graph neighbors: %w", err)
			}
			return writeJSON(stdout, result, pretty)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&id, "id", "", "exact page ID or gnosis URI")
	flags.StringVar(&direction, "direction", string(vault.DirectionBoth), "edge direction: out, in, or both")
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	return command
}

func newGraphPathCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, from, to, direction string
	var relations []string
	var depth int
	var pretty bool
	command := &cobra.Command{
		Use:   "path [flags]",
		Short: "Find a typed path between exact pages",
		Args:  noArgs("graph path"),
		RunE: func(_ *cobra.Command, _ []string) error {
			from = strings.TrimSpace(from)
			to = strings.TrimSpace(to)
			if from == "" {
				return errors.New("graph path: --from must not be empty")
			}
			if to == "" {
				return errors.New("graph path: --to must not be empty")
			}
			if depth < 0 {
				return errors.New("graph path: --depth must be zero or greater")
			}
			result, err := vault.TracePath(vaultPath, from, to, vault.Direction(direction), relations, depth)
			if err != nil {
				return fmt.Errorf("graph path: %w", err)
			}
			return writeJSON(stdout, result, pretty)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&from, "from", "", "source page ID or gnosis URI")
	flags.StringVar(&to, "to", "", "target page ID or gnosis URI")
	flags.StringVar(&direction, "direction", string(vault.DirectionBoth), "edge direction: out, in, or both")
	flags.StringSliceVar(&relations, "relation", nil, "relationship type filter")
	flags.IntVar(&depth, "depth", 3, "maximum traversal depth")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	return command
}

func newMCPCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "mcp",
		Short: "Serve the gnosis agent contract over MCP",
		Args:  noArgs("mcp"),
	}
	command.AddCommand(newMCPServeCommand())
	return command
}

func newMCPServeCommand() *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "serve [flags]",
		Short: "Run the MCP server over standard input and output",
		Args:  noArgs("mcp serve"),
		RunE: func(command *cobra.Command, _ []string) error {
			if err := mcpserver.Serve(command.Context(), vaultPath); err != nil {
				return fmt.Errorf("mcp serve: %w", err)
			}
			return nil
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}

func newSearchQueryCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var top, maxRead, depth int
	var jsonOutput, pretty bool
	command := &cobra.Command{
		Use:   "search [flags] <question>",
		Short: "Find relevant vault pages for a question",
		Args:  questionArgs("query search"),
		RunE: func(_ *cobra.Command, args []string) error {
			return runQuery(vaultPath, top, maxRead, depth, jsonOutput, pretty, args[0], stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	flags.BoolVar(&jsonOutput, "json", false, "emit machine-readable JSON")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output (implies --json)")
	return command
}

func newGraphQueryCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var top, maxRead, depth int
	var pretty bool
	command := &cobra.Command{
		Use:   "graph [flags] <question>",
		Short: "Query the vault and emit graph-aware JSON",
		Args:  questionArgs("query graph"),
		RunE: func(_ *cobra.Command, args []string) error {
			return runGraphQuery(vaultPath, top, maxRead, depth, pretty, args[0], stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.IntVar(&top, "top", 3, "number of candidate pages to return")
	flags.IntVar(&maxRead, "max-read", 3, "maximum number of pages to recommend reading")
	flags.IntVar(&depth, "depth", 3, "maximum graph traversal depth")
	flags.BoolVar(&pretty, "pretty", false, "pretty-print JSON output")
	return command
}

func runQuery(vaultPath string, top, maxRead, depth int, jsonOutput, pretty bool, question string, stdout io.Writer) error {
	if err := validateQueryOptions(top, maxRead, depth); err != nil {
		return fmt.Errorf("query search: %w", err)
	}

	result, err := queryVault(vaultPath, question, vault.QueryOptions{
		Top:      top,
		MaxRead:  maxRead,
		MaxDepth: depth,
	})
	if err != nil {
		return err
	}
	if jsonOutput || pretty {
		return writeQueryJSON(stdout, result, pretty)
	}
	writeQueryText(stdout, result)
	return nil
}

func runGraphQuery(vaultPath string, top, maxRead, depth int, pretty bool, question string, stdout io.Writer) error {
	if err := validateQueryOptions(top, maxRead, depth); err != nil {
		return fmt.Errorf("query graph: %w", err)
	}

	result, err := queryVault(vaultPath, question, vault.QueryOptions{
		Top:      top,
		MaxRead:  maxRead,
		MaxDepth: depth,
	})
	if err != nil {
		return err
	}
	return writeQueryJSON(stdout, result, pretty)
}

func queryVault(root, question string, options vault.QueryOptions) (vault.QueryResult, error) {
	return vault.QueryKnowledge(root, question, options)
}

func validateQueryOptions(top, maxRead, depth int) error {
	if top <= 0 {
		return errors.New("-top must be greater than zero")
	}
	if maxRead < 0 {
		return errors.New("-max-read must be zero or greater")
	}
	if depth <= 0 {
		return errors.New("-depth must be greater than zero")
	}
	return nil
}

func writeQueryJSON(output io.Writer, result vault.QueryResult, pretty bool) error {
	return writeJSON(output, result, pretty)
}

func writeJSON(output io.Writer, value any, pretty bool) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(value)
}

func writeQueryText(output io.Writer, result vault.QueryResult) {
	fmt.Fprintf(output, "answer_type: %s\n", result.AnswerType)
	fmt.Fprintf(output, "index_only: %t\n", result.IndexOnly)
	if len(result.Candidates) == 0 {
		fmt.Fprintln(output, "no matches")
		return
	}
	fmt.Fprintln(output, "candidates:")
	for _, candidate := range result.Candidates {
		fmt.Fprintf(output, "- %s (%s)", candidate.Title, candidate.Page)
		if candidate.Description != "" {
			fmt.Fprintf(output, " - %s", candidate.Description)
		}
		fmt.Fprintln(output)
	}
	if len(result.Path) > 0 {
		fmt.Fprintln(output, "path:")
		fmt.Fprintln(output, strings.Join(result.Path, " -> "))
	}
	if len(result.ShouldRead) > 0 {
		fmt.Fprintln(output, "should_read:")
		for _, page := range result.ShouldRead {
			fmt.Fprintf(output, "- %s\n", page)
		}
	}
}

func newIndexCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "index [flags]",
		Short: "Generate vault indexes",
		Args:  noArgs("index"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runIndex(vaultPath, stdout)
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}

func runIndex(vaultPath string, stdout io.Writer) error {
	root := vaultPath
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	resolution, err := vault.ResolveConfig(root)
	if err != nil {
		return err
	}
	if !resolution.Config.IndexEnabled() {
		fmt.Fprintf(stdout, "ok: index disabled under %s\n", filepath.Clean(root))
		return nil
	}

	var written []string
	for _, vaultRoot := range resolution.LocalVaultRoots {
		paths, err := vault.GenerateIndexes(vaultRoot, vault.IndexOptions{Overwrite: true})
		if err != nil {
			return err
		}
		written = append(written, paths...)
	}
	for _, path := range written {
		fmt.Fprintln(stdout, path)
	}
	fmt.Fprintf(stdout, "ok: index generated under %s\n", filepath.Clean(root))
	return nil
}

func newValidateCommand(stdout, stderr io.Writer) *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "validate [flags]",
		Short: "Validate vault structure and links",
		Args:  noArgs("validate"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runValidate(vaultPath, stdout, stderr)
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}

func runValidate(vaultPath string, stdout, stderr io.Writer) error {
	result, err := vault.Validate(vaultPath)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintf(stderr, "warning: %s\n", warning)
	}
	if len(result.Errors) > 0 {
		for _, validationErr := range result.Errors {
			fmt.Fprintf(stderr, "error: %s\n", validationErr)
		}
		return fmt.Errorf("validation failed: %d error(s)", len(result.Errors))
	}
	fmt.Fprintf(stdout, "ok: %d markdown file(s) validated\n", result.FilesChecked)
	return nil
}

func newScaffoldCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, vaultName string
	var force, includeConcepts bool
	command := &cobra.Command{
		Use:   "scaffold [flags]",
		Short: "Create an OKF-compatible gnosis vault",
		Args:  noArgs("scaffold"),
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
			resolution, err := vault.ResolveConfig(root)
			if err != nil {
				return err
			}
			if resolution.Config.IndexEnabled() {
				indexPaths, err := vault.GenerateIndexes(root, vault.IndexOptions{Overwrite: true})
				if err != nil {
					return err
				}
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

func newSetupCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var imports []string
	var force bool
	command := &cobra.Command{
		Use:   "setup [flags]",
		Short: "Configure a workspace to import gnosis vaults",
		Args:  noArgs("setup"),
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

func questionArgs(command string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("%s: missing question", command)
		}
		if len(args) > 1 {
			return fmt.Errorf("%s: unexpected argument(s): %s", command, strings.Join(args[1:], " "))
		}
		if strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("%s: question must not be empty", command)
		}
		return nil
	}
}

func noArgs(command string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("%s: unexpected argument(s): %s", command, strings.Join(args, " "))
	}
}

// normalizeLegacyLongFlags keeps the single-dash long options accepted by the
// previous flag-based CLI while Cobra uses its standard double-dash spelling.
func normalizeLegacyLongFlags(args []string) []string {
	longFlags := map[string]bool{
		"vault": true, "top": true, "max-read": true, "depth": true,
		"json": true, "pretty": true, "force": true, "concepts": true, "name": true, "import": true,
		"type": true, "title": true, "id": true, "from": true, "to": true, "direction": true, "relation": true, "overwrite": true,
	}
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--" {
			normalized = append(normalized, args[len(normalized):]...)
			break
		}
		if strings.HasPrefix(arg, "-") && !strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "-"), "=")
			if longFlags[name] {
				arg = "--" + name
				if hasValue {
					arg += "=" + value
				}
			}
		}
		normalized = append(normalized, arg)
	}
	return normalized
}
