package trace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultReadEntries    = 100
	DefaultReadCharacters = 256 * 1024
	MaxReadEntries        = 1000
	MaxReadCharacters     = 1024 * 1024
	MaxReadDiagnostics    = 100
)

type ReadOptions struct {
	Cursor        *int64 `json:"cursor,omitempty"`
	MaxEntries    int    `json:"max_entries,omitempty"`
	MaxCharacters int    `json:"max_characters,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Sequence *int64 `json:"sequence,omitempty"`
}

type Run struct {
	AgentID      string       `json:"agent_id"`
	RunID        string       `json:"run_id"`
	Entries      []Record     `json:"entries"`
	Complete     bool         `json:"complete"`
	Diagnostics  []Diagnostic `json:"diagnostics"`
	Truncated    bool         `json:"truncated"`
	Continuation *int64       `json:"continuation,omitempty"`
}

type sequenceFile struct {
	sequence int64
	path     string
}

func (s *Store) Read(runID string, options ReadOptions) (Run, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return Run{}, errors.New("get run trace: run_id must not be empty")
	}
	if len(runID) > MaxIDBytes {
		return Run{}, fmt.Errorf("get run trace: run_id exceeds %d bytes", MaxIDBytes)
	}
	if options.Cursor != nil && *options.Cursor < 0 {
		return Run{}, errors.New("get run trace: cursor must not be negative")
	}
	if options.MaxEntries == 0 {
		options.MaxEntries = DefaultReadEntries
	}
	if options.MaxCharacters == 0 {
		options.MaxCharacters = DefaultReadCharacters
	}
	if options.MaxEntries < 1 || options.MaxEntries > MaxReadEntries {
		return Run{}, fmt.Errorf("get run trace: max_entries must be between 1 and %d", MaxReadEntries)
	}
	if options.MaxCharacters < 1 || options.MaxCharacters > MaxReadCharacters {
		return Run{}, fmt.Errorf("get run trace: max_characters must be between 1 and %d", MaxReadCharacters)
	}

	run := Run{
		AgentID: s.config.AgentID, RunID: runID,
		Entries: []Record{}, Diagnostics: []Diagnostic{},
	}
	runDir := filepath.Join(s.config.Dir, hash([]byte(s.config.AgentID)), hash([]byte(runID)))
	entries, err := os.ReadDir(runDir)
	if errors.Is(err, os.ErrNotExist) {
		return Run{}, errors.New("get run trace: run not found")
	}
	if err != nil {
		return Run{}, fmt.Errorf("get run trace: read run directory: %w", err)
	}

	files := make([]sequenceFile, 0, len(entries))
	for _, entry := range entries {
		sequence, ok := traceSequence(entry)
		if !ok {
			addDiagnostic(&run, Diagnostic{
				Code: "malformed_file", Message: fmt.Sprintf("unexpected retained file %q", entry.Name()),
			})
			continue
		}
		if options.Cursor != nil && sequence < *options.Cursor {
			continue
		}
		files = append(files, sequenceFile{sequence: sequence, path: filepath.Join(runDir, entry.Name())})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].sequence < files[j].sequence })

	characters := 0
	var previous *int64
	hasOutcome := false
	for _, file := range files {
		if len(run.Entries) == options.MaxEntries {
			run.Truncated = true
			run.Continuation = int64Pointer(file.sequence)
			break
		}
		record, diagnostic := readRecord(file, s.config.AgentID, runID)
		if diagnostic != nil {
			addDiagnostic(&run, *diagnostic)
			continue
		}
		nextCharacters := characters + utf8.RuneCountInString(record.Content)
		if nextCharacters > options.MaxCharacters {
			run.Truncated = true
			run.Continuation = int64Pointer(file.sequence)
			break
		}
		if previous != nil && record.Sequence != *previous+1 {
			addDiagnostic(&run, Diagnostic{
				Code:     "sequence_gap",
				Message:  fmt.Sprintf("sequence gap after %d before %d", *previous, record.Sequence),
				Sequence: int64Pointer(record.Sequence),
			})
		}
		run.Entries = append(run.Entries, record)
		characters = nextCharacters
		previous = int64Pointer(record.Sequence)
		hasOutcome = hasOutcome || record.Kind == "outcome"
	}
	if !run.Truncated && !hasOutcome {
		addDiagnostic(&run, Diagnostic{
			Code: "missing_outcome", Message: "run has no outcome entry",
		})
	}
	run.Complete = !run.Truncated && len(run.Diagnostics) == 0 && hasOutcome
	return run, nil
}

func traceSequence(entry os.DirEntry) (int64, bool) {
	if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
		return 0, false
	}
	value := strings.TrimSuffix(entry.Name(), ".json")
	sequence, err := strconv.ParseInt(value, 10, 64)
	return sequence, err == nil && sequence >= 0 && strconv.FormatInt(sequence, 10) == value
}

func readRecord(file sequenceFile, agentID, runID string) (Record, *Diagnostic) {
	info, err := os.Stat(file.path)
	if err != nil {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", err.Error())
	}
	if info.Size() > MaxRecordBytes {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", "record exceeds retained size bound")
	}
	data, err := os.ReadFile(file.path)
	if err != nil {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", err.Error())
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", err.Error())
	}
	if record.AgentID != agentID || record.RunID != runID || record.Sequence != file.sequence {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", "retained identity does not match the requested run")
	}
	input := Input{
		RunID: record.RunID, Sequence: record.Sequence, Kind: record.Kind,
		OccurredAt: record.OccurredAt, Content: record.Content, Metadata: record.Metadata,
		KnowledgeURI: record.KnowledgeURI, KnowledgeRevision: record.KnowledgeRevision,
		Feedback: record.Feedback,
	}
	if err := validateInput(input); err != nil {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", err.Error())
	}
	authored, err := json.Marshal(input)
	if err != nil || record.ContentHash != hash(authored) {
		return Record{}, recordDiagnostic(file.sequence, "content_hash", "retained content hash does not match authored fields")
	}
	if _, err := time.Parse(time.RFC3339, record.RecordedAt); err != nil {
		return Record{}, recordDiagnostic(file.sequence, "malformed_record", "recorded_at must be RFC 3339")
	}
	return record, nil
}

func recordDiagnostic(sequence int64, code, message string) *Diagnostic {
	return &Diagnostic{Code: code, Message: message, Sequence: int64Pointer(sequence)}
}

func addDiagnostic(run *Run, diagnostic Diagnostic) {
	if len(run.Diagnostics) < MaxReadDiagnostics {
		run.Diagnostics = append(run.Diagnostics, diagnostic)
	} else if run.Diagnostics[MaxReadDiagnostics-1].Code != "diagnostics_truncated" {
		run.Diagnostics[MaxReadDiagnostics-1] = Diagnostic{
			Code: "diagnostics_truncated", Message: "additional retained-record diagnostics omitted",
		}
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
