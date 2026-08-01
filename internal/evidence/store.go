// Package evidence stores bounded immutable source records outside the curated vault.
package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gnosis/internal/s3store"
)

const (
	MaxPayloadBytes = 2 << 20
	StatusCreated   = "created"
	StatusUnchanged = "unchanged"
)

// Input is one observed upstream object version.
type Input struct {
	Vault      string          `json:"vault"`
	Repository string          `json:"repository"`
	Kind       string          `json:"kind"`
	SourceID   string          `json:"source_id"`
	UpdatedAt  string          `json:"updated_at"`
	ObservedAt string          `json:"observed_at"`
	Visibility string          `json:"visibility"`
	URL        string          `json:"url,omitempty"`
	Tombstone  bool            `json:"tombstone,omitempty"`
	DeliveryID string          `json:"delivery_id,omitempty"`
	RawPayload json.RawMessage `json:"raw_payload,omitempty"`
}

// Record is the durable form of Input.
type Record struct {
	Input
	Digest string `json:"digest"`
}

// Result reports whether a record was created or already existed.
type Result struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	Record Record `json:"record"`
}

// Cursor is the only mutable evidence state.
type Cursor struct {
	Version        int    `json:"version"`
	Repository     string `json:"repository"`
	Value          string `json:"value"`
	UpdatedAt      string `json:"updated_at"`
	storageVersion string
}

// Backend is the durable contract shared by filesystem and S3 evidence stores.
type Backend interface {
	Record(Input) (Result, error)
	Tombstone(Input) (Result, error)
	ClaimDelivery(string, string) (string, error)
	CheckDelivery(string, string) (string, error)
	LoadCursor(string) (Cursor, error)
	CommitCursor(Cursor, string) (Cursor, error)
	Latest(string, string) (map[string]Record, error)
}

// Store owns one absolute evidence directory.
type Store struct {
	dir     string
	objects objectStore
	now     func() time.Time
}

func New(dir string) (*Store, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" || !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("evidence directory must be an absolute path")
	}
	dir = filepath.Clean(dir)
	return &Store{dir: dir, objects: fileObjects{dir: dir}, now: time.Now}, nil
}

func (s *Store) Record(input Input) (Result, error) {
	record, data, key, err := prepareRecord(input)
	if err != nil {
		return Result{}, err
	}
	created, _, err := s.objects.Create(context.Background(), key, data)
	if err != nil {
		if s3store.IsConflict(err) {
			existing, _, readErr := s.objects.Read(context.Background(), key)
			var stored Record
			if readErr == nil && json.Unmarshal(existing, &stored) == nil && sameVersion(stored, record) {
				return Result{Status: StatusUnchanged, Path: s.path(key), Record: stored}, nil
			}
			return Result{}, fmt.Errorf("evidence record %s conflicts with existing content", s.path(key))
		}
		return Result{}, err
	}
	status := StatusUnchanged
	if created {
		status = StatusCreated
	}
	return Result{Status: status, Path: s.path(key), Record: record}, nil
}

func prepareRecord(input Input) (Record, []byte, string, error) {
	normalize(&input)
	if err := validate(input); err != nil {
		return Record{}, nil, "", err
	}
	contentDigest := digest(input.RawPayload)
	if input.Tombstone {
		contentDigest = digest([]byte("tombstone:" + input.UpdatedAt))
	}
	record := Record{Input: input, Digest: contentDigest}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return Record{}, nil, "", err
	}
	data = append(data, '\n')
	key := strings.Join([]string{
		input.Vault, input.Repository, input.Kind, input.SourceID, input.UpdatedAt, contentDigest,
	}, "\x00")
	return record, data, "records/" + digestBytes([]byte(key)) + ".json", nil
}

func sameVersion(left, right Record) bool {
	left.ObservedAt, right.ObservedAt = "", ""
	left.DeliveryID, right.DeliveryID = "", ""
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftJSON) == string(rightJSON)
}

func (s *Store) Tombstone(input Input) (Result, error) {
	input.Tombstone = true
	input.RawPayload = nil
	return s.Record(input)
}

