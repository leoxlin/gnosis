package vault

import (
	"path/filepath"
	"strings"
)

func isOpenSpecArtifact(relativePath string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(relativePath)), "/")
	if len(parts) == 4 &&
		parts[0] == "openspec" &&
		parts[1] == "specs" &&
		parts[2] != "" &&
		parts[3] == "spec.md" {
		return true
	}

	if len(parts) < 4 || parts[0] != "openspec" || parts[1] != "changes" {
		return false
	}
	changeIndex := 2
	if parts[changeIndex] == "archive" {
		changeIndex++
	}
	if changeIndex >= len(parts) || parts[changeIndex] == "" {
		return false
	}
	artifact := parts[changeIndex+1:]

	if len(artifact) == 1 {
		switch artifact[0] {
		case "proposal.md", "design.md", "tasks.md":
			return true
		}
	}

	return len(artifact) == 3 &&
		artifact[0] == "specs" &&
		artifact[1] != "" &&
		artifact[2] == "spec.md"
}
