package main

import (
	"bytes"
	"strings"
	"testing"

	"gnosis/internal/search"
)

func TestIndexRequiresResource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"index"}, &stdout, &stderr); err == nil || exitCode(err) != 2 {
		t.Fatalf("bare index error = %v", err)
	}
}

func TestIndexVaultPreservesDisabledState(t *testing.T) {
	workspace := commandVault(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--vault", workspace, "index", "vault"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"action: index", "resource: vault", "status: disabled", "changed: false"} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestWriteSemanticIndexResultUsesTOON(t *testing.T) {
	result := search.SemanticIndexResult{
		Documents:   2,
		Chunks:      3,
		Scope:       "scope",
		Fingerprint: "fingerprint",
	}

	var output bytes.Buffer
	if err := writeSemanticIndexResult(&output, result); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{
		"action: index", "resource: knowledge", "status: synchronized",
		"documents: 2", "chunks: 3", "scope: scope", "fingerprint: fingerprint",
	} {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("output = %q, missing %q", output.String(), value)
		}
	}
}
