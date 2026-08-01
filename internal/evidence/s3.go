package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"gnosis/internal/s3store"
)

type s3Objects interface {
	Read(context.Context, string) ([]byte, string, error)
	List(context.Context, string, int) ([]s3store.Object, error)
	Create(context.Context, string, []byte) (bool, string, error)
	Replace(context.Context, string, []byte, string) (string, error)
	Location() string
}

type S3Store struct {
	objects s3Objects
	now     func() time.Time
}

func NewS3(ctx context.Context, config s3store.Config) (*S3Store, error) {
	objects, err := s3store.New(ctx, config)
	if err != nil {
		return nil, err
	}
	return newS3(objects), nil
}

func newS3(objects s3Objects) *S3Store {
	return &S3Store{objects: objects, now: time.Now}
}

func (s *S3Store) Record(input Input) (Result, error) {
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

func (s *S3Store) Tombstone(input Input) (Result, error) {
	input.Tombstone = true
	input.RawPayload = nil
	return s.Record(input)
}

func (s *S3Store) CheckDelivery(deliveryID, payloadDigest string) (string, error) {
	deliveryID = strings.TrimSpace(deliveryID)
	if deliveryID == "" || len(deliveryID) > 512 {
		return "", fmt.Errorf("evidence delivery ID must be between 1 and 512 bytes")
	}
	if payloadDigest == "" {
		return "", fmt.Errorf("evidence delivery digest must not be empty")
	}
	data, _, err := s.objects.Read(context.Background(), deliveryKey(deliveryID))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(data)) != payloadDigest {
		return "", fmt.Errorf("evidence delivery %q conflicts with existing payload", deliveryID)
	}
	return StatusUnchanged, nil
}

func (s *S3Store) ClaimDelivery(deliveryID, payloadDigest string) (string, error) {
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

func (s *S3Store) LoadCursor(repository string) (Cursor, error) {
	data, etag, err := s.objects.Read(context.Background(), cursorKey(repository))
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
	cursor.storageVersion = etag
	return cursor, nil
}

func (s *S3Store) CommitCursor(previous Cursor, value string) (Cursor, error) {
	if previous.Repository == "" {
		return Cursor{}, fmt.Errorf("evidence cursor repository must not be empty")
	}
	cursor := Cursor{Version: 1, Repository: previous.Repository, Value: value, UpdatedAt: s.now().UTC().Format(time.RFC3339Nano)}
	data, err := json.MarshalIndent(cursor, "", "  ")
	if err != nil {
		return Cursor{}, err
	}
	data = append(data, '\n')
	key := cursorKey(previous.Repository)
	var etag string
	if previous.storageVersion == "" {
		_, etag, err = s.objects.Create(context.Background(), key, data)
	} else {
		etag, err = s.objects.Replace(context.Background(), key, data, previous.storageVersion)
	}
	if err != nil {
		if s3store.IsConflict(err) {
			return Cursor{}, fmt.Errorf("evidence cursor changed concurrently")
		}
		return Cursor{}, err
	}
	cursor.storageVersion = etag
	return cursor, nil
}

func (s *S3Store) Latest(vault, repository string) (map[string]Record, error) {
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

func (s *S3Store) path(key string) string { return s.objects.Location() + "/" + key }
func cursorKey(repository string) string {
	return "cursors/" + digestBytes([]byte(repository)) + ".json"
}
func deliveryKey(deliveryID string) string {
	return "deliveries/" + digestBytes([]byte(strings.TrimSpace(deliveryID))) + ".txt"
}
