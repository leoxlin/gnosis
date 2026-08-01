package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecordVersionsRepeatsTombstonesAndPermissions(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	input := Input{
		Vault: "test", Repository: "owner/repo", Kind: "issue", SourceID: "1",
		UpdatedAt: "2026-07-29T12:00:00Z", ObservedAt: "2026-07-29T12:01:00Z",
		Visibility: "private", RawPayload: json.RawMessage(`{"id":1}`),
	}
	first, err := store.Record(input)
	if err != nil || first.Status != StatusCreated {
		t.Fatalf("first = %+v, %v", first, err)
	}
	repeat, err := store.Record(input)
	if err != nil || repeat.Status != StatusUnchanged || repeat.Path != first.Path {
		t.Fatalf("repeat = %+v, %v", repeat, err)
	}
	input.UpdatedAt = "2026-07-29T12:02:00Z"
	input.RawPayload = json.RawMessage(`{"id":1,"title":"changed"}`)
	version, err := store.Record(input)
	if err != nil || version.Status != StatusCreated || version.Path == first.Path {
		t.Fatalf("version = %+v, %v", version, err)
	}
	tombstone, err := store.Tombstone(Input{
		Vault: input.Vault, Repository: input.Repository, Kind: input.Kind, SourceID: input.SourceID,
		UpdatedAt: "2026-07-29T12:03:00Z", ObservedAt: "2026-07-29T12:03:00Z", Visibility: "private",
	})
	if err != nil || !tombstone.Record.Tombstone {
		t.Fatalf("tombstone = %+v, %v", tombstone, err)
	}
	for _, path := range []string{root, first.Path} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		want := os.FileMode(0o700)
		if path == first.Path {
			want = 0o600
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}
}

func TestRecordRejectsInvalidAndConflictingExistingRecords(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	input := Input{
		Vault: "test", Repository: "owner/repo", Kind: "issue", SourceID: "1",
		UpdatedAt: "2026-07-29T12:00:00Z", ObservedAt: "2026-07-29T12:01:00Z",
		Visibility: "private", RawPayload: json.RawMessage(`{"id":1}`),
	}
	result, err := store.Record(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.Path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Record(input); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("conflict error = %v", err)
	}
	input.RawPayload = json.RawMessage(`{`)
	if _, err := store.Record(input); err == nil {
		t.Fatal("invalid payload succeeded")
	}
}

func TestCursorIsVersionedAndAtomicallyReplaceable(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	empty, err := store.LoadCursor("owner/repo")
	if err != nil || empty.Version != 1 || empty.Value != "" {
		t.Fatalf("empty cursor = %+v, %v", empty, err)
	}
	written, err := store.CommitCursor(empty, "next")
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadCursor("owner/repo")
	if err != nil || loaded != written {
		t.Fatalf("loaded = %+v, want %+v, err %v", loaded, written, err)
	}
	info, err := os.Stat(filepath.Join(store.dir, "cursors", digestBytes([]byte("owner/repo"))+".json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cursor mode = %v, %v", info.Mode().Perm(), err)
	}
}
