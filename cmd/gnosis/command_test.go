package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	toon "github.com/toon-format/toon-go"
)

func TestHomeShowsLiveContext(t *testing.T) {
	workspace := commandVault(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"bin:", "description:", "counts:", "vaults[", "concept_types[", "help["} {
		if !strings.Contains(stdout.String(), key) {
			t.Fatalf("home output = %q, missing %q", stdout.String(), key)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCommandTreeUsesVerbResourceGrammar(t *testing.T) {
	for _, args := range [][]string{
		{"get", "vaults", "--help"},
		{"get", "concepts", "--help"},
		{"get", "pages", "--help"},
		{"get", "procedures", "--help"},
		{"search", "knowledge", "--help"},
		{"graph", "neighbors", "--help"},
		{"graph", "path", "--help"},
		{"create", "vault", "--help"},
		{"apply", "workspace", "--help"},
		{"apply", "page", "--help"},
		{"index", "vault", "--help"},
		{"index", "knowledge", "--help"},
		{"validate", "vault", "--help"},
		{"serve", "http", "--help"},
		{"serve", "mcp", "--help"},
	} {
		t.Run(strings.Join(args[:len(args)-1], " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(args, &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout.String(), "command:") || !strings.Contains(stdout.String(), "usage:") {
				t.Fatalf("help output = %q", stdout.String())
			}
		})
	}
}

func TestEveryCommandHelpIsContextual(t *testing.T) {
	root := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{})
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Name() != "help" {
			args := strings.Fields(strings.TrimPrefix(command.CommandPath(), "gnosis"))
			args = append(args, "--help")
			var stdout, stderr bytes.Buffer
			if err := run(args, &stdout, &stderr); err != nil {
				t.Errorf("%s --help: %v", command.CommandPath(), err)
			} else {
				if _, err := toon.Decode(stdout.Bytes()); err != nil {
					t.Errorf("decode %s help: %v", command.CommandPath(), err)
				}
				for _, key := range []string{"command:", "description:", "usage:", "flags["} {
					if !strings.Contains(stdout.String(), key) {
						t.Errorf("%s help = %q, missing %q", command.CommandPath(), stdout.String(), key)
					}
				}
				if !strings.Contains(stdout.String(), "examples[2]") &&
					!strings.Contains(stdout.String(), "examples[3]") {
					t.Errorf("%s help = %q, missing two or three examples", command.CommandPath(), stdout.String())
				}
				if stderr.Len() != 0 {
					t.Errorf("%s stderr = %q", command.CommandPath(), stderr.String())
				}
			}
		}
		for _, child := range command.Commands() {
			if child.IsAvailableCommand() {
				visit(child)
			}
		}
	}
	visit(root)
}

func TestRemovedCommandsAndFlagsFail(t *testing.T) {
	workspace := commandVault(t)
	for _, args := range [][]string{
		{"read", "gnosis://test/missing.md"},
		{"write", "gnosis://test/missing.md"},
		{"scaffold"},
		{"setup"},
		{"procedure", "discovery"},
		{"get", "vaults", "--vault", workspace, "--json"},
		{"graph", "neighbors", "--uri", "gnosis://test/missing.md"},
		{"graph", "path", "--from", "gnosis://test/a.md", "--to", "gnosis://test/b.md"},
		{"validate", "--vault", workspace},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(args, &stdout, &stderr); err == nil {
				t.Fatalf("run(%q) succeeded", args)
			}
		})
	}
}

func TestGraphIdentitiesArePositional(t *testing.T) {
	workspace := commandVault(t)
	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--vault", workspace, "graph", "neighbors", "gnosis://test/missing.md"},
		&stdout,
		&stderr,
	)
	if err == nil || strings.Contains(err.Error(), "accepts 0 arg") {
		t.Fatalf("error = %v, want lookup error after positional argument validation", err)
	}

	stdout.Reset()
	stderr.Reset()
	err = run(
		[]string{
			"--vault", workspace, "graph", "path",
			"gnosis://test/a.md", "gnosis://test/b.md",
		},
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGraphRejectsInvalidIdentityAndOptionValuesAsUsage(t *testing.T) {
	for _, args := range [][]string{
		{"graph", "neighbors", "not-a-uri"},
		{"graph", "neighbors", "gnosis://test/page.md?query=1"},
		{"graph", "neighbors", "gnosis://test/page.md", "--direction", "sideways"},
		{"graph", "path", "gnosis://test/a.md", "gnosis://test/b.md", "--depth", "-1"},
	} {
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("run(%q) error = %v, exit = %d", args, err, exitCode(err))
		}
	}
}

func TestProcessExitAndChannelContract(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gnosis")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gnosis: %v\n%s", err, output)
	}

	tests := []struct {
		name       string
		args       []string
		exit       int
		wantTOON   bool
		wantOutput string
	}{
		{"success", []string{"version"}, 0, true, "version:"},
		{"usage", []string{"get", "vaults", "--json"}, 2, true, "Valid flags:"},
		{"runtime", []string{"--vault", filepath.Join(t.TempDir(), "missing"), "get", "pages"}, 1, true, "error:"},
		{"completion", []string{"completion", "bash"}, 0, false, "bash completion"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(binary, test.args...)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			gotExit := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatal(err)
				}
				gotExit = exitErr.ExitCode()
			}
			if gotExit != test.exit {
				t.Fatalf("exit = %d, want %d; stdout=%q stderr=%q", gotExit, test.exit, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("stdout = %q, missing %q", stdout.String(), test.wantOutput)
			}
			if test.wantTOON {
				if _, err := toon.Decode(stdout.Bytes()); err != nil {
					t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}
