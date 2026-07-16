package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	toon "github.com/toon-format/toon-go"
)

const detailPreviewLimit = 1000

type fieldSelector struct {
	names []string
}

func parseFields(raw string, defaults, allowed []string) (fieldSelector, error) {
	if raw == "" {
		return fieldSelector{names: append([]string{}, defaults...)}, nil
	}

	names := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(names))
	for i, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			return fieldSelector{}, errors.New("--fields must not contain an empty field")
		}
		if !slices.Contains(allowed, name) {
			return fieldSelector{}, fmt.Errorf(
				"unknown field %q; valid fields: %s",
				name,
				strings.Join(allowed, ", "),
			)
		}
		if _, exists := seen[name]; exists {
			return fieldSelector{}, fmt.Errorf("duplicate field %q", name)
		}
		seen[name] = struct{}{}
		names[i] = name
	}
	return fieldSelector{names: names}, nil
}

func (s fieldSelector) object(value func(string) (any, bool)) toon.Object {
	fields := make([]toon.Field, 0, len(s.names))
	for _, name := range s.names {
		current, ok := value(name)
		if !ok {
			continue
		}
		fields = append(fields, toon.Field{Key: name, Value: current})
	}
	return toon.NewObject(fields...)
}

func listOutput(
	name string,
	total int,
	rows []toon.Object,
	emptyMessage string,
	help []string,
) toon.Object {
	fields := []toon.Field{
		{Key: "count", Value: total},
		{Key: name, Value: rows},
	}
	if total == 0 {
		fields = append(fields, toon.Field{Key: "message", Value: emptyMessage})
	}
	if len(help) > 0 {
		fields = append(fields, toon.Field{Key: "help", Value: help})
	}
	return toon.NewObject(fields...)
}

func writeTOON(output io.Writer, value toon.Object) error {
	data, err := toon.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode toon: %w", err)
	}
	if _, err := output.Write(data); err != nil {
		return fmt.Errorf("write toon: %w", err)
	}
	return nil
}

func truncate(content string, full bool) (string, int, bool) {
	total := utf8.RuneCountInString(content)
	if full || total <= detailPreviewLimit {
		return content, total, false
	}
	runes := []rune(content)
	return string(runes[:detailPreviewLimit]), total, true
}

func executablePath() string {
	path, err := os.Executable()
	if err != nil {
		return "gnosis"
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "gnosis"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(path)
	}
	home = filepath.Clean(home)
	path = filepath.Clean(path)
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

type usageError struct {
	cause error
}

func (e *usageError) Error() string { return e.cause.Error() }
func (e *usageError) Unwrap() error { return e.cause }

func newUsageError(err error) error {
	if err == nil {
		return nil
	}
	return &usageError{cause: err}
}

type commandError struct {
	cause      error
	isUsage    bool
	command    string
	usage      string
	validFlags []string
}

func (e *commandError) Error() string { return e.cause.Error() }
func (e *commandError) Unwrap() error { return e.cause }

func wrapCommandError(command *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	return &commandError{
		cause:      err,
		isUsage:    isUsageFailure(err),
		command:    command.CommandPath(),
		usage:      command.UseLine(),
		validFlags: validFlagNames(command),
	}
}

func isUsageFailure(err error) bool {
	var usage *usageError
	if errors.As(err, &usage) {
		return true
	}
	message := err.Error()
	for _, prefix := range []string{
		"unknown command",
		"unknown flag",
		"unknown shorthand flag",
		"flag needs an argument",
		"required flag",
		"invalid argument",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return strings.Contains(message, " arg(s)")
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var commandErr *commandError
	if errors.As(err, &commandErr) && commandErr.isUsage {
		return 2
	}
	return 1
}

func writeCommandError(output io.Writer, err error) error {
	fields := []toon.Field{{Key: "error", Value: err.Error()}}
	var commandErr *commandError
	if errors.As(err, &commandErr) && commandErr.isUsage {
		help := []string{"Usage: " + commandErr.usage}
		if len(commandErr.validFlags) > 0 {
			help = append(help, "Valid flags: "+strings.Join(commandErr.validFlags, ", "))
		}
		fields = append(fields, toon.Field{Key: "help", Value: help})
	}
	return writeTOON(output, toon.NewObject(fields...))
}

func validFlagNames(command *cobra.Command) []string {
	names := []string{"--help"}
	visit := func(flag *pflag.Flag) {
		name := "--" + flag.Name
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	command.NonInheritedFlags().VisitAll(visit)
	command.InheritedFlags().VisitAll(visit)
	sort.Strings(names)
	return names
}

func setCommandHelp(root *cobra.Command) {
	root.SetHelpFunc(func(command *cobra.Command, _ []string) {
		if err := writeTOON(command.OutOrStdout(), commandHelp(command)); err != nil {
			fmt.Fprintln(command.ErrOrStderr(), err)
		}
	})
}

func commandHelp(command *cobra.Command) toon.Object {
	fields := []toon.Field{
		{Key: "command", Value: command.CommandPath()},
		{Key: "description", Value: command.Short},
		{Key: "usage", Value: command.UseLine()},
	}

	commands := commandRows(command)
	if len(commands) > 0 {
		fields = append(fields, toon.Field{Key: "commands", Value: commands})
	}
	flags := flagRows(command)
	if len(flags) > 0 {
		fields = append(fields, toon.Field{Key: "flags", Value: flags})
	}
	examples := exampleLines(command.Example)
	if len(examples) > 0 {
		fields = append(fields, toon.Field{Key: "examples", Value: examples})
	}
	return toon.NewObject(fields...)
}

func commandRows(command *cobra.Command) []toon.Object {
	rows := []toon.Object{}
	for _, child := range command.Commands() {
		if !child.IsAvailableCommand() || child.Name() == "help" {
			continue
		}
		rows = append(rows, toon.NewObject(
			toon.Field{Key: "command", Value: child.Name()},
			toon.Field{Key: "description", Value: child.Short},
		))
	}
	return rows
}

func flagRows(command *cobra.Command) []toon.Object {
	flags := map[string]*pflag.Flag{}
	appendFlag := func(flag *pflag.Flag) {
		flags[flag.Name] = flag
	}
	command.NonInheritedFlags().VisitAll(appendFlag)
	command.InheritedFlags().VisitAll(appendFlag)
	names := make([]string, 0, len(flags))
	for name := range flags {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]toon.Object, 0, len(names))
	for _, name := range names {
		flag := flags[name]
		rows = append(rows, toon.NewObject(
			toon.Field{Key: "name", Value: "--" + flag.Name},
			toon.Field{Key: "default", Value: flag.DefValue},
			toon.Field{Key: "description", Value: flag.Usage},
		))
	}
	return rows
}

func exampleLines(examples string) []string {
	lines := []string{}
	for line := range strings.SplitSeq(examples, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
