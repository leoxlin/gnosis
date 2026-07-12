package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
	"io"
	"os"
	"strings"
)

func newWriteCommand(input io.Reader, stdout io.Writer) *cobra.Command {
	var filename string
	var update bool
	command := &cobra.Command{
		Use:   "write <gnosis-uri>",
		Short: "Write a typed Markdown document into the current vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			uri := strings.TrimSpace(args[0])
			if !strings.HasPrefix(uri, "gnosis://") {
				return errors.New("write: argument must be a gnosis URI")
			}
			var content []byte
			var err error
			if filename != "" {
				content, err = os.ReadFile(filename)
				if err != nil {
					return fmt.Errorf("write: read %s: %w", filename, err)
				}
			} else {
				content, err = io.ReadAll(input)
				if err != nil {
					return fmt.Errorf("write: read standard input: %w", err)
				}
			}
			path, err := vault.WriteDocument(defaultVault, uri, content, update)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, path)
			return err
		},
	}
	flags := command.Flags()
	flags.StringVar(&filename, "filename", "", "read Markdown content from this file")
	flags.BoolVar(&update, "update", false, "allow shadowing an imported or built-in document")
	return command
}
