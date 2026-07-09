package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := run([]string{"version"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "gnosis 0.1.0\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunMissingCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err == nil || err.Error() != "missing command" {
		t.Fatalf("error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
