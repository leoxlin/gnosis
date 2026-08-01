package vault

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gnosis/internal/s3store"
)

const s3BackendName = "s3"

type vaultObjectStore interface {
	Read(context.Context, string) ([]byte, string, error)
	Create(context.Context, string, []byte) (bool, string, error)
	Replace(context.Context, string, []byte, string) (string, error)
	Location() string
}

type s3Backend struct {
	store       vaultObjectStore
	root        string
	cacheRoot   string
	manifest    string
	currentETag string
}

type vaultManifest struct {
	Version int             `json:"version"`
	Entries []manifestEntry `json:"entries"`
}

type manifestEntry struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type currentPointer struct {
	Version  int    `json:"version"`
	Manifest string `json:"manifest"`
}

var openS3Store = func(ctx context.Context, config s3store.Config) (vaultObjectStore, error) {
	return s3store.New(ctx, config)
}

func prepareS3Backend(config VaultConfig) (*s3Backend, error) {
	store, err := openS3Store(context.Background(), s3store.Config{
		Bucket: config.S3Bucket, Region: config.S3Region, Prefix: config.S3Prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("prepare S3 vault: %w", err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("S3 vault cache: %w", err)
	}
	identity := sha256.Sum256([]byte(store.Location()))
	backend := &s3Backend{
		store: store, cacheRoot: filepath.Join(cache, "gnosis", "s3-vaults", hex.EncodeToString(identity[:])),
	}
	if err := backend.synchronize(context.Background()); err != nil {
		return nil, fmt.Errorf("synchronize S3 vault: %w", err)
	}
	return backend, nil
}

func (b *s3Backend) preparedRoot() string { return b.root }

func (b *s3Backend) synchronize(ctx context.Context) error {
	data, etag, err := b.store.Read(ctx, "vault/current")
	if errors.Is(err, fs.ErrNotExist) {
		root := filepath.Join(b.cacheRoot, "empty")
		empty := vaultManifest{Version: 1, Entries: []manifestEntry{}}
		if err := verifyMaterialized(root, empty); err != nil {
			if err := b.materialize(ctx, root, empty); err != nil {
				return err
			}
		}
		b.root, b.manifest, b.currentETag = root, "", ""
		return nil
	}
	if err != nil {
		return err
	}
	pointer, err := decodeCurrent(data)
	if err != nil {
		return err
	}
	manifestData, _, err := b.store.Read(ctx, manifestKey(pointer.Manifest))
	if err != nil {
		return err
	}
	if digestBytes(manifestData) != pointer.Manifest {
		return fmt.Errorf("S3 vault manifest digest does not match current pointer")
	}
	manifest, err := decodeManifest(manifestData)
	if err != nil {
		return err
	}
	root := filepath.Join(b.cacheRoot, pointer.Manifest)
	if err := verifyMaterialized(root, manifest); err != nil {
		if err := b.materialize(ctx, root, manifest); err != nil {
			return err
		}
	}
	b.root, b.manifest, b.currentETag = root, pointer.Manifest, etag
	return nil
}

func (b *s3Backend) materialize(ctx context.Context, root string, manifest vaultManifest) error {
	if err := os.MkdirAll(b.cacheRoot, 0o700); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(b.cacheRoot, ".sync-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	for _, entry := range manifest.Entries {
		data, _, err := b.store.Read(ctx, objectKey(entry.Digest))
		if err != nil {
			return err
		}
		if int64(len(data)) != entry.Size || digestBytes(data) != entry.Digest {
			return fmt.Errorf("S3 vault object for %q failed digest or size verification", entry.Path)
		}
		destination := filepath.Join(temporary, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, 0o600); err != nil {
			return err
		}
	}
	backup := root + ".previous"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(root); err == nil {
		if err := os.Rename(root, backup); err != nil {
			return fmt.Errorf("replace S3 vault cache: %w", err)
		}
	}
	if err := os.Rename(temporary, root); err != nil {
		_ = os.Rename(backup, root)
		return fmt.Errorf("install S3 vault cache: %w", err)
	}
	_ = os.RemoveAll(backup)
	return nil
}

func (b *s3Backend) publish(_ string) error {
	manifest, manifestData, digest, err := buildManifest(b.root)
	if err != nil {
		return fmt.Errorf("publish S3 vault: %w", err)
	}
	if digest == b.manifest {
		return nil
	}
	ctx := context.Background()
	for _, entry := range manifest.Entries {
		data, err := os.ReadFile(filepath.Join(b.root, filepath.FromSlash(entry.Path)))
		if err != nil {
			return err
		}
		if _, _, err := b.store.Create(ctx, objectKey(entry.Digest), data); err != nil {
			return err
		}
	}
	if _, _, err := b.store.Create(ctx, manifestKey(digest), manifestData); err != nil {
		return err
	}
	pointerData, err := encodeCurrent(currentPointer{Version: 1, Manifest: digest})
	if err != nil {
		return err
	}
	var etag string
	if b.currentETag == "" {
		_, etag, err = b.store.Create(ctx, "vault/current", pointerData)
	} else {
		etag, err = b.store.Replace(ctx, "vault/current", pointerData, b.currentETag)
	}
	if err != nil {
		return fmt.Errorf("commit S3 vault: %w", err)
	}
	b.manifest, b.currentETag = digest, etag
	return nil
}

func buildManifest(root string) (vaultManifest, []byte, string, error) {
	manifest := vaultManifest{Version: 1, Entries: []manifestEntry{}}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && ignoredVaultDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("vault snapshot path %s is not a regular file", path)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if err := validateManifestPath(relative); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		manifest.Entries = append(manifest.Entries, manifestEntry{Path: relative, Digest: digestBytes(data), Size: int64(len(data))})
		return nil
	})
	if err != nil {
		return vaultManifest{}, nil, "", err
	}
	sort.Slice(manifest.Entries, func(i, j int) bool { return manifest.Entries[i].Path < manifest.Entries[j].Path })
	data, err := encodeManifest(manifest)
	if err != nil {
		return vaultManifest{}, nil, "", err
	}
	return manifest, data, digestBytes(data), nil
}

