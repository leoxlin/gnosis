package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRecordPersistsIdempotentlyWithRestrictivePermissions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "traces")
	store, err := New(Config{Dir: root, AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC) }
	input := Input{
		RunID: "run-1", Sequence: 7, Kind: "tool",
		OccurredAt: "2026-07-29T11:59:00Z", Content: "called test tool",
		Metadata: map[string]any{"tool": "go test"},
	}

	created, err := store.Record(input)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := store.Record(input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != StatusCreated || repeated.Status != StatusNoop ||
		repeated.Record.ContentHash != created.Record.ContentHash {
		t.Fatalf("created = %+v, repeated = %+v", created, repeated)
	}

	var files []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Mode().Perm() != 0o700 {
			t.Errorf("directory %s mode = %o", path, info.Mode().Perm())
		}
		if !info.IsDir() {
			files = append(files, path)
			if info.Mode().Perm() != 0o600 {
				t.Errorf("file %s mode = %o", path, info.Mode().Perm())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %v", files)
	}
	var stored Record
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.AgentID != "agent" || stored.RunID != input.RunID ||
		stored.Sequence != input.Sequence || stored.Kind != input.Kind ||
		stored.OccurredAt != input.OccurredAt || stored.Content != input.Content ||
		stored.Metadata["tool"] != "go test" || stored.RecordedAt != "2026-07-29T12:00:00Z" {
		t.Fatalf("stored = %+v", stored)
	}
}

func TestRecordRejectsConflictAndInvalidInputs(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	valid := Input{
		RunID: "run", Sequence: 1, Kind: "run",
		OccurredAt: "2026-07-29T12:00:00Z", Content: "started",
	}
	if _, err := store.Record(valid); err != nil {
		t.Fatal(err)
	}
	changed := valid
	changed.Content = "replaced"
	if _, err := store.Record(changed); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("conflict error = %v", err)
	}

	tests := []Input{
		{Sequence: 1, Kind: "run", OccurredAt: valid.OccurredAt, Content: "x"},
		{RunID: "run", Sequence: -1, Kind: "run", OccurredAt: valid.OccurredAt, Content: "x"},
		{RunID: "run", Sequence: 2, Kind: "other", OccurredAt: valid.OccurredAt, Content: "x"},
		{RunID: "run", Sequence: 2, Kind: "run", OccurredAt: "today", Content: "x"},
		{RunID: "run", Sequence: 2, Kind: "run", OccurredAt: valid.OccurredAt},
		{RunID: "run", Sequence: 2, Kind: "run", OccurredAt: valid.OccurredAt, Content: strings.Repeat("x", MaxContentBytes+1)},
		{RunID: "run", Sequence: 2, Kind: "run", OccurredAt: valid.OccurredAt, Content: "x", Metadata: map[string]any{"x": strings.Repeat("x", MaxMetadataBytes)}},
	}
	for _, input := range tests {
		if _, err := store.Record(input); err == nil {
			t.Fatalf("Record(%+v) succeeded", input)
		}
	}
}

func TestRecordConcurrentRepeatsPublishOneCompleteRecord(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{
		RunID: "run", Sequence: 1, Kind: "run",
		OccurredAt: "2026-07-29T12:00:00Z", Content: strings.Repeat("x", MaxContentBytes),
	}
	const calls = 16
	results := make(chan Result, calls)
	failures := make(chan error, calls)
	var group sync.WaitGroup
	for range calls {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := store.Record(input)
			if err != nil {
				failures <- err
				return
			}
			results <- result
		}()
	}
	group.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	created := 0
	for result := range results {
		if result.Status == StatusCreated {
			created++
		} else if result.Status != StatusNoop {
			t.Errorf("status = %q", result.Status)
		}
	}
	if created != 1 {
		t.Fatalf("created = %d, want 1", created)
	}
}

func TestNewFromEnvValidatesConfiguration(t *testing.T) {
	t.Setenv(EnvAgentID, "")
	t.Setenv(EnvDir, "")
	if _, err := NewFromEnv(); err == nil || !strings.Contains(err.Error(), EnvAgentID) {
		t.Fatalf("missing agent error = %v", err)
	}
	t.Setenv(EnvAgentID, "agent")
	t.Setenv(EnvDir, "relative")
	if _, err := NewFromEnv(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative directory error = %v", err)
	}
}

