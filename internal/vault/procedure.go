package vault

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

const ProcedureType = "Procedure"

// ProcessSections are the canonical executable sections of a Procedure record.
type ProcessSections struct {
	KnowledgeInputs string `json:"knowledge_inputs"`
	Process         string `json:"process"`
	Completion      string `json:"completion"`
}

// ProcessStep is one ordered stage of a multi-step Procedure.
type ProcessStep struct {
	Number   int             `json:"number"`
	Name     string          `json:"name"`
	Sections ProcessSections `json:"sections"`
}

// ProcessSummary is a Procedure descriptor for invocation.
type ProcessSummary struct {
	DocumentRef
	Invocation string   `json:"invocation"`
	Tags       []string `json:"tags"`
}

// ProcessInvocation binds one exact Procedure revision for an agent to execute.
type ProcessInvocation struct {
	Process  ProcessSummary  `json:"process"`
	Sections ProcessSections `json:"sections,omitzero"`
	Steps    []ProcessStep   `json:"steps,omitempty"`
}

type procedureContract struct {
	invocation string
	tags       []string
	sections   ProcessSections
	steps      []ProcessStep
}

type procedureEligibility struct {
	invocation string
	tags       []string
}

// DiscoverProcesses returns valid, model-invocable Procedures matching every
// requested tag.
func DiscoverProcesses(root string, requestedTags []string) (ConceptRecordCatalog, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.validationPages()
	if err != nil {
		return nil, err
	}

	records := make([]map[string]any, 0)
	for _, page := range pages {
		if !isProcedureType(page.document.Type) {
			continue
		}
		eligibility, eligibilityProblems := parseProcedureEligibility(page.fields)
		if len(eligibilityProblems) > 0 || eligibility.invocation != "model" ||
			!containsAll(eligibility.tags, requestedTags) {
			continue
		}
		problems := procedurePageProblems(page)
		_, contractProblems := parseProcedureWithEligibility(page.fields, page.document.Body, eligibility, nil)
		problems = append(problems, contractProblems...)
		if len(problems) > 0 {
			return nil, invalidProcedure(page.document.URI, problems)
		}

		records = append(records, page.authoredRecord())
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i]["uri"].(string) < records[j]["uri"].(string)
	})
	return ConceptRecordCatalog{"procedures": records}, nil
}

func containsAll(values, required []string) bool {
	for _, value := range required {
		if !slices.Contains(values, strings.TrimSpace(value)) {
			return false
		}
	}
	return true
}

// InvokeProcess loads one exact Procedure as an execution contract. It is read-only.
func InvokeProcess(root, selector string) (ProcessInvocation, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return ProcessInvocation{}, err
	}
	pages, err := vault.validationPages()
	if err != nil {
		return ProcessInvocation{}, err
	}
	page, ok := selectPage(pages, selector)
	if !ok {
		return ProcessInvocation{}, fmt.Errorf("no process found with URI %q", selector)
	}
	if page.parseProblem != nil {
		return ProcessInvocation{}, invalidProcedure(page.document.URI, []string{page.parseProblem.Error()})
	}
	if !isProcedureType(page.document.Type) {
		return ProcessInvocation{}, fmt.Errorf("document %q has non-executable type %q", page.document.URI, page.document.Type)
	}

	problems := procedurePageProblems(page)
	procedure, contractProblems := parseProcedure(page.fields, page.document.Body)
	problems = append(problems, contractProblems...)
	if len(problems) > 0 {
		return ProcessInvocation{}, invalidProcedure(page.document.URI, problems)
	}
	return ProcessInvocation{
		Process: ProcessSummary{
			DocumentRef: page.document.Ref(),
			Invocation:  procedure.invocation,
			Tags:        append([]string(nil), procedure.tags...),
		},
		Sections: procedure.sections,
		Steps:    procedure.steps,
	}, nil
}

func procedurePageProblems(page *effectivePage) []string {
	problems := make([]string, 0, len(page.metadataProblems))
	for _, problem := range page.metadataProblems {
		problems = append(problems, problem.Error())
	}
	return problems
}

func isProcedureType(value string) bool {
	return value == ProcedureType
}

func invalidProcedure(uri string, problems []string) error {
	return fmt.Errorf("process %q is invalid: %s", uri, strings.Join(problems, "; "))
}

func parseProcedure(fields frontmatterFields, body string) (procedureContract, []string) {
	eligibility, eligibilityProblems := parseProcedureEligibility(fields)
	return parseProcedureWithEligibility(fields, body, eligibility, eligibilityProblems)
}

func parseProcedureWithEligibility(fields frontmatterFields, body string, eligibility procedureEligibility, eligibilityProblems []string) (procedureContract, []string) {
	sections, steps, problems := parseProcedureSections(body)
	if description, scalar := frontmatterScalar(fields, "description"); !scalar || strings.TrimSpace(description) == "" {
		problems = append(problems, fmt.Sprintf("process requires non-empty %q frontmatter", "description"))
	}

	problems = append(problems, eligibilityProblems...)

	for _, field := range []string{"effects", "relationships"} {
		if _, exists := fields[field]; exists {
			problems = append(problems, fmt.Sprintf("procedure frontmatter must not contain %q", field))
		}
	}

	return procedureContract{
		invocation: eligibility.invocation,
		tags:       eligibility.tags,
		sections:   sections,
		steps:      steps,
	}, problems
}

