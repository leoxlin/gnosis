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
	commandVault(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", "test"}, &stdout, &stderr); err != nil {
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

func TestOmittedVaultSelectsNearestLocalConfiguration(t *testing.T) {
	workspace := commandVault(t)
	t.Chdir(workspace)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"get", "vaults"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "test,local") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestInvalidRemoteVaultSyntaxIsUsageError(t *testing.T) {
	commandVault(t)
	for _, target := range []string{
		"ftp://example.test/vault.git",
		"https://token@example.test/vault.git",
		"https://example.test/vault.git?branch=main",
	} {
		var stdout, stderr bytes.Buffer
		err := run([]string{"--vault", target, "get", "pages"}, &stdout, &stderr)
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("target %q error = %v, exit = %d", target, err, exitCode(err))
		}
	}
}

func TestCommandTreeAndHelpContract(t *testing.T) {
	expected := map[string]struct{}{
		"get vaults": {}, "get concepts": {}, "get pages": {}, "get procedures": {},
		"get history": {}, "get diff": {}, "get changes": {},
		"search knowledge": {}, "context knowledge": {}, "search memory": {}, "add memory": {},
		"graph neighbors": {}, "graph path": {}, "create vault": {},
		"plan knowledge-change": {}, "apply workspace": {}, "apply page": {},
		"apply knowledge-change": {}, "index vault": {}, "index knowledge": {},
		"validate vault": {}, "serve http": {}, "serve mcp": {},
	}
	root := newRootCommand(&bytes.Buffer{}, &bytes.Buffer{})
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		if command.Name() != "help" {
			path := strings.TrimSpace(strings.TrimPrefix(command.CommandPath(), "gnosis"))
			delete(expected, path)
			args := strings.Fields(path)
			args = append(args, "--help")
			var stdout, stderr bytes.Buffer
			if err := run(args, &stdout, &stderr); err != nil {
				t.Errorf("%s --help: %v", command.CommandPath(), err)
			} else {
				decoded, err := toon.Decode(stdout.Bytes())
				if err != nil {
					t.Errorf("decode %s help: %v", command.CommandPath(), err)
				} else {
					fields := decoded.(map[string]any)
					for _, key := range []string{"command", "description", "usage", "flags"} {
						if _, ok := fields[key]; !ok {
							t.Errorf("%s help missing %q: %#v", command.CommandPath(), key, fields)
						}
					}
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
	for path := range expected {
		t.Errorf("missing command %q", path)
	}
}

func TestRemovedCommandsAndFlagsFail(t *testing.T) {
	commandVault(t)
	for _, args := range [][]string{
		{"read", "gnosis://test/missing.md"},
		{"write", "gnosis://test/missing.md"},
		{"scaffold"},
		{"setup"},
		{"procedure", "discovery"},
		{"get", "vaults", "--vault", "test", "--json"},
		{"graph", "neighbors", "--uri", "gnosis://test/missing.md"},
		{"graph", "path", "--from", "gnosis://test/a.md", "--to", "gnosis://test/b.md"},
		{"validate", "--vault", "test"},
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
	commandVault(t)
	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--vault", "test", "graph", "neighbors", "gnosis://test/missing.md"},
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
			"--vault", "test", "graph", "path",
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
		{"removed path target", []string{"--vault", filepath.Join(t.TempDir(), "missing"), "get", "pages"}, 2, true, "configured canonical vault name"},
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
