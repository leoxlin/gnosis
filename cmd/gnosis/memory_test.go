package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"
	agentmemory "gnosis/internal/memory"
)

func TestMemoryCLIUsesVaultBackend(t *testing.T) {
	workspace := commandVault(t)
	clearCommandMemoryEnv(t)
	t.Setenv(agentmemory.EnvUserID, "user")
	t.Setenv(agentmemory.EnvAgentID, "agent")

	var stdout, stderr bytes.Buffer
	if err := run(
		[]string{"--vault", workspace, "add", "memory", "I prefer dark mode"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := toon.Decode(stdout.Bytes()); err != nil {
		t.Fatalf("decode add output: %v\n%s", err, stdout.String())
	}
	for _, want := range []string{"event: ADD", "backend: vault", "created_at:", "updated_at:"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("add output = %q, missing %q", stdout.String(), want)
		}
	}

	stdout.Reset()
	if err := run(
		[]string{"--vault", workspace, "search", "memory", "dark mode", "--limit", "1"},
		&stdout,
		&stderr,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := toon.Decode(stdout.Bytes()); err != nil {
		t.Fatalf("decode search output: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "I prefer dark mode") ||
		!strings.Contains(stdout.String(), "backend: vault") ||
		!strings.Contains(stdout.String(), "score:") {
		t.Fatalf("search output = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestMemoryCLIRejectsUsageBeforeMutation(t *testing.T) {
	workspace := commandVault(t)
	clearCommandMemoryEnv(t)
	t.Setenv(agentmemory.EnvUserID, "user")
	t.Setenv(agentmemory.EnvAgentID, "agent")

	for _, args := range [][]string{
		{"--vault", workspace, "add", "memory"},
		{"--vault", workspace, "add", "memory", "one", "two"},
		{"--vault", workspace, "add", "memory", " \t"},
		{"--vault", workspace, "search", "memory"},
		{"--vault", workspace, "search", "memory", "one", "two"},
		{"--vault", workspace, "search", "memory", "query", "--limit", "0"},
		{"--vault", workspace, "search", "memory", "query", "--limit", "21"},
	} {
		var stdout, stderr bytes.Buffer
		err := run(args, &stdout, &stderr)
		if err == nil || exitCode(err) != 2 {
			t.Fatalf("run(%q) error = %v, exit = %d", args, err, exitCode(err))
		}
	}
	matches, err := filepath.Glob(filepath.Join(workspace, "memories", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("invalid commands wrote %v", matches)
	}
}

func TestMemoryCLIConfigurationFailureIsRuntimeError(t *testing.T) {
	workspace := commandVault(t)
	clearCommandMemoryEnv(t)
	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--vault", workspace, "add", "memory", "remember this"},
		&stdout,
		&stderr,
	)
	if err == nil || exitCode(err) != 1 ||
		!strings.Contains(err.Error(), agentmemory.EnvUserID) {
		t.Fatalf("error = %v, exit = %d", err, exitCode(err))
	}
	if _, statErr := os.Stat(filepath.Join(workspace, "memories")); !os.IsNotExist(statErr) {
		t.Fatalf("memory directory stat error = %v", statErr)
	}
}

func clearCommandMemoryEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		agentmemory.EnvAPIKey,
		agentmemory.EnvUserID,
		agentmemory.EnvAgentID,
		agentmemory.EnvProvider,
		agentmemory.EnvBaseURL,
	} {
		t.Setenv(name, "")
	}
}