// ClaimDelivery de-duplicates one upstream delivery before normalization.
func (s *Store) ClaimDelivery(deliveryID, payloadDigest string) (string, error) {
	status, err := s.CheckDelivery(deliveryID, payloadDigest)
	if err != nil || status != "" {
		return status, err
	}
	created, _, err := s.objects.Create(context.Background(), deliveryKey(deliveryID), []byte(payloadDigest+"\n"))
	if s3store.IsConflict(err) {
		return s.CheckDelivery(deliveryID, payloadDigest)
	}
	if err != nil {
		return "", err
	}
	if !created {
		return StatusUnchanged, nil
	}
	return StatusCreated, nil
}

// CheckDelivery reports an identical prior delivery without creating state.
func (s *Store) CheckDelivery(deliveryID, payloadDigest string) (string, error) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" || len(deliveryID) > 512 {
		return "", fmt.Errorf("evidence delivery ID must be between 1 and 512 bytes")
	}
	if payloadDigest == "" {
		return "", fmt.Errorf("evidence delivery digest must not be empty")
	}
	existing, _, err := s.objects.Read(context.Background(), deliveryKey(deliveryID))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(existing)) != payloadDigest {
		return "", fmt.Errorf("evidence delivery %q conflicts with existing payload", deliveryID)
	}
	return StatusUnchanged, nil
}

func (s *Store) LoadCursor(repository string) (Cursor, error) {
	data, version, err := s.objects.Read(context.Background(), cursorKey(repository))
	if errors.Is(err, fs.ErrNotExist) {
		return Cursor{Version: 1, Repository: repository}, nil
	}
	if err != nil {
		return Cursor{}, err
	}
	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("parse evidence cursor: %w", err)
	}
	if cursor.Version != 1 || cursor.Repository != repository {
		return Cursor{}, fmt.Errorf("unsupported or mismatched evidence cursor")
	}
	cursor.storageVersion = version
	return cursor, nil
}

func (s *Store) CommitCursor(previous Cursor, value string) (Cursor, error) {
	repository := previous.Repository
	if repository == "" {
		return Cursor{}, fmt.Errorf("evidence cursor repository must not be empty")
	}
	cursor := Cursor{
		Version: 1, Repository: repository, Value: value,
		UpdatedAt: s.now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.MarshalIndent(cursor, "", "  ")
	if err != nil {
		return Cursor{}, err
	}
	data = append(data, '\n')
	key := cursorKey(repository)
	var version string
	if previous.storageVersion == "" {
		_, version, err = s.objects.Create(context.Background(), key, data)
	} else {
		version, err = s.objects.Replace(context.Background(), key, data, previous.storageVersion)
	}
	if err != nil {
		if s3store.IsConflict(err) {
			return Cursor{}, fmt.Errorf("evidence cursor changed concurrently")
		}
		return Cursor{}, err
	}
	cursor.storageVersion = version
	return cursor, nil
}

// Latest returns the newest non-tombstone record for every source identity.
func (s *Store) Latest(vault, repository string) (map[string]Record, error) {
	objects, err := s.objects.List(context.Background(), "records/", s3store.MaxListObjects)
	if err != nil {
		return nil, err
	}
	latest := map[string]Record{}
	for _, object := range objects {
		data, _, err := s.objects.Read(context.Background(), object.Key)
		if err != nil {
			return nil, err
		}
		var record Record
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("parse evidence record %s: %w", s.path(object.Key), err)
		}
		if record.Vault != vault || record.Repository != repository {
			continue
		}
		key := record.Kind + "\x00" + record.SourceID
		if current, ok := latest[key]; !ok || current.ObservedAt < record.ObservedAt {
			latest[key] = record
		}
	}
	return latest, nil
}

func (s *Store) path(key string) string {
	if s.dir != "" {
		return filepath.Join(s.dir, filepath.FromSlash(key))
	}
	return s.objects.Location() + "/" + key
}

