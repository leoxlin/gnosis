package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"
	"time"

	"gnosis/internal/s3store"
)

func TestEvidenceStoreContractFilesystemAndS3(t *testing.T) {
	fake := newEvidenceObjects()
	for _, test := range []struct {
		name string
		new  func() Backend
	}{
		{"filesystem", func() Backend { store, _ := New(t.TempDir()); return store }},
		{"s3", func() Backend { return newS3(fake) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := test.new()
			input := contractInput()
			first, err := store.Record(input)
			if err != nil || first.Status != StatusCreated {
				t.Fatalf("create = %+v, %v", first, err)
			}
			input.ObservedAt = "2026-07-29T12:02:00Z"
			repeat, err := store.Record(input)
			if err != nil || repeat.Status != StatusUnchanged {
				t.Fatalf("repeat = %+v, %v", repeat, err)
			}
			tombstone, err := store.Tombstone(Input{
				Vault: "test", Repository: "owner/repo", Kind: "issue", SourceID: "1",
				UpdatedAt: "2026-07-29T12:03:00Z", ObservedAt: "2026-07-29T12:03:00Z", Visibility: "private",
			})
			if err != nil || !tombstone.Record.Tombstone {
				t.Fatalf("tombstone = %+v, %v", tombstone, err)
			}
			cursor, err := store.LoadCursor("owner/repo")
			if err != nil {
				t.Fatal(err)
			}
			cursor, err = store.CommitCursor(cursor, "issue:2")
			if err != nil || cursor.Value != "issue:2" {
				t.Fatalf("cursor = %+v, %v", cursor, err)
			}
			latest, err := store.Latest("test", "owner/repo")
			if err != nil || len(latest) != 1 || !latest["issue\x001"].Tombstone {
				t.Fatalf("latest = %+v, %v", latest, err)
			}
		})
	}
}

func TestS3EvidenceCursorOrderingResumeAndConcurrency(t *testing.T) {
	objects := newEvidenceObjects()
	first, second := newS3(objects), newS3(objects)
	first.now = func() time.Time { return time.Unix(1, 0) }
	cursor1, _ := first.LoadCursor("owner/repo")
	cursor2, _ := second.LoadCursor("owner/repo")
	objects.failCreatePrefix = "records/"
	if _, err := first.Record(contractInput()); err == nil {
		t.Fatal("interrupted evidence write succeeded")
	}
	loaded, _ := first.LoadCursor("owner/repo")
	if loaded.Value != "" {
		t.Fatalf("failed record advanced cursor: %+v", loaded)
	}
	objects.failCreatePrefix = ""
	if _, err := first.Record(contractInput()); err != nil {
		t.Fatal(err)
	}
	advanced, err := first.CommitCursor(cursor1, "issue:2")
	if err != nil || advanced.Value != "issue:2" {
		t.Fatalf("advance = %+v, %v", advanced, err)
	}
	if _, err := second.CommitCursor(cursor2, "issue:3"); err == nil || !strings.Contains(err.Error(), "concurrently") {
		t.Fatalf("concurrent advance = %v", err)
	}
	resumed, err := second.LoadCursor("owner/repo")
	if err != nil || resumed.Value != "issue:2" {
		t.Fatalf("resume = %+v, %v", resumed, err)
	}
}

func contractInput() Input {
	return Input{
		Vault: "test", Repository: "owner/repo", Kind: "issue", SourceID: "1",
		UpdatedAt: "2026-07-29T12:00:00Z", ObservedAt: "2026-07-29T12:01:00Z",
		Visibility: "private", RawPayload: json.RawMessage(`{"id":1}`),
	}
}

type evidenceObjects struct {
	data             map[string][]byte
	etags            map[string]string
	next             int
	failCreatePrefix string
}

func newEvidenceObjects() *evidenceObjects {
	return &evidenceObjects{data: map[string][]byte{}, etags: map[string]string{}, next: 1}
}

func (s *evidenceObjects) Location() string { return "s3://bucket/evidence" }

func (s *evidenceObjects) Read(_ context.Context, key string) ([]byte, string, error) {
	data, ok := s.data[key]
	if !ok {
		return nil, "", fs.ErrNotExist
	}
	return append([]byte(nil), data...), s.etags[key], nil
}

func (s *evidenceObjects) List(_ context.Context, prefix string, limit int) ([]s3store.Object, error) {
	keys := []string{}
	for key := range s.data {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}
	result := make([]s3store.Object, 0, len(keys))
	for _, key := range keys {
		result = append(result, s3store.Object{Key: key, ETag: s.etags[key], Size: int64(len(s.data[key]))})
	}
	return result, nil
}

func (s *evidenceObjects) Create(_ context.Context, key string, data []byte) (bool, string, error) {
	if strings.HasPrefix(key, s.failCreatePrefix) && s.failCreatePrefix != "" {
		return false, "", errors.New("interrupted")
	}
	if existing, ok := s.data[key]; ok {
		if bytes.Equal(existing, data) {
			return false, s.etags[key], nil
		}
		return false, "", s3store.Conflict{Key: key}
	}
	return true, s.put(key, data), nil
}

func (s *evidenceObjects) Replace(_ context.Context, key string, data []byte, etag string) (string, error) {
	if s.etags[key] != etag {
		return "", s3store.Conflict{Key: key}
	}
	return s.put(key, data), nil
}

func (s *evidenceObjects) put(key string, data []byte) string {
	s.next++
	etag := fmt.Sprintf("etag-%d", s.next)
	s.data[key], s.etags[key] = append([]byte(nil), data...), etag
	return etag
}