func TestRecordDoesNotReplaceMalformedExistingRecord(t *testing.T) {
	root := t.TempDir()
	store, err := New(Config{Dir: root, AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	input := Input{
		RunID: "run", Sequence: 1, Kind: "run",
		OccurredAt: "2026-07-29T12:00:00Z", Content: "started",
	}
	runDir := filepath.Join(root, hash([]byte("agent")), hash([]byte("run")))
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(runDir, "1.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(input); err == nil || !strings.Contains(err.Error(), "existing record is invalid") {
		t.Fatalf("malformed existing error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{" {
		t.Fatalf("existing record changed: %q", data)
	}
}

func TestRecordValidatesKnowledgeAttributionAndFeedback(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	revision := "sha256:" + strings.Repeat("a", 64)
	valid := Input{
		RunID: "run", Sequence: 1, Kind: "feedback",
		OccurredAt: "2026-07-29T12:00:00Z", Content: "useful",
		KnowledgeURI:      "gnosis://test/procedures/query.md",
		KnowledgeRevision: revision, Feedback: "helpful",
	}
	result, err := store.Record(valid)
	if err != nil {
		t.Fatal(err)
	}
	if result.Record.KnowledgeURI != valid.KnowledgeURI ||
		result.Record.KnowledgeRevision != revision || result.Record.Feedback != "helpful" {
		t.Fatalf("record = %+v", result.Record)
	}

	for _, change := range []func(*Input){
		func(input *Input) { input.KnowledgeURI = "query.md" },
		func(input *Input) { input.KnowledgeRevision = "latest" },
		func(input *Input) { input.Feedback = "liked" },
		func(input *Input) {
			input.Sequence = 2
			input.Kind = "knowledge_use"
			input.Feedback = "helpful"
		},
		func(input *Input) {
			input.Sequence = 2
			input.Kind = "tool"
			input.Feedback = ""
		},
	} {
		input := valid
		input.Sequence++
		change(&input)
		if _, err := store.Record(input); err == nil {
			t.Fatalf("Record(%+v) succeeded", input)
		}
	}
}

func TestReadReturnsCompleteLegacyCompatibleRunInSequenceOrder(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []Input{
		{RunID: "run", Sequence: 2, Kind: "outcome", OccurredAt: "2026-07-29T12:02:00Z", Content: "done", Metadata: map[string]any{"success": true}},
		{RunID: "run", Sequence: 0, Kind: "run", OccurredAt: "2026-07-29T12:00:00Z", Content: "started"},
		{RunID: "run", Sequence: 1, Kind: "tool", OccurredAt: "2026-07-29T12:01:00Z", Content: "tested"},
	} {
		if _, err := store.Record(input); err != nil {
			t.Fatal(err)
		}
	}

	run, err := store.Read("run", ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !run.Complete || run.Truncated || len(run.Diagnostics) != 0 ||
		len(run.Entries) != 3 || run.Entries[0].Sequence != 0 ||
		run.Entries[1].Sequence != 1 || run.Entries[2].Sequence != 2 {
		t.Fatalf("run = %+v", run)
	}
}

func TestReadReportsGapsMalformedRecordsAndTamperedHashes(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []Input{
		{RunID: "run", Sequence: 0, Kind: "run", OccurredAt: "2026-07-29T12:00:00Z", Content: "started"},
		{RunID: "run", Sequence: 2, Kind: "outcome", OccurredAt: "2026-07-29T12:02:00Z", Content: "done"},
	} {
		if _, err := store.Record(input); err != nil {
			t.Fatal(err)
		}
	}
	runDir := filepath.Join(store.config.Dir, hash([]byte("agent")), hash([]byte("run")))
	if err := os.WriteFile(filepath.Join(runDir, "3.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(runDir, "2.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record.Content = "tampered"
	data, err = json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "2.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "unexpected"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	run, err := store.Read("run", ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, diagnostic := range run.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if run.Complete || !codes["content_hash"] || !codes["malformed_record"] ||
		!codes["malformed_file"] || !codes["missing_outcome"] {
		t.Fatalf("run = %+v", run)
	}
}

func TestReadReturnsBoundedContinuationDeterministically(t *testing.T) {
	store, err := New(Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	for sequence := int64(0); sequence < 4; sequence++ {
		kind := "tool"
		if sequence == 3 {
			kind = "outcome"
		}
		if _, err := store.Record(Input{
			RunID: "run", Sequence: sequence, Kind: kind,
			OccurredAt: "2026-07-29T12:00:00Z", Content: "entry",
		}); err != nil {
			t.Fatal(err)
		}
	}

	first, err := store.Read("run", ReadOptions{MaxEntries: 2, MaxCharacters: 100})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Truncated || first.Continuation == nil || *first.Continuation != 2 ||
		len(first.Entries) != 2 {
		t.Fatalf("first = %+v", first)
	}
	second, err := store.Read("run", ReadOptions{
		Cursor: first.Continuation, MaxEntries: 2, MaxCharacters: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Complete || second.Truncated || len(second.Entries) != 2 ||
		second.Entries[0].Sequence != 2 || second.Entries[1].Sequence != 3 {
		t.Fatalf("second = %+v", second)
	}

	charBounded, err := store.Read("run", ReadOptions{MaxEntries: 4, MaxCharacters: 5})
	if err != nil {
		t.Fatal(err)
	}
	if !charBounded.Truncated || charBounded.Continuation == nil ||
		*charBounded.Continuation != 1 || len(charBounded.Entries) != 1 {
		t.Fatalf("character bounded = %+v", charBounded)
	}
}
