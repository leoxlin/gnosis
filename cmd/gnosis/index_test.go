package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

func TestIndexRequiresResource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"index"}, &stdout, &stderr); err == nil {
		t.Fatal("bare index succeeded")
	}
}

func TestIndexVaultPreservesGeneratedIndexBehavior(t *testing.T) {
	workspace := commandVault(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"index", "vault", "--vault", workspace}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); !strings.Contains(got, "ok: index disabled under "+workspace) {
		t.Fatalf("output = %q", got)
	}
}

func TestWriteSemanticIndexResult(t *testing.T) {
	result := vault.SemanticIndexResult{
		Documents:   2,
		Chunks:      3,
		Scope:       "scope",
		Fingerprint: "fingerprint",
	}

	var output bytes.Buffer
	if err := writeSemanticIndexResult(&output, result, false); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"documents: 2", "chunks: 3", "scope: scope", "fingerprint: fingerprint",
	} {
		if !strings.Contains(output.String(), line+"\n") {
			t.Fatalf("text output = %q, missing %q", output.String(), line)
		}
	}

	output.Reset()
	if err := writeSemanticIndexResult(&output, result, true); err != nil {
		t.Fatal(err)
	}
	var decoded vault.SemanticIndexResult
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != result {
		t.Fatalf("decoded = %+v, want %+v", decoded, result)
	}
}
