package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	case "index":
		return runIndex(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "scaffold":
		return runScaffold(args[1:], stdout, stderr)
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
	_, vaultRoots, err := vault.LoadConfig(root)
	if err != nil {
		return err
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

func runScaffold(args []string, stdout, stderr io.Writer) error {
	return runScaffoldCommand("scaffold", "path to the OKF vault", "scaffold checked", args, stdout, stderr)
}

func runSetup(args []string, stdout, stderr io.Writer) error {
	return runScaffoldCommand("setup", "path to the new OKF vault", "vault setup", args, stdout, stderr)
}

func runScaffoldCommand(name, vaultDescription, success string, args []string, stdout, stderr io.Writer) error {
	fs := newFlagSet(name, fmt.Sprintf("gnosis %s [-vault <path>] [-force] [-concepts]", name), stderr)
	vaultPath := fs.String("vault", defaultVault, vaultDescription)
	force := fs.Bool("force", false, "overwrite existing scaffold files")
	includeConcepts := fs.Bool("concepts", false, "include reusable project concept definitions")
	help, err := parseFlags(fs, args, stdout)
	if err != nil || help {
		return err
	}

	root := *vaultPath
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	_, vaultRoots, err := vault.LoadConfig(root)
	if err != nil {
		return err
	}

	var created []string
	for _, vaultRoot := range vaultRoots {
		if err := os.MkdirAll(vaultRoot, 0o755); err != nil {
			return err
		}
		paths, err := vault.Scaffold(vaultRoot, vault.ScaffoldOptions{
			Force:           *force,
			IncludeConcepts: *includeConcepts,
		})
		if err != nil {
			return err
		}
		created = append(created, paths...)
	}
	for _, path := range created {
		fmt.Fprintln(stdout, path)
	}
	fmt.Fprintf(stdout, "ok: %s under %s\n", success, filepath.Clean(root))
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

func unexpectedArguments(command string, args []string) error {
	return fmt.Errorf("%s: unexpected argument(s): %s", command, strings.Join(args, " "))
}

func usage(output io.Writer) {
	fmt.Fprintln(output, `gnosis manages an OKF-compatible Obsidian vault.

Usage:
  gnosis setup [-vault <path>] [-force] [-concepts]
  gnosis index [-vault <path>]
  gnosis validate [-vault <path>]
  gnosis scaffold [-vault <path>] [-force] [-concepts]
  gnosis version`)
}
