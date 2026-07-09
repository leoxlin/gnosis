package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"gnosis/internal/vault"
)

const defaultVault = "."

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return errors.New("missing command")
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:])
	case "scaffold":
		return runScaffold(args[1:])
	case "setup":
		return runSetup(args[1:])
	case "version":
		fmt.Println("gnosis 0.1.0")
		return nil
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the Gnosis vault")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := vault.Validate(*vaultPath)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Printf("warning: %s\n", warning)
	}
	if len(result.Errors) > 0 {
		for _, validationErr := range result.Errors {
			fmt.Printf("error: %s\n", validationErr)
		}
		return fmt.Errorf("validation failed: %d error(s)", len(result.Errors))
	}
	fmt.Printf("ok: %d markdown file(s) validated\n", result.FilesChecked)
	return nil
}

func runScaffold(args []string) error {
	fs := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the Gnosis vault")
	force := fs.Bool("force", false, "overwrite existing scaffold files")
	if err := fs.Parse(args); err != nil {
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
		paths, err := vault.Scaffold(vaultRoot, vault.ScaffoldOptions{Force: *force})
		if err != nil {
			return err
		}
		created = append(created, paths...)
	}
	for _, path := range created {
		fmt.Println(path)
	}
	fmt.Printf("ok: scaffold checked under %s\n", filepath.Clean(root))
	return nil
}

func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	vaultPath := fs.String("vault", defaultVault, "path to the new Gnosis vault")
	force := fs.Bool("force", false, "overwrite existing scaffold files")
	if err := fs.Parse(args); err != nil {
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
		paths, err := vault.Scaffold(vaultRoot, vault.ScaffoldOptions{Force: *force})
		if err != nil {
			return err
		}
		created = append(created, paths...)
	}
	for _, path := range created {
		fmt.Println(path)
	}
	fmt.Printf("ok: vault setup under %s\n", filepath.Clean(root))
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, `gnosis manages an OKF-compatible Obsidian vault.

Usage:
  gnosis setup [-vault <path>] [-force]
  gnosis validate [-vault <path>]
  gnosis scaffold [-vault <path>] [-force]
  gnosis version`)
}
