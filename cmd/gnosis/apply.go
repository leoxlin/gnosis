package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
	"gnosis/internal/vault"
)

func newApplyCommand(options *rootOptions, input io.Reader, stdout io.Writer) *cobra.Command {
	command := &cobra.Command{
		Use:     "apply",
		Short:   "Apply vault resources and workspace configuration",
		Args:    cobra.NoArgs,
		GroupID: "basic",
		Example: "gnosis apply page <gnosis-uri> --filename <file>\n" +
			"gnosis apply workspace --import <path>",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("apply: missing resource"))
		},
	}
	command.AddCommand(
		newApplyWorkspaceCommand(options, stdout),
		newApplyPageCommand(options, input, stdout),
		newApplyKnowledgeChangeCommand(options, input, stdout),
	)
	return command
}

func newApplyWorkspaceCommand(options *rootOptions, stdout io.Writer) *cobra.Command {
	var githubWiki, vaultName string
	var imports []string
	var isForce bool
	command := &cobra.Command{
		Use:   "workspace [flags]",
		Short: "Apply gnosis workspace configuration",
		Args:  cobra.NoArgs,
		Example: "gnosis apply workspace --import <path>\n" +
			"gnosis apply workspace --import <path-a> --import <path-b>\n" +
			"gnosis apply workspace --github-wiki <owner/repository> --name <name>",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateWorkspaceOptions(imports, githubWiki, vaultName); err != nil {
				return newUsageError(err)
			}
			return runApplyWorkspace(
				options.vaultPath,
				imports,
				githubWiki,
				vaultName,
				isForce,
				stdout,
			)
		},
	}
	flags := command.Flags()
	flags.StringSliceVar(&imports, "import", nil, "path of a vault to import")
	flags.StringVar(&githubWiki, "github-wiki", "", "GitHub owner/repository for the primary vault")
	flags.StringVar(&vaultName, "name", "", "canonical name for the primary vault")
	flags.BoolVar(&isForce, "force", false, "overwrite existing gnosis.toml")
	return command
}

func validateWorkspaceOptions(imports []string, githubWiki, vaultName string) error {
	if githubWiki != "" && len(imports) > 0 {
		return errors.New("apply workspace: --github-wiki cannot be combined with --import")
	}
	if githubWiki != "" && vaultName == "" {
		return errors.New("apply workspace: --name is required with --github-wiki")
	}
	if githubWiki == "" && vaultName != "" {
		return errors.New("apply workspace: --name requires --github-wiki")
	}
	if githubWiki == "" && len(imports) == 0 {
		return errors.New("apply workspace: at least one --import is required")
	}
	return nil
}

func runApplyWorkspace(
	vaultPath string,
	imports []string,
	githubWiki string,
	vaultName string,
	isForce bool,
	stdout io.Writer,
) error {
	var isChanged bool
	var configPath string
	var err error
	if githubWiki != "" {
		isChanged, configPath, err = vault.WriteGitHubWikiConfig(vaultPath, vaultName, githubWiki, isForce)
	} else {
		isChanged, configPath, err = vault.WriteWorkspaceConfig(vaultPath, imports, isForce)
	}
	if err != nil {
		return fmt.Errorf("apply workspace: %w", err)
	}
	status := "configured"
	if !isChanged {
		status = "no-op"
	}
	return writeTOON(stdout, toon.NewObject(
		toon.Field{Key: "action", Value: "apply"},
		toon.Field{Key: "resource", Value: "workspace"},
		toon.Field{Key: "status", Value: status},
		toon.Field{Key: "changed", Value: isChanged},
		toon.Field{Key: "path", Value: configPath},
	))
}

