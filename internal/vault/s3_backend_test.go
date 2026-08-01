package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/s3store"
)

func TestVaultManifestIsDeterministicAndValidated(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "z.md"), "z")
	writeTestFile(t, filepath.Join(root, "nested", "a.md"), "a")
	first, firstData, firstDigest, err := buildManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	_, secondData, secondDigest, err := buildManifest(root)
	if err != nil || string(firstData) != string(secondData) || firstDigest != secondDigest {
		t.Fatalf("manifest is not deterministic: %v", err)
	}
	if len(first.Entries) != 2 || first.Entries[0].Path != "nested/a.md" || first.Entries[1].Path != "z.md" {
		t.Fatalf("entries = %+v", first.Entries)
	}
	invalid := []vaultManifest{
		{Version: 1, Entries: []manifestEntry{{Path: "../bad", Digest: firstDigest}}},
		{Version: 1, Entries: []manifestEntry{{Path: "same", Digest: firstDigest}, {Path: "same", Digest: firstDigest}}},
		{Version: 1, Entries: []manifestEntry{{Path: "bad", Digest: "no", Size: -1}}},
	}
	for _, manifest := range invalid {
		if _, err := encodeManifest(manifest); err == nil {
			t.Fatalf("invalid manifest succeeded: %+v", manifest)
		}
	}
}

func TestS3VaultSynchronizationPublicationAndConflicts(t *testing.T) {
	ctx := context.Background()
	store := newVaultFakeStore()
	first := &s3Backend{store: store, cacheRoot: filepath.Join(t.TempDir(), "first")}
	if err := first.synchronize(ctx); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(first.root, "one.md"), "one")
	writeTestFile(t, filepath.Join(first.root, "removed.md"), "removed")
	if err := first.publish("initial"); err != nil {
		t.Fatal(err)
	}
	initialPointer := string(store.objects["vault/current"])
	if err := first.publish("no-op"); err != nil || string(store.objects["vault/current"]) != initialPointer {
		t.Fatalf("no-op publish = %v", err)
	}

	second := &s3Backend{store: store, cacheRoot: filepath.Join(t.TempDir(), "second")}
	if err := second.synchronize(ctx); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, filepath.Join(second.root, "one.md")); got != "one" {
		t.Fatalf("synchronized = %q", got)
	}
	writeTestFile(t, filepath.Join(first.root, "one.md"), "updated")
	if err := os.Remove(filepath.Join(first.root, "removed.md")); err != nil {
		t.Fatal(err)
	}
	if err := first.publish("update"); err != nil {
		t.Fatal(err)
	}
	third := &s3Backend{store: store, cacheRoot: filepath.Join(t.TempDir(), "third")}
	if err := third.synchronize(ctx); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, filepath.Join(third.root, "one.md")); got != "updated" {
		t.Fatalf("updated = %q", got)
	}
	if _, err := os.Stat(filepath.Join(third.root, "removed.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("removed file stat = %v", err)
	}

	writeTestFile(t, filepath.Join(second.root, "one.md"), "stale")
	if err := second.publish("conflict"); !s3store.IsConflict(err) {
		t.Fatalf("stale publication = %v", err)
	}
	restored := &s3Backend{store: store, cacheRoot: second.cacheRoot}
	if err := restored.synchronize(ctx); err != nil || readTestFile(t, filepath.Join(restored.root, "one.md")) != "updated" {
		t.Fatalf("reader observed uncommitted cache: %v", err)
	}

	store.failWrite = "vault/current"
	writeTestFile(t, filepath.Join(third.root, "one.md"), "failed")
	if err := third.publish("failure"); err == nil {
		t.Fatal("failed current commit succeeded")
	}
	if string(store.objects["vault/current"]) != string(store.committedCurrent) {
		t.Fatal("failed publication changed current pointer")
	}
}

func TestS3VaultRejectsCorruptInterruptedAndUncommittedSnapshots(t *testing.T) {
	ctx := context.Background()
	store := newVaultFakeStore()
	writer := &s3Backend{store: store, cacheRoot: filepath.Join(t.TempDir(), "writer")}
	if err := writer.synchronize(ctx); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(writer.root, "page.md"), "committed")
	if err := writer.publish("commit"); err != nil {
		t.Fatal(err)
	}

	reader := &s3Backend{store: store, root: "last-usable", cacheRoot: filepath.Join(t.TempDir(), "reader")}
	store.failReadPrefix = "vault/objects/"
	if err := reader.synchronize(ctx); err == nil || reader.root != "last-usable" {
		t.Fatalf("interrupted sync = %v, root %q", err, reader.root)
	}
	store.failReadPrefix = ""

	var pointer currentPointer
	if err := json.Unmarshal(store.objects["vault/current"], &pointer); err != nil {
		t.Fatal(err)
	}
	var manifest vaultManifest
	if err := json.Unmarshal(store.objects[manifestKey(pointer.Manifest)], &manifest); err != nil {
		t.Fatal(err)
	}
	store.objects[objectKey(manifest.Entries[0].Digest)] = []byte("corrupt")
	if err := reader.synchronize(ctx); err == nil || reader.root != "last-usable" {
		t.Fatalf("corrupt sync = %v, root %q", err, reader.root)
	}
}

