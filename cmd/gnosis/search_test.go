package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

func TestSearchKnowledgeDefaultsToVector(t *testing.T) {
	workspace := commandVault(t)
	t.Setenv("GNOSIS_DATABASE_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_MODEL", "")

	var stdout, stderr bytes.Buffer
	err := run([]string{"search", "knowledge", "what is gnosis?", "--vault", workspace}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "GNOSIS_DATABASE_URL") {
		t.Fatalf("error = %v, want vector configuration error", err)
	}
}

func TestSearchKnowledgeLexicalUsesLiveVault(t *testing.T) {
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
		"search", "knowledge", "semantic retrieval", "--backend", "lexical",
		"--vault", workspace, "--json",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var result vault.QueryResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) == 0 || result.Candidates[0].URI != "gnosis://test/retrieval.md" {
		t.Fatalf("result = %+v", result)
	}
}

func TestSearchKnowledgeRejectsUnknownBackend(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"search", "knowledge", "question", "--backend", "other"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "backend") {
		t.Fatalf("error = %v, want backend error", err)
	}
}