func encodeManifest(manifest vaultManifest) ([]byte, error) {
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	data, err := json.Marshal(manifest)
	return append(data, '\n'), err
}

func decodeManifest(data []byte) (vaultManifest, error) {
	var manifest vaultManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse S3 vault manifest: %w", err)
	}
	return manifest, validateManifest(manifest)
}

func validateManifest(manifest vaultManifest) error {
	if manifest.Version != 1 {
		return fmt.Errorf("unsupported S3 vault manifest version %d", manifest.Version)
	}
	previous := ""
	for _, entry := range manifest.Entries {
		if err := validateManifestPath(entry.Path); err != nil {
			return err
		}
		if entry.Path <= previous {
			return fmt.Errorf("S3 vault manifest paths must be sorted and unique")
		}
		if !validDigest(entry.Digest) || entry.Size < 0 {
			return fmt.Errorf("S3 vault manifest entry %q has an invalid digest or size", entry.Path)
		}
		previous = entry.Path
	}
	return nil
}

func validateManifestPath(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || filepath.ToSlash(filepath.Clean(value)) != value || strings.Contains(value, "//") {
		return fmt.Errorf("unsafe S3 vault manifest path %q", value)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("unsafe S3 vault manifest path %q", value)
		}
	}
	return nil
}

func encodeCurrent(pointer currentPointer) ([]byte, error) {
	if pointer.Version != 1 || !validDigest(pointer.Manifest) {
		return nil, fmt.Errorf("invalid S3 vault current pointer")
	}
	data, err := json.Marshal(pointer)
	return append(data, '\n'), err
}

func decodeCurrent(data []byte) (currentPointer, error) {
	var pointer currentPointer
	if err := json.Unmarshal(data, &pointer); err != nil {
		return pointer, fmt.Errorf("parse S3 vault current pointer: %w", err)
	}
	_, err := encodeCurrent(pointer)
	return pointer, err
}

func verifyMaterialized(root string, manifest vaultManifest) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("cached S3 vault %s is not a directory", root)
	}
	expected := make(map[string]manifestEntry, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		expected[entry.Path] = entry
	}
	seen := 0
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		want, ok := expected[relative]
		if !ok {
			return fmt.Errorf("cached S3 vault contains uncommitted path %q", relative)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if int64(len(data)) != want.Size || digestBytes(data) != want.Digest {
			return fmt.Errorf("cached S3 vault object %q failed verification", relative)
		}
		seen++
		return nil
	})
	if err != nil {
		return err
	}
	if seen != len(expected) {
		return fmt.Errorf("cached S3 vault is incomplete")
	}
	return nil
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func objectKey(digest string) string   { return "vault/objects/" + digest }
func manifestKey(digest string) string { return "vault/manifests/" + digest + ".json" }

func s3Location(config VaultConfig) string {
	location := "s3://" + strings.TrimSpace(config.S3Bucket)
	if prefix, _ := s3store.NormalizePrefix(config.S3Prefix); prefix != "" {
		location += "/" + prefix
	}
	return location
}
