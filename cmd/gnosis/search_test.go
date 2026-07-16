package main

import (
	"bytes"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"
)

func TestSearchKnowledgeDefaultsToVector(t *testing.T) {
	workspace := commandVault(t)
	t.Setenv("GNOSIS_DATABASE_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_MODEL", "")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--vault", workspace, "search", "knowledge", "what is gnosis?"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "GNOSIS_DATABASE_URL") {
		t.Fatalf("error = %v, want vector configuration error", err)
	}
}

func TestSearchKnowledgeLexicalUsesLiveVaultAndFields(t *testing.T) {
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "retrieval.md", `---
type: Concept
title: Semantic Retrieval
description: Finds relevant knowledge for a question.
---

Semantic retrieval returns bounded candidates.
`)

	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--vault", workspace, "search", "knowledge", "semantic retrieval",
		"--backend", "lexical", "--fields", "uri,title,score",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := toon.Decode(stdout.Bytes()); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "candidates[1]{uri,title,score}") ||
		!strings.Contains(stdout.String(), "gnosis://test/retrieval.md") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestSearchKnowledgeRejectsUnknownBackendAsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"search", "knowledge", "question", "--backend", "other"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "backend") || exitCode(err) != 2 {
		t.Fatalf("error = %v, exit = %d", err, exitCode(err))
	}
}