func TestS3BackedVaultReadsWritesAndIndexes(t *testing.T) {
	store := newVaultFakeStore()
	previous := openS3Store
	openS3Store = func(context.Context, s3store.Config) (vaultObjectStore, error) { return store, nil }
	t.Cleanup(func() { openS3Store = previous })
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	workspace := t.TempDir()
	config := `[vault]
vault_name = "team"
backend = "s3"
s3_bucket = "bucket"
s3_region = "us-east-1"
s3_prefix = "vault/team"
link_format = "relative"
vault_index = true
vault_log = true
`
	if err := os.WriteFile(filepath.Join(workspace, "gnosis.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	page := []byte(`---
type: Concept
title: S3 page
description: Stored in one committed S3 snapshot.
---

# S3 page
`)
	result, err := WriteDocument(context.Background(), workspace, "gnosis://team/concepts/s3-page.md", page, false)
	if err != nil || !result.Changed {
		t.Fatalf("write = %+v, %v", result, err)
	}
	read, err := ReadPage(workspace, "gnosis://team/concepts/s3-page.md")
	if err != nil || read.Document.Origin.Kind != OriginS3 || read.Document.Origin.Root != "s3://bucket/vault/team" {
		t.Fatalf("read = %+v, %v", read.Document, err)
	}
	history, err := ReadPageHistory(workspace, "gnosis://team/concepts/s3-page.md", "", 0)
	if err != nil || len(history.Entries) != 1 || history.Entries[0].Origin.Kind != OriginS3 || history.Entries[0].Origin.Root != "s3://bucket/vault/team" {
		t.Fatalf("history = %+v, %v", history, err)
	}
	beforeInvalid := string(store.objects["vault/current"])
	if _, err := WriteDocument(context.Background(), workspace, "gnosis://team/concepts/invalid.md", []byte("not markdown"), false); err == nil {
		t.Fatal("invalid page write succeeded")
	}
	if string(store.objects["vault/current"]) != beforeInvalid {
		t.Fatal("invalid page changed the committed snapshot")
	}
	written, enabled, err := GenerateWorkspaceIndexes(workspace, IndexOptions{Overwrite: true})
	if err != nil || !enabled || len(written) == 0 {
		t.Fatalf("indexes = %v, %t, %v", written, enabled, err)
	}
	var pointer currentPointer
	if err := json.Unmarshal(store.objects["vault/current"], &pointer); err != nil {
		t.Fatal(err)
	}
	var manifest vaultManifest
	if err := json.Unmarshal(store.objects[manifestKey(pointer.Manifest)], &manifest); err != nil {
		t.Fatal(err)
	}
	foundIndex := false
	for _, entry := range manifest.Entries {
		foundIndex = foundIndex || entry.Path == "concepts/index.md"
	}
	if !foundIndex {
		t.Fatalf("generated index missing from manifest: %+v", manifest.Entries)
	}
}

func TestS3VaultAndEvidenceConfigurationValidation(t *testing.T) {
	base := defaultConfig(t.TempDir())
	base.Vault = VaultConfig{Name: "team", Backend: s3BackendName, S3Bucket: "bucket", S3Region: "us-east-1", S3Prefix: "vault/team", LinkFormat: "relative"}
	if err := validateConfig(base, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Config){
		func(config *Config) { config.Vault.S3Bucket = "" },
		func(config *Config) { config.Vault.S3Region = "" },
		func(config *Config) { config.Vault.Root = "." },
		func(config *Config) { config.Vault.Repository = "owner/repo" },
		func(config *Config) { config.Vault.S3Prefix = "../bad" },
	} {
		config := base
		mutate(&config)
		if err := validateConfig(config, t.TempDir()); err == nil {
			t.Fatalf("invalid S3 vault config succeeded: %+v", config.Vault)
		}
	}

	base.GitHub = []GitHubConfig{{
		Repository: "owner/repo", EvidenceBackend: s3BackendName, S3Bucket: "bucket", S3Region: "us-east-1", TokenEnv: "GITHUB_TOKEN",
	}}
	if err := validateConfig(base, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	base.GitHub[0].EvidenceDir = t.TempDir()
	if err := validateConfig(base, t.TempDir()); err == nil {
		t.Fatal("S3 evidence config accepted evidence_dir")
	}
}

type vaultFakeStore struct {
	objects          map[string][]byte
	etags            map[string]string
	next             int
	failReadPrefix   string
	failWrite        string
	committedCurrent []byte
}

func newVaultFakeStore() *vaultFakeStore {
	return &vaultFakeStore{objects: map[string][]byte{}, etags: map[string]string{}, next: 1}
}

func (s *vaultFakeStore) Location() string { return "s3://bucket/prefix" }

func (s *vaultFakeStore) Read(_ context.Context, key string) ([]byte, string, error) {
	if s.failReadPrefix != "" && strings.HasPrefix(key, s.failReadPrefix) {
		return nil, "", errors.New("interrupted")
	}
	data, ok := s.objects[key]
	if !ok {
		return nil, "", fs.ErrNotExist
	}
	return append([]byte(nil), data...), s.etags[key], nil
}

func (s *vaultFakeStore) Create(_ context.Context, key string, data []byte) (bool, string, error) {
	if key == s.failWrite {
		return false, "", errors.New("write failed")
	}
	if existing, ok := s.objects[key]; ok {
		if string(existing) == string(data) {
			return false, s.etags[key], nil
		}
		return false, "", s3store.Conflict{Key: key}
	}
	s.next++
	etag := fmt.Sprintf("etag-%d", s.next)
	s.objects[key], s.etags[key] = append([]byte(nil), data...), etag
	if key == "vault/current" {
		s.committedCurrent = append([]byte(nil), data...)
	}
	return true, etag, nil
}

func (s *vaultFakeStore) Replace(_ context.Context, key string, data []byte, etag string) (string, error) {
	if key == s.failWrite {
		return "", errors.New("write failed")
	}
	if s.etags[key] != etag {
		return "", s3store.Conflict{Key: key}
	}
	s.next++
	result := fmt.Sprintf("etag-%d", s.next)
	s.objects[key], s.etags[key] = append([]byte(nil), data...), result
	if key == "vault/current" {
		s.committedCurrent = append([]byte(nil), data...)
	}
	return result, nil
}
