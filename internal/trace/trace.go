// Package trace stores explicit, bounded agent-run evidence outside the gnosis vault.
package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDir     = "GNOSIS_TRACE_DIR"
	EnvAgentID = "GNOSIS_TRACE_AGENT_ID"

	MaxIDBytes       = 256
	MaxContentBytes  = 64 * 1024
	MaxMetadataBytes = 64 * 1024
	MaxRecordBytes   = 132 * 1024

	StatusCreated = "CREATED"
	StatusNoop    = "NOOP"
)

var supportedKinds = map[string]bool{
	"run": true, "plan": true, "tool": true, "patch": true,
	"test": true, "failure": true, "outcome": true,
}

// Config fixes the server-owned trace identity and storage directory.
type Config struct {
	Dir     string
	AgentID string
}

// Input is the caller-authored portion of one trace entry.
type Input struct {
	RunID      string         `json:"run_id"`
	Sequence   int64          `json:"sequence"`
	Kind       string         `json:"kind"`
	OccurredAt string         `json:"occurred_at"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// Record is one immutable persisted trace entry.
type Record struct {
	AgentID     string         `json:"agent_id"`
	RunID       string         `json:"run_id"`
	Sequence    int64          `json:"sequence"`
	Kind        string         `json:"kind"`
	OccurredAt  string         `json:"occurred_at"`
	Content     string         `json:"content"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	ContentHash string         `json:"content_hash"`
	RecordedAt  string         `json:"recorded_at"`
}

// Result reports whether a record was created or already existed identically.
type Result struct {
	Status string `json:"status"`
	Record Record `json:"record"`
}

// Store persists entries for one configured agent identity.
type Store struct {
	config Config
	now    func() time.Time
}

// NewFromEnv validates GNOSIS_TRACE_* and constructs a store.
func NewFromEnv() (*Store, error) {
	return New(Config{
		Dir:     os.Getenv(EnvDir),
		AgentID: os.Getenv(EnvAgentID),
	})
}

// New validates configuration without creating filesystem state.
func New(config Config) (*Store, error) {
	config.Dir = strings.TrimSpace(config.Dir)
	config.AgentID = strings.TrimSpace(config.AgentID)
	if config.AgentID == "" {
		return nil, fmt.Errorf("trace config: %s must not be empty", EnvAgentID)
	}
	if len(config.AgentID) > MaxIDBytes {
		return nil, fmt.Errorf("trace config: %s exceeds %d bytes", EnvAgentID, MaxIDBytes)
	}
	if config.Dir == "" {
		return nil, fmt.Errorf("trace config: %s must not be empty", EnvDir)
	}
	if !filepath.IsAbs(config.Dir) {
		return nil, fmt.Errorf("trace config: %s must be an absolute path", EnvDir)
	}
	config.Dir = filepath.Clean(config.Dir)
	return &Store{config: config, now: time.Now}, nil
}

// Record creates one exclusive JSON file or returns the identical existing record.
func (s *Store) Record(input Input) (Result, error) {
	input.RunID = strings.TrimSpace(input.RunID)
	input.Kind = strings.TrimSpace(input.Kind)
	input.OccurredAt = strings.TrimSpace(input.OccurredAt)
	if err := validateInput(input); err != nil {
		return Result{}, err
	}

	authored, err := json.Marshal(input)
	if err != nil {
		return Result{}, fmt.Errorf("record trace: metadata must be valid JSON: %w", err)
	}
	if len(authored) > MaxRecordBytes {
		return Result{}, fmt.Errorf("record trace: authored record exceeds %d bytes", MaxRecordBytes)
	}
	contentHash := hash(authored)
	record := Record{
		AgentID: s.config.AgentID, RunID: input.RunID, Sequence: input.Sequence,
		Kind: input.Kind, OccurredAt: input.OccurredAt, Content: input.Content,
		Metadata: input.Metadata, ContentHash: contentHash,
		RecordedAt: s.now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return Result{}, fmt.Errorf("record trace: encode record: %w", err)
	}
	data = append(data, '\n')
	if len(data) > MaxRecordBytes {
		return Result{}, fmt.Errorf("record trace: persisted record exceeds %d bytes", MaxRecordBytes)
	}

	runDir := filepath.Join(s.config.Dir, hash([]byte(s.config.AgentID)), hash([]byte(input.RunID)))
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("record trace: create run directory: %w", err)
	}
	if err := os.Chmod(runDir, 0o700); err != nil {
		return Result{}, fmt.Errorf("record trace: protect run directory: %w", err)
	}
	path := filepath.Join(runDir, strconv.FormatInt(input.Sequence, 10)+".json")
	file, err := os.CreateTemp(runDir, ".trace-*")
	if err != nil {
		return Result{}, fmt.Errorf("record trace: create temporary record: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return Result{}, fmt.Errorf("record trace: write record: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return Result{}, fmt.Errorf("record trace: sync record: %w", err)
	}
	if err := file.Close(); err != nil {
		return Result{}, fmt.Errorf("record trace: close record: %w", err)
	}
	if err := os.Link(file.Name(), path); errors.Is(err, os.ErrExist) {
		return existing(path, contentHash)
	} else if err != nil {
		return Result{}, fmt.Errorf("record trace: publish record: %w", err)
	}
	directory, err := os.Open(runDir)
	if err != nil {
		return Result{}, fmt.Errorf("record trace: open run directory: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return Result{}, fmt.Errorf("record trace: sync run directory: %w", err)
	}
	if err := directory.Close(); err != nil {
		return Result{}, fmt.Errorf("record trace: close run directory: %w", err)
	}
	return Result{Status: StatusCreated, Record: record}, nil
}

func validateInput(input Input) error {
	if input.RunID == "" {
		return errors.New("record trace: run_id must not be empty")
	}
	if len(input.RunID) > MaxIDBytes {
		return fmt.Errorf("record trace: run_id exceeds %d bytes", MaxIDBytes)
	}
	if input.Sequence < 0 {
		return errors.New("record trace: sequence must not be negative")
	}
	if !supportedKinds[input.Kind] {
		return errors.New("record trace: kind must be run, plan, tool, patch, test, failure, or outcome")
	}
	if _, err := time.Parse(time.RFC3339, input.OccurredAt); err != nil {
		return errors.New("record trace: occurred_at must be RFC 3339")
	}
	if strings.TrimSpace(input.Content) == "" {
		return errors.New("record trace: content must not be empty")
	}
	if len(input.Content) > MaxContentBytes {
		return fmt.Errorf("record trace: content exceeds %d bytes", MaxContentBytes)
	}
	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return fmt.Errorf("record trace: metadata must be valid JSON: %w", err)
	}
	if len(metadata) > MaxMetadataBytes {
		return fmt.Errorf("record trace: metadata exceeds %d bytes", MaxMetadataBytes)
	}
	return nil
}

func existing(path, contentHash string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("record trace: read existing record: %w", err)
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil {
		return Result{}, fmt.Errorf("record trace: existing record is invalid: %w", err)
	}
	if record.ContentHash != contentHash {
		return Result{}, errors.New("record trace: sequence conflicts with an existing entry")
	}
	return Result{Status: StatusNoop, Record: record}, nil
}

func hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
