package vault

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// DirectiveType is the intent handoff concept type.
const DirectiveType = "Directive"

var directiveStatuses = []string{"draft", "open", "blocked", "done"}

var directiveTaskHeading = regexp.MustCompile(`^Task ([1-9][0-9]*): \S`)

// DirectiveSummary is one Directive record with its derived task progress.
type DirectiveSummary struct {
	DocumentRef
	Status     string `json:"status"`
	TasksTotal int    `json:"tasks_total"`
	TasksDone  int    `json:"tasks_done"`
}

// Directives lists every effective Directive record with derived progress.
func Directives(root string) ([]DirectiveSummary, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.validationPages()
	if err != nil {
		return nil, err
	}
	summaries := make([]DirectiveSummary, 0)
	for _, page := range pages {
		if page.document.Type != DirectiveType {
			continue
		}
		status, _ := frontmatterScalar(page.fields, "status")
		done, total := directiveTaskProgress(page.document.Body)
		summaries = append(summaries, DirectiveSummary{
			DocumentRef: page.document.Ref(),
			Status:      strings.TrimSpace(status),
			TasksTotal:  total,
			TasksDone:   done,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].URI < summaries[j].URI
	})
	return summaries, nil
}

// directiveTaskProgress counts checkbox tasks outside code fences.
func directiveTaskProgress(body string) (done, total int) {
	fence := fenceState{}
	for _, line := range strings.Split(body, "\n") {
		if fence.advance(line) {
			continue
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "- [ ]"):
			total++
		case strings.HasPrefix(trimmed, "- [x]"), strings.HasPrefix(trimmed, "- [X]"):
			total++
			done++
		}
	}
	return done, total
}

// parseDirective checks the Directive contract and returns every problem.
func parseDirective(fields frontmatterFields, body string) []string {
	problems := []string{}
	status, scalar := frontmatterScalar(fields, "status")
	status = strings.TrimSpace(status)
	if !scalar || status == "" {
		problems = append(problems, "directive requires non-empty \"status\" frontmatter")
	} else if !slices.Contains(directiveStatuses, status) {
		problems = append(problems, fmt.Sprintf("directive status %q must be one of: %s", status, strings.Join(directiveStatuses, ", ")))
	}

	sections, duplicates := markdownSectionsAtLevel(body, 1)
	for _, name := range duplicates {
		problems = append(problems, fmt.Sprintf("duplicate section %q", name))
	}
	for _, name := range []string{"Goal", "Scope", "Acceptance criteria"} {
		if strings.TrimSpace(sections[name]) == "" {
			problems = append(problems, fmt.Sprintf("missing required section %q", name))
		}
	}
	problems = append(problems, directivePlanProblems(sections["Implementation plan"])...)
	problems = append(problems, directiveDeltaProblems(sections["Purpose/Decision Changes"])...)
	problems = append(problems, directiveScenarioProblems(sections["Acceptance criteria"])...)
	return problems
}

func directivePlanProblems(plan string) []string {
	if strings.TrimSpace(plan) == "" {
		return nil
	}
	blocks := markdownSectionBlocks(plan, 3)
	if len(blocks) == 0 {
		return []string{"implementation plan requires at least one \"### Task <N>:\" section"}
	}
	problems := []string{}
	for index, block := range blocks {
		match := directiveTaskHeading.FindStringSubmatch(block.Title)
		if match == nil {
			problems = append(problems, fmt.Sprintf("invalid task heading %q; want \"Task <N>: <deliverable>\"", block.Title))
			continue
		}
		if match[1] != strconv.Itoa(index+1) {
			problems = append(problems, fmt.Sprintf("task %q is out of order; expected Task %d", block.Title, index+1))
		}
		_, total := directiveTaskProgress(block.Body)
		if total == 0 {
			problems = append(problems, fmt.Sprintf("task %q requires at least one checkbox step", block.Title))
		}
	}
	return problems
}

func directiveDeltaProblems(changes string) []string {
	if strings.TrimSpace(changes) == "" {
		return nil
	}
	problems := []string{}
	for _, block := range markdownSectionBlocks(changes, 2) {
		if !slices.Contains([]string{"Added", "Modified", "Removed"}, block.Title) {
			problems = append(problems, fmt.Sprintf("invalid delta section %q; want Added, Modified, or Removed", block.Title))
			continue
		}
		if strings.TrimSpace(block.Body) == "" {
			problems = append(problems, fmt.Sprintf("delta section %q requires at least one entry", block.Title))
		}
	}
	return problems
}

func directiveScenarioProblems(criteria string) []string {
	problems := []string{}
	for _, block := range markdownSectionBlocks(criteria, 4) {
		name, ok := strings.CutPrefix(block.Title, "Scenario: ")
		if !ok || strings.TrimSpace(name) == "" {
			problems = append(problems, fmt.Sprintf("invalid scenario heading %q; want \"Scenario: <name>\"", block.Title))
			continue
		}
		if !strings.Contains(block.Body, "**WHEN**") {
			problems = append(problems, fmt.Sprintf("scenario %q requires a **WHEN** bullet", name))
		}
		if !strings.Contains(block.Body, "**THEN**") {
			problems = append(problems, fmt.Sprintf("scenario %q requires a **THEN** bullet", name))
		}
	}
	return problems
}
