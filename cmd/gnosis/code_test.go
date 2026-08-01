package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCodeCommandsRejectMissingInputsWithoutSideEffects(t *testing.T) {
	commandVault(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	cases := [][]string{
		{"parsers"},
		{"parsers", "install"},
		{"parsers", "install", "go"},
		{"index", "code"},
		{"search", "code", "Handler"},
		{"get", "code-symbol", "id"},
		{"get", "code-diagnostics"},
		{"graph", "code", "id", "--direction", "sideways", "--scope", "app"},
		{"parsers", "status", "--unknown"},
	}
	for _, arguments := range cases {
		var stdout, stderr bytes.Buffer
		err := run(arguments, &stdout, &stderr)
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("run(%v) = %v (exit %d), stdout %q, stderr %q", arguments, err, exitCode(err), stdout.String(), stderr.String())
		}
	}
}

func TestParserStatusHasDefinitiveEmptyStateAndTotals(t *testing.T) {
	commandVault(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var stdout, stderr bytes.Buffer
	if err := run([]string{"parsers", "status", "go", "typescript"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"count: 2", "total: 2", "go,false", "typescript,false"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("output = %q, missing %q", stdout.String(), expected)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