func parseProcedureEligibility(fields frontmatterFields) (procedureEligibility, []string) {
	eligibility := procedureEligibility{invocation: "model"}
	problems := []string{}

	tags, valid := frontmatterScalars(fields, "tags")
	if !valid {
		problems = append(problems, fmt.Sprintf("frontmatter %q must be a scalar or sequence of scalars", "tags"))
	} else if len(tags) == 0 {
		problems = append(problems, fmt.Sprintf("process requires non-empty %q frontmatter", "tags"))
	}
	eligibility.tags = tags

	if value, scalar := frontmatterScalar(fields, "invocation"); scalar {
		if value = strings.TrimSpace(value); value != "" {
			eligibility.invocation = value
		}
	} else if _, exists := fields["invocation"]; exists {
		problems = append(problems, fmt.Sprintf("frontmatter %q must be a scalar", "invocation"))
	}
	if !validProcedureInvocation(eligibility.invocation) {
		problems = append(problems, fmt.Sprintf("frontmatter %q must be %q or %q", "invocation", "model", "explicit"))
	}
	return eligibility, problems
}

func parseProcedureSections(body string) (ProcessSections, []ProcessStep, []string) {
	blocks := markdownSectionBlocks(body, 2)
	multiStep := false
	for _, block := range blocks {
		if _, _, ok := parseProcedureStepHeading(block.Title); ok {
			multiStep = true
			break
		}
	}
	if !multiStep {
		sections, missing, duplicates := parseRequiredSections(body, 2)
		return sections, nil, procedureSectionProblems("", missing, duplicates)
	}

	problems := []string{}
	steps := make([]ProcessStep, 0, len(blocks))
	names := make(map[string]struct{}, len(blocks))
	for index, block := range blocks {
		number, name, ok := parseProcedureStepHeading(block.Title)
		if !ok {
			problems = append(problems, fmt.Sprintf("invalid multi-step heading %q; want %q", block.Title, "STEP <number> - <name>"))
			continue
		}
		expected := index + 1
		if number != expected {
			problems = append(problems, fmt.Sprintf("%s is out of order; expected STEP %d", block.Title, expected))
		}
		if _, exists := names[name]; exists {
			problems = append(problems, fmt.Sprintf("duplicate step name %q", name))
		}
		names[name] = struct{}{}
		sections, missing, duplicates := parseRequiredSections(block.Body, 3)
		problems = append(problems, procedureSectionProblems(block.Title, missing, duplicates)...)
		steps = append(steps, ProcessStep{Number: number, Name: name, Sections: sections})
	}
	if len(steps) < 2 {
		problems = append(problems, "multi-step procedure requires at least two steps")
	}
	return ProcessSections{}, steps, problems
}

func parseProcedureStepHeading(title string) (int, string, bool) {
	rest, ok := strings.CutPrefix(title, "STEP ")
	if !ok {
		return 0, "", false
	}
	numberText, name, ok := strings.Cut(rest, " - ")
	number, err := strconv.Atoi(numberText)
	name = strings.TrimSpace(name)
	if !ok || err != nil || number < 1 || strconv.Itoa(number) != numberText || name == "" {
		return 0, "", false
	}
	return number, name, true
}

func parseRequiredSections(body string, level int) (ProcessSections, []string, []string) {
	sections, duplicates := markdownSectionsAtLevel(body, level)
	required := []string{"Inputs", "Process", "Completion"}
	missing := []string{}
	for _, name := range required {
		if strings.TrimSpace(sections[name]) == "" {
			missing = append(missing, name)
		}
	}
	return ProcessSections{
		KnowledgeInputs: sections["Inputs"],
		Process:         sections["Process"],
		Completion:      sections["Completion"],
	}, missing, duplicates
}

func procedureSectionProblems(context string, missing, duplicates []string) []string {
	prefix := ""
	if context != "" {
		prefix = context + " "
	}
	problems := make([]string, 0, len(missing)+len(duplicates))
	for _, section := range missing {
		problems = append(problems, fmt.Sprintf("%smissing required section %q", prefix, section))
	}
	for _, section := range duplicates {
		problems = append(problems, fmt.Sprintf("%sduplicate process section %q", prefix, section))
	}
	return problems
}

func markdownSectionsAtLevel(markdown string, level int) (map[string]string, []string) {
	sections := make(map[string]string)
	duplicates := []string{}
	for _, block := range markdownSectionBlocks(markdown, level) {
		if _, exists := sections[block.Title]; exists {
			duplicates = append(duplicates, block.Title)
			continue
		}
		sections[block.Title] = block.Body
	}
	sort.Strings(duplicates)
	return sections, duplicates
}

type markdownSection struct {
	Title string
	Body  string
}

func markdownSectionBlocks(markdown string, level int) []markdownSection {
	prefix := strings.Repeat("#", level) + " "
	blocks := []markdownSection{}
	current := ""
	var content []string
	inFence := false
	fence := ""
	flush := func() {
		if current != "" {
			blocks = append(blocks, markdownSection{Title: current, Body: strings.TrimSpace(strings.Join(content, "\n"))})
		}
	}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fence = marker
			} else if marker == fence {
				inFence = false
				fence = ""
			}
		}
		if !inFence && strings.HasPrefix(line, prefix) {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			content = nil
			continue
		}
		if current != "" {
			content = append(content, line)
		}
	}
	flush()
	return blocks
}

func validProcedureInvocation(value string) bool {
	return value == "model" || value == "explicit"
}
