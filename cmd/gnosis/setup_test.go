package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupGitHubWiki(t *testing.T) {
	workspace := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{"setup", "--vault", workspace, "--github-wiki", "OWNER/REPOSITORY", "--name", "wiki"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(workspace, "gnosis.toml"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[vault]
vault_name = "wiki"
backend = "github-wiki"
repository = "OWNER/REPOSITORY"
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true

[gnosis]
processes = ["vault"]
`
	if string(content) != want {
		t.Fatalf("gnosis.toml = %q, want %q", content, want)
	}
}

func TestSetupGitHubWikiRejectsInvalidFlagCombinations(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"missing name", []string{"setup", "--github-wiki", "OWNER/REPOSITORY"}, "--name is required"},
		{"mixed import", []string{"setup", "--github-wiki", "OWNER/REPOSITORY", "--name", "wiki", "--import", "local"}, "cannot be combined"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(test.args, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}
