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

	"gnosis/internal/forge"
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
	command.AddCommand(newSetupCommand(stdout), newIndexCommand(stdout), newReadCommand(stdout), newValidateCommand(stdout, stderr), newQueryCommand(stdout), newConceptsCommand(stdout))
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
	command := &cobra.Command{
		Use:   "concepts [flags]",
		Short: "List concept types or concepts of an exact type",
		Args:  noArgs("concepts"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return vault.ListConcepts(vaultPath, conceptType, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&conceptType, "type", "", "exact concept type")
	return command
}

func newReadCommand(stdout io.Writer) *cobra.Command {
	var vaultPath, conceptType, title string
	command := &cobra.Command{
		Use:   "read [flags]",
		Short: "Print a vault document by type and title",
		Args:  noArgs("read"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runRead(vaultPath, conceptType, title, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	flags.StringVar(&conceptType, "type", "", "exact document type")
	flags.StringVar(&title, "title", "", "exact document title")
	return command
}

func runRead(vaultPath, conceptType, title string, stdout io.Writer) error {
	conceptType = strings.TrimSpace(conceptType)
	if conceptType == "" {
		return errors.New("read: -type must not be empty")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("read: -title must not be empty")
	}

	document, err := vault.Read(vaultPath, conceptType, title)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	_, err = stdout.Write(document)
	return err
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
	source, err := vault.NewSearchSource(root)
	if err != nil {
		return vault.QueryResult{}, err
	}
	documents, err := source.Documents()
	if err != nil {
		return vault.QueryResult{}, err
	}
	engine := vault.New(documents)
	return engine.Query(question, options), nil
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
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
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
	config, vaultRoots, err := vault.LoadConfig(root)
	if err != nil {
		return err
	}
	if !config.IndexEnabled() {
		fmt.Fprintf(stdout, "ok: index disabled under %s\n", filepath.Clean(root))
		return nil
	}

	var written []string
	for _, vaultRoot := range vaultRoots {
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

func newSetupCommand(stdout io.Writer) *cobra.Command {
	var vaultPath string
	var force, includeConcepts bool
	command := &cobra.Command{
		Use:   "setup [flags]",
		Short: "Create an OKF-compatible vault",
		Args:  noArgs("setup"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSetup(vaultPath, force, includeConcepts, stdout)
		},
	}
	flags := command.Flags()
	flags.StringVar(&vaultPath, "vault", defaultVault, "path to the new OKF vault")
	flags.BoolVar(&force, "force", false, "overwrite existing files")
	flags.BoolVar(&includeConcepts, "concepts", false, "include reusable project concept definitions")
	return command
}

func runSetup(vaultPath string, force, includeConcepts bool, stdout io.Writer) error {
	root := vaultPath
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	config, vaultRoots, err := vault.LoadConfig(root)
	if err != nil {
		return err
	}

	var created []string
	for _, vaultRoot := range vaultRoots {
		if err := os.MkdirAll(vaultRoot, 0o755); err != nil {
			return err
		}
		paths, err := vault.Scaffold(vaultRoot, vault.ScaffoldOptions{
			Force:        force,
			DisableIndex: !config.IndexEnabled(),
			DisableLog:   !config.LogEnabled(),
		})
		if err != nil {
			return err
		}
		created = append(created, paths...)

		if includeConcepts {
			conceptPaths, err := forge.Concepts(vaultRoot, forge.ConceptOptions{Force: force})
			if err != nil {
				return err
			}
			created = append(created, conceptPaths...)
			if config.IndexEnabled() {
				// Refresh indexes so newly written concept pages are listed.
				// The base scaffold generated indexes before the concepts
				// existed, so overwrite when concepts changed this run.
				overwrite := force || len(conceptPaths) > 0
				indexPaths, err := vault.GenerateIndexes(vaultRoot, vault.IndexOptions{Overwrite: overwrite})
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
	fmt.Fprintf(stdout, "ok: vault setup under %s\n", filepath.Clean(root))
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
		"json": true, "pretty": true, "force": true, "concepts": true,
		"type": true, "title": true,
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
