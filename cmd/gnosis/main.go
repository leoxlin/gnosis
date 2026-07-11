package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gnosis/internal/forge"
	"gnosis/internal/search"
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
		usage(stderr)
		return errors.New("missing command")
	}

	switch args[0] {
	case "query":
		return runQuery(args[1:], stdout, stderr)
	case "graph-query":
		return runGraphQuery(args[1:], stdout, stderr)
	case "index":
		return runIndex(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "setup":
		return runSetup(args[1:], stdout, stderr)
	case "version":
		if len(args) != 1 {
			return unexpectedArguments("version", args[1:])
		}
		fmt.Fprintln(stdout, "gnosis 0.1.0")
		return nil
	case "help", "-h", "--help":
		if len(args) != 1 {
			return unexpectedArguments(args[0], args[1:])
		}
		usage(stdout)
		return nil
	default:
		usage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runQuery(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("query", "gnosis query [-vault <path>] [-top <n>] [-max-read <n>] [-depth <n>] [-json] [-pretty] <question>", stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the OKF vault")
	top := fs.Int("top", 3, "number of candidate pages to return")
	maxRead := fs.Int("max-read", 3, "maximum number of pages to recommend reading")
	depth := fs.Int("depth", 3, "maximum graph traversal depth")
	jsonOutput := fs.Bool("json", false, "emit machine-readable JSON")
	pretty := fs.Bool("pretty", false, "pretty-print JSON output (implies -json)")
	question, help, err := parseQuestionFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}
	if err := validateQueryOptions(*top, *maxRead, *depth); err != nil {
		return fmt.Errorf("query: %w", err)
	}

	result, err := queryVault(*vaultPath, question, search.QueryOptions{
		Top:      *top,
		MaxRead:  *maxRead,
		MaxDepth: *depth,
	})
	if err != nil {
		return err
	}
	if *jsonOutput || *pretty {
		return writeQueryJSON(stdout, result, *pretty)
	}
	writeQueryText(stdout, result)
	return nil
}

func runGraphQuery(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("graph-query", "gnosis graph-query [-vault <path>] [-top <n>] [-max-read <n>] [-depth <n>] [-pretty] <question>", stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the OKF vault")
	top := fs.Int("top", 3, "number of candidate pages to return")
	maxRead := fs.Int("max-read", 3, "maximum number of pages to recommend reading")
	depth := fs.Int("depth", 3, "maximum graph traversal depth")
	pretty := fs.Bool("pretty", false, "pretty-print JSON output")
	question, help, err := parseQuestionFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}
	if err := validateQueryOptions(*top, *maxRead, *depth); err != nil {
		return fmt.Errorf("graph-query: %w", err)
	}

	result, err := queryVault(*vaultPath, question, search.QueryOptions{
		Top:      *top,
		MaxRead:  *maxRead,
		MaxDepth: *depth,
	})
	if err != nil {
		return err
	}
	return writeQueryJSON(stdout, result, *pretty)
}

func queryVault(root, question string, options search.QueryOptions) (search.Result, error) {
	source, err := vault.NewSearchSource(root)
	if err != nil {
		return search.Result{}, err
	}
	var documentSource search.Source = source
	documents, err := documentSource.Documents()
	if err != nil {
		return search.Result{}, err
	}
	engine := search.New(documents)
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

func writeQueryJSON(output io.Writer, result search.Result, pretty bool) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}

func writeQueryText(output io.Writer, result search.Result) {
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

func runIndex(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("index", "gnosis index [-vault <path>]", stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the OKF vault")
	help, err := parseFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}

	root := *vaultPath
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

func runValidate(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("validate", "gnosis validate [-vault <path>]", stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the OKF vault")
	help, err := parseFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}

	result, err := vault.Validate(*vaultPath)
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

func runSetup(args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet("setup", "gnosis setup [-vault <path>] [-force] [-concepts]", stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the new OKF vault")
	force := fs.Bool("force", false, "overwrite existing files")
	includeConcepts := fs.Bool("concepts", false, "include reusable project concept definitions")
	help, err := parseFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}

	root := *vaultPath
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
			Force:        *force,
			DisableIndex: !config.IndexEnabled(),
			DisableLog:   !config.LogEnabled(),
		})
		if err != nil {
			return err
		}
		created = append(created, paths...)

		if *includeConcepts {
			conceptPaths, err := forge.Concepts(vaultRoot, forge.ConceptOptions{Force: *force})
			if err != nil {
				return err
			}
			created = append(created, conceptPaths...)
			if config.IndexEnabled() {
				// Refresh indexes so newly written concept pages are listed.
				// The base scaffold generated indexes before the concepts
				// existed, so overwrite when concepts changed this run.
				overwrite := *force || len(conceptPaths) > 0
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

func newFlagSet(name, commandUsage string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s\n\nOptions:\n", commandUsage)
		fs.PrintDefaults()
	}
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string, helpOutput io.Writer) (bool, error) {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fs.SetOutput(helpOutput)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, nil
		}
		return false, err
	}
	if fs.NArg() > 0 {
		return false, unexpectedArguments(fs.Name(), fs.Args())
	}
	return false, nil
}

func parseQuestionFlags(fs *flag.FlagSet, args []string, helpOutput io.Writer) (string, bool, error) {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		fs.SetOutput(helpOutput)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return "", true, nil
		}
		return "", false, err
	}
	if fs.NArg() == 0 {
		return "", false, fmt.Errorf("%s: missing question", fs.Name())
	}
	if fs.NArg() > 1 {
		return "", false, unexpectedArguments(fs.Name(), fs.Args()[1:])
	}
	question := strings.TrimSpace(fs.Arg(0))
	if question == "" {
		return "", false, fmt.Errorf("%s: question must not be empty", fs.Name())
	}
	return question, false, nil
}

func unexpectedArguments(command string, args []string) error {
	return fmt.Errorf("%s: unexpected argument(s): %s", command, strings.Join(args, " "))
}

func usage(output io.Writer) {
	fmt.Fprintln(output, `gnosis manages an OKF-compatible Obsidian vault.

Usage:
  gnosis setup [-vault <path>] [-force] [-concepts]
  gnosis index [-vault <path>]
  gnosis validate [-vault <path>]
  gnosis query [-vault <path>] [-top <n>] [-max-read <n>] [-depth <n>] [-json] [-pretty] <question>
  gnosis graph-query [-vault <path>] [-top <n>] [-max-read <n>] [-depth <n>] [-pretty] <question>
  gnosis version`)
}