func newApplyPageCommand(
	options *rootOptions,
	input io.Reader,
	stdout io.Writer,
) *cobra.Command {
	var filename string
	var isUpdate bool
	command := &cobra.Command{
		Use:   "page <gnosis-uri> [flags]",
		Short: "Apply one typed Markdown page to the current vault",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis apply page <gnosis-uri> --filename <file>\n" +
			"gnosis apply page <gnosis-uri> -f <file> --update\n" +
			"gnosis apply page <gnosis-uri> < <file>",
		RunE: func(command *cobra.Command, args []string) error {
			uri := strings.TrimSpace(args[0])
			if !vault.IsCanonicalURI(uri) {
				return newUsageError(errors.New("apply page: argument must be a gnosis uri"))
			}
			content, err := readPageInput(input, filename)
			if err != nil {
				return err
			}
			if existing, readErr := vault.ReadPage(options.vaultPath, uri); readErr == nil {
				raw, rawErr := os.ReadFile(existing.Document.Origin.Path)
				if rawErr != nil {
					raw = []byte(existing.Markdown)
				}
				if bytes.Equal(raw, content) {
					return writeTOON(stdout, toon.NewObject(
						toon.Field{Key: "action", Value: "apply"},
						toon.Field{Key: "resource", Value: "page"},
						toon.Field{Key: "status", Value: "no-op"},
						toon.Field{Key: "uri", Value: uri},
						toon.Field{Key: "path", Value: existing.Document.Origin.Path},
						toon.Field{Key: "changed", Value: false},
					))
				}
			}
			result, err := vault.WriteDocument(command.Context(), options.vaultPath, uri, content, isUpdate)
			if err != nil {
				return fmt.Errorf("apply page: %w", err)
			}
			fields := []toon.Field{
				toon.Field{Key: "action", Value: "apply"},
				toon.Field{Key: "resource", Value: "page"},
				toon.Field{Key: "status", Value: "applied"},
				toon.Field{Key: "uri", Value: uri},
				toon.Field{Key: "path", Value: result.Path},
				toon.Field{Key: "revision", Value: result.Revision},
				toon.Field{Key: "changed", Value: result.Changed},
			}
			if len(result.Deliveries) > 0 {
				fields = append(fields, toon.Field{Key: "deliveries", Value: hookDeliveryObjects(result.Deliveries)})
			}
			return writeTOON(stdout, toon.NewObject(fields...))
		},
	}
	flags := command.Flags()
	flags.StringVarP(&filename, "filename", "f", "", "read Markdown content from this file")
	flags.BoolVar(&isUpdate, "update", false, "allow shadowing a lower-precedence page")
	return command
}

func readPageInput(input io.Reader, filename string) ([]byte, error) {
	return readMarkdownInput(input, filename, "apply page")
}

func readMarkdownInput(input io.Reader, filename, operation string) ([]byte, error) {
	if filename == "" {
		content, err := io.ReadAll(input)
		if err != nil {
			return nil, fmt.Errorf("%s: read standard input: %w", operation, err)
		}
		return content, nil
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("%s: read %s: %w", operation, filename, err)
	}
	return content, nil
}

func newApplyKnowledgeChangeCommand(
	options *rootOptions,
	input io.Reader,
	stdout io.Writer,
) *cobra.Command {
	var filename, expectedRevision, digest string
	var expectedAbsent bool
	command := &cobra.Command{
		Use:   "knowledge-change <gnosis-uri> [flags]",
		Short: "Apply one accepted revision-bound knowledge change",
		Args:  cobra.ExactArgs(1),
		Example: "gnosis apply knowledge-change <gnosis-uri> --expected-revision <revision> --digest <digest> --filename <file>\n" +
			"gnosis apply knowledge-change <gnosis-uri> --expected-absent --digest <digest> < <file>",
		RunE: func(command *cobra.Command, args []string) error {
			change, err := knowledgeChangeInput(
				input,
				filename,
				args[0],
				expectedRevision,
				expectedAbsent,
				"apply knowledge-change",
			)
			if err != nil {
				return err
			}
			result, err := vault.ApplyKnowledgeChange(command.Context(), options.vaultPath, change, digest)
			if err != nil {
				return err
			}
			return writeKnowledgeChangeResult(stdout, result)
		},
	}
	flags := command.Flags()
	flags.StringVarP(&filename, "filename", "f", "", "read complete Markdown candidate from this file")
	flags.StringVar(&expectedRevision, "expected-revision", "", "required current revision for an update or archive")
	flags.BoolVar(&expectedAbsent, "expected-absent", false, "require the target page to be absent")
	flags.StringVar(&digest, "digest", "", "digest returned by plan knowledge-change")
	_ = command.MarkFlagRequired("digest")
	return command
}
