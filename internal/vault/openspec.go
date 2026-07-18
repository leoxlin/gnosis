package vault

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	openSpecSpecType     = "OpenSpecSpec"
	openSpecProposalType = "OpenSpecProposal"
	openSpecDesignType   = "OpenSpecDesign"
	openSpecTasksType    = "OpenSpecTasks"
)

func projectedOpenSpecPage(root, path string, data []byte) (parsedPage, pageMetadata, bool) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return parsedPage{}, pageMetadata{}, false
	}
	metadata, ok := openSpecArtifactMetadata(filepath.ToSlash(relative))
	if !ok {
		return parsedPage{}, pageMetadata{}, false
	}
	fields := frontmatterFields{
		"type":        metadata.conceptType,
		"title":       metadata.title,
		"description": metadata.description,
	}
	if len(metadata.tags) > 0 {
		tags := make([]any, len(metadata.tags))
		for index, tag := range metadata.tags {
			tags[index] = tag
		}
		fields["tags"] = tags
	}
	return parsedPage{fields: fields, body: string(data)}, metadata, true
}

func openSpecArtifactMetadata(relativePath string) (pageMetadata, bool) {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(relativePath)), "/")
	if len(parts) == 4 &&
		parts[0] == "openspec" &&
		parts[1] == "specs" &&
		parts[2] != "" &&
		parts[3] == "spec.md" {
		capability := parts[2]
		return pageMetadata{
			conceptType: openSpecSpecType,
			title:       humanizeName(capability) + " Specification",
			description: fmt.Sprintf("Current OpenSpec specification for %s.", capability),
			tags:        []string{"openspec", "spec"},
		}, true
	}

	if len(parts) < 4 || parts[0] != "openspec" || parts[1] != "changes" {
		return pageMetadata{}, false
	}
	changeIndex := 2
	if parts[changeIndex] == "archive" {
		changeIndex++
	}
	if changeIndex >= len(parts) || parts[changeIndex] == "" {
		return pageMetadata{}, false
	}
	change := parts[changeIndex]
	artifact := parts[changeIndex+1:]

	if len(artifact) == 1 {
		title := humanizeName(change)
		switch artifact[0] {
		case "proposal.md":
			return pageMetadata{
				conceptType: openSpecProposalType,
				title:       title + " Proposal",
				description: fmt.Sprintf("OpenSpec proposal for change %s.", change),
				tags:        []string{"openspec", "change", "proposal"},
			}, true
		case "design.md":
			return pageMetadata{
				conceptType: openSpecDesignType,
				title:       title + " Design",
				description: fmt.Sprintf("OpenSpec design for change %s.", change),
				tags:        []string{"openspec", "change", "design"},
			}, true
		case "tasks.md":
			return pageMetadata{
				conceptType: openSpecTasksType,
				title:       title + " Tasks",
				description: fmt.Sprintf("OpenSpec implementation tasks for change %s.", change),
				tags:        []string{"openspec", "change", "tasks"},
			}, true
		}
	}

	if len(artifact) == 3 &&
		artifact[0] == "specs" &&
		artifact[1] != "" &&
		artifact[2] == "spec.md" {
		capability := artifact[1]
		return pageMetadata{
			conceptType: openSpecSpecType,
			title:       humanizeName(change) + " " + humanizeName(capability) + " Specification",
			description: fmt.Sprintf("OpenSpec delta specification for %s in change %s.", capability, change),
			tags:        []string{"openspec", "change", "spec"},
		}, true
	}
	return pageMetadata{}, false
}