func cursorKey(repository string) string {
	return "cursors/" + digestBytes([]byte(repository)) + ".json"
}
func deliveryKey(deliveryID string) string {
	return "deliveries/" + digestBytes([]byte(strings.TrimSpace(deliveryID))) + ".txt"
}

type fileObjects struct{ dir string }

func (s fileObjects) Location() string { return s.dir }

func (s fileObjects) Read(_ context.Context, key string) ([]byte, string, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, filepath.FromSlash(key)))
	if err != nil {
		return nil, "", err
	}
	return data, digestBytes(data), nil
}

func (s fileObjects) List(_ context.Context, prefix string, limit int) ([]s3store.Object, error) {
	paths, err := filepath.Glob(filepath.Join(s.dir, filepath.FromSlash(prefix), "*"))
	if err != nil {
		return nil, err
	}
	objects := make([]s3store.Object, 0, min(len(paths), limit))
	for _, path := range paths[:min(len(paths), limit)] {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		key, err := filepath.Rel(s.dir, path)
		if err != nil {
			return nil, err
		}
		objects = append(objects, s3store.Object{Key: filepath.ToSlash(key), Size: info.Size()})
	}
	return objects, nil
}

func (s fileObjects) Create(_ context.Context, key string, data []byte) (bool, string, error) {
	path := filepath.Join(s.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, "", err
	}
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return false, "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if os.IsExist(err) {
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return false, "", readErr
		}
		if bytes.Equal(existing, data) {
			return false, digestBytes(existing), nil
		}
		return false, "", s3store.Conflict{Key: key}
	}
	if err != nil {
		return false, "", err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return false, "", err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return false, "", err
	}
	if err := file.Close(); err != nil {
		return false, "", err
	}
	return true, digestBytes(data), nil
}

func (s fileObjects) Replace(_ context.Context, key string, data []byte, version string) (string, error) {
	path := filepath.Join(s.dir, filepath.FromSlash(key))
	lock, err := os.OpenFile(path+".lock", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", s3store.Conflict{Key: key}
	}
	lock.Close()
	defer os.Remove(path + ".lock")
	existing, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if digestBytes(existing) != version {
		return "", s3store.Conflict{Key: key}
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".cursor-*")
	if err != nil {
		return "", err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return "", err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(name, path); err != nil {
		return "", err
	}
	return digestBytes(data), nil
}

func normalize(input *Input) {
	input.Vault = strings.TrimSpace(input.Vault)
	input.Repository = strings.ToLower(strings.TrimSpace(input.Repository))
	input.Kind = strings.TrimSpace(input.Kind)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.UpdatedAt = strings.TrimSpace(input.UpdatedAt)
	input.ObservedAt = strings.TrimSpace(input.ObservedAt)
	input.Visibility = strings.TrimSpace(input.Visibility)
	input.URL = strings.TrimSpace(input.URL)
	input.DeliveryID = strings.TrimSpace(input.DeliveryID)
}

func validate(input Input) error {
	for name, value := range map[string]string{
		"vault": input.Vault, "repository": input.Repository, "kind": input.Kind,
		"source_id": input.SourceID, "updated_at": input.UpdatedAt,
		"observed_at": input.ObservedAt, "visibility": input.Visibility,
	} {
		if value == "" {
			return fmt.Errorf("evidence %s must not be empty", name)
		}
		if len(value) > 512 {
			return fmt.Errorf("evidence %s exceeds 512 bytes", name)
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, input.UpdatedAt); err != nil {
		return fmt.Errorf("evidence updated_at: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, input.ObservedAt); err != nil {
		return fmt.Errorf("evidence observed_at: %w", err)
	}
	if input.Tombstone {
		if len(input.RawPayload) != 0 {
			return fmt.Errorf("evidence tombstone must not contain a payload")
		}
		return nil
	}
	if len(input.RawPayload) == 0 || len(input.RawPayload) > MaxPayloadBytes || !json.Valid(input.RawPayload) {
		return fmt.Errorf("evidence payload must be valid JSON no larger than %d bytes", MaxPayloadBytes)
	}
	return nil
}

func digest(data []byte) string {
	return "sha256:" + digestBytes(data)
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
