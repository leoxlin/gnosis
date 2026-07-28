package vault

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	DefaultHistoryLimit = 20
	MaxHistoryLimit     = 100
	DefaultDiffLimit    = 20_000
	MaxDiffLimit        = 100_000
)

var (
	ErrInvalidHistoryCursor = errors.New("invalid history cursor")
	ErrCursorExpired        = errors.New("history cursor expired")
	ErrRevisionNotFound     = errors.New("revision not found")
)

// ChangeClassification describes a page lifecycle transition.
type ChangeClassification string

const (
	ChangeWorking            ChangeClassification = "working"
	ChangeAdded              ChangeClassification = "added"
	ChangeUpdated            ChangeClassification = "updated"
	ChangeArchived           ChangeClassification = "archived"
	ChangeSuperseded         ChangeClassification = "superseded"
	ChangeEffectiveRemoved   ChangeClassification = "effective_removed"
	ChangeOriginReplaced     ChangeClassification = "origin_replaced"
	ChangeHistoryUnavailable ChangeClassification = "history_unavailable"
)

// ResultBound reports the requested bound and whether more data was omitted.
type ResultBound struct {
	Limit     int  `json:"limit"`
	Truncated bool `json:"truncated"`
}

// HistoryEntry is one committed or current working revision of a page.
type HistoryEntry struct {
	URI            string               `json:"uri"`
	Commit         string               `json:"commit,omitempty"`
	Revision       string               `json:"revision"`
	Timestamp      string               `json:"timestamp,omitempty"`
	Actor          string               `json:"actor,omitempty"`
	Classification ChangeClassification `json:"classification"`
	Working        bool                 `json:"working,omitempty"`
	Origin         Origin               `json:"origin"`
}

// PageHistoryResult is bounded newest-first history for one canonical page.
type PageHistoryResult struct {
	URI        string         `json:"uri"`
	Current    string         `json:"current"`
	Entries    []HistoryEntry `json:"entries"`
	NextCursor string         `json:"next_cursor,omitempty"`
	Bound      ResultBound    `json:"bound"`
}

// PageDiffResult is a deterministic bounded Markdown diff between exact revisions.
type PageDiffResult struct {
	URI            string               `json:"uri"`
	FromRevision   string               `json:"from_revision"`
	ToRevision     string               `json:"to_revision"`
	Classification ChangeClassification `json:"classification"`
	Diff           string               `json:"diff"`
	Characters     int                  `json:"characters"`
	Bound          ResultBound          `json:"bound"`
}

// VaultChange is one net committed effective-view transition.
type VaultChange struct {
	URI            string               `json:"uri"`
	PreviousURI    string               `json:"previous_uri,omitempty"`
	Classification ChangeClassification `json:"classification"`
	BeforeRevision string               `json:"before_revision,omitempty"`
	AfterRevision  string               `json:"after_revision,omitempty"`
	BeforeOrigin   *Origin              `json:"before_origin,omitempty"`
	AfterOrigin    *Origin              `json:"after_origin,omitempty"`
}

// ChangeFeedResult is one deterministic bounded page of committed changes.
type ChangeFeedResult struct {
	Changes    []VaultChange `json:"changes"`
	NextCursor string        `json:"next_cursor"`
	Bound      ResultBound   `json:"bound"`
}

type gitRepository struct {
	root        string
	fingerprint string
}

type gitHistoryRecord struct {
	commit    string
	timestamp string
	actor     string
	data      []byte
}

type historyCursor struct {
	Version    int    `json:"v"`
	Kind       string `json:"kind"`
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Position   string `json:"position"`
}

type changeCursorSource struct {
	Key        string `json:"key"`
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
}

type changeCursor struct {
	Version     int                  `json:"v"`
	Kind        string               `json:"kind"`
	Composition string               `json:"composition"`
	Base        []changeCursorSource `json:"base"`
	Target      []changeCursorSource `json:"target,omitempty"`
	Offset      int                  `json:"offset,omitempty"`
}

type historyTarget struct {
	uri     string
	origin  Origin
	path    string
	data    []byte
	current bool
	repo    gitRepository
	relPath string
}

// ReadPageHistory returns bounded newest-first history for one canonical URI.
func ReadPageHistory(root, uri, cursor string, limit int) (PageHistoryResult, error) {
	limit, err := boundedLimit(limit)
	if err != nil {
		return PageHistoryResult{}, err
	}
	target, err := resolveHistoryTarget(root, uri)
	if err != nil {
		return PageHistoryResult{}, err
	}
	result := PageHistoryResult{
		URI:     target.uri,
		Current: "absent",
		Entries: []HistoryEntry{},
		Bound:   ResultBound{Limit: limit},
	}
	if target.current {
		result.Current = "present"
	}
	if target.repo.root == "" {
		result.Current = "history_unavailable"
		if target.current {
			result.Entries = append(result.Entries, HistoryEntry{
				URI:            target.uri,
				Revision:       documentRevision(target.data),
				Classification: ChangeHistoryUnavailable,
				Working:        true,
				Origin:         target.origin,
			})
		}
		return result, nil
	}

	records, err := pageGitHistory(target.repo.root, target.relPath)
	if err != nil {
		return PageHistoryResult{}, err
	}
	start := 0
	if cursor != "" {
		var payload historyCursor
		if err := decodeCursor(cursor, &payload); err != nil ||
			payload.Version != 1 || payload.Kind != "page-history" ||
			payload.Repository != target.repo.fingerprint || payload.Path != target.relPath {
			return PageHistoryResult{}, ErrInvalidHistoryCursor
		}
		start = recordIndex(records, payload.Position)
		if start < 0 {
			return PageHistoryResult{}, ErrCursorExpired
		}
		start++
	}

	if cursor == "" && target.current &&
		(len(records) == 0 || documentRevision(target.data) != documentRevision(records[0].data)) {
		result.Entries = append(result.Entries, HistoryEntry{
			URI:            target.uri,
			Revision:       documentRevision(target.data),
			Classification: ChangeWorking,
			Working:        true,
			Origin:         target.origin,
		})
	}
	end := min(start+limit, len(records))
	for index := start; index < end; index++ {
		classification := ChangeAdded
		if index+1 < len(records) {
			classification = historyLifecycle(records[index+1].data, records[index].data)
		}
		result.Entries = append(result.Entries, HistoryEntry{
			URI:            target.uri,
			Commit:         records[index].commit,
			Revision:       documentRevision(records[index].data),
			Timestamp:      records[index].timestamp,
			Actor:          records[index].actor,
			Classification: classification,
			Origin:         target.origin,
		})
	}
	if end < len(records) {
		result.Bound.Truncated = true
		result.NextCursor, err = encodeCursor(historyCursor{
			Version: 1, Kind: "page-history", Repository: target.repo.fingerprint,
			Path: target.relPath, Position: records[end-1].commit,
		})
		if err != nil {
			return PageHistoryResult{}, err
		}
	}
	return result, nil
}

// DiffPage returns a deterministic bounded diff between two exact content revisions.
func DiffPage(root, uri, fromRevision, toRevision string, limit int) (PageDiffResult, error) {
	if limit == 0 {
		limit = DefaultDiffLimit
	}
	if limit < 1 || limit > MaxDiffLimit {
		return PageDiffResult{}, fmt.Errorf("diff limit must be between 1 and %d", MaxDiffLimit)
	}
	target, err := resolveHistoryTarget(root, uri)
	if err != nil {
		return PageDiffResult{}, err
	}
	revisions := map[string][]byte{}
	if target.current {
		revisions[documentRevision(target.data)] = target.data
	}
	records := []gitHistoryRecord{}
	if target.repo.root != "" {
		records, err = pageGitHistory(target.repo.root, target.relPath)
		if err != nil {
			return PageDiffResult{}, err
		}
		for _, record := range records {
			revisions[documentRevision(record.data)] = record.data
		}
	}
	before, ok := revisions[fromRevision]
	if !ok {
		return PageDiffResult{}, fmt.Errorf("%w: %s", ErrRevisionNotFound, fromRevision)
	}
	after, ok := revisions[toRevision]
	if !ok {
		return PageDiffResult{}, fmt.Errorf("%w: %s", ErrRevisionNotFound, toRevision)
	}
	diff := markdownDiff(before, after)
	total := utf8.RuneCountInString(diff)
	truncated := total > limit
	if truncated {
		diff = string([]rune(diff)[:limit])
	}
	return PageDiffResult{
		URI:            target.uri,
		FromRevision:   fromRevision,
		ToRevision:     toRevision,
		Classification: historyLifecycle(before, after),
		Diff:           diff,
		Characters:     total,
		Bound:          ResultBound{Limit: limit, Truncated: truncated},
	}, nil
}

// ChangesSince returns committed effective-view changes after an opaque cursor.
// An empty cursor establishes a baseline and returns no changes.
func ChangesSince(root, cursor string, limit int) (ChangeFeedResult, error) {
	limit, err := boundedLimit(limit)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	sources, composition, err := cursorSources(vault)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	if cursor == "" {
		next, err := encodeCursor(changeCursor{
			Version: 1, Kind: "change-feed", Composition: composition, Base: sources,
		})
		return ChangeFeedResult{
			Changes: []VaultChange{}, NextCursor: next, Bound: ResultBound{Limit: limit},
		}, err
	}

	var payload changeCursor
	if err := decodeCursor(cursor, &payload); err != nil ||
		payload.Version != 1 || payload.Kind != "change-feed" || payload.Offset < 0 {
		return ChangeFeedResult{}, ErrInvalidHistoryCursor
	}
	if payload.Composition != composition || !sameCursorRepositories(payload.Base, sources) {
		return ChangeFeedResult{}, ErrCursorExpired
	}
	target := payload.Target
	if len(target) == 0 {
		target = sources
	} else if !sameCursorRepositories(target, sources) {
		return ChangeFeedResult{}, ErrCursorExpired
	}
	if err := validateCursorRange(vault, payload.Base, target); err != nil {
		return ChangeFeedResult{}, err
	}
	before, err := committedSnapshot(vault, payload.Base)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	after, err := committedSnapshot(vault, target)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	changes := compareSnapshots(before, after)
	if payload.Offset > len(changes) {
		return ChangeFeedResult{}, ErrInvalidHistoryCursor
	}
	end := min(payload.Offset+limit, len(changes))
	page := changes[payload.Offset:end]
	nextPayload := changeCursor{
		Version: 1, Kind: "change-feed", Composition: composition, Base: target,
	}
	truncated := end < len(changes)
	if truncated {
		nextPayload.Base = payload.Base
		nextPayload.Target = target
		nextPayload.Offset = end
	}
	next, err := encodeCursor(nextPayload)
	if err != nil {
		return ChangeFeedResult{}, err
	}
	return ChangeFeedResult{
		Changes: page, NextCursor: next, Bound: ResultBound{Limit: limit, Truncated: truncated},
	}, nil
}

func boundedLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultHistoryLimit, nil
	}
	if limit < 1 || limit > MaxHistoryLimit {
		return 0, fmt.Errorf("limit must be between 1 and %d", MaxHistoryLimit)
	}
	return limit, nil
}

func resolveHistoryTarget(root, uri string) (historyTarget, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return historyTarget{}, err
	}
	pages, err := vault.pages()
	if err != nil {
		return historyTarget{}, err
	}
	target := historyTarget{uri: uri}
	if page, ok := selectPage(pages, uri); ok {
		target.uri = page.document.URI
		target.origin = page.document.Origin
		target.path = page.path
		target.data = page.data
		target.current = true
	} else {
		vaultName, pagePath, ok := canonicalGnosisParts(uri)
		if !ok {
			return historyTarget{}, fmt.Errorf("history: uri must be a canonical gnosis URI")
		}
		for _, source := range vault.sources {
			if vaultName != anyVaultAuthority && source.config.Vault.Name != vaultName {
				continue
			}
			path := filepath.Clean(filepath.Join(source.path, filepath.FromSlash(pagePath)))
			relative, err := filepath.Rel(source.path, path)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			kind := OriginImport
			if source.vaultRoot == vault.root {
				kind = OriginLocal
			}
			target.origin = Origin{
				Vault: source.config.Vault.Name, Kind: kind, Root: source.path, Path: path,
			}
			target.path = path
			target.uri = documentURI(source.config.Vault.Name, pagePath)
			break
		}
		if target.path == "" {
			return historyTarget{}, fmt.Errorf("%w with URI %q", ErrPageNotFound, uri)
		}
	}
	repo, ok := repositoryAt(filepath.Dir(target.path))
	if !ok {
		return target, nil
	}
	relative, err := filepath.Rel(repo.root, target.path)
	if err != nil {
		return historyTarget{}, err
	}
	target.repo = repo
	target.relPath = filepath.ToSlash(relative)
	return target, nil
}

func repositoryAt(path string) (gitRepository, bool) {
	root, err := gitCommandOutput("-C", path, "rev-parse", "--show-toplevel")
	if err != nil {
		return gitRepository{}, false
	}
	root = filepath.Clean(strings.TrimSpace(root))
	identity := root
	if remote, err := gitCommandOutput("-C", root, "config", "--get", "remote.origin.url"); err == nil {
		identity = strings.TrimSpace(remote)
	}
	return gitRepository{root: root, fingerprint: digestString("repository\x00" + identity)}, true
}

func pageGitHistory(repoRoot, relativePath string) ([]gitHistoryRecord, error) {
	output, err := gitCommandOutput(
		"-C", repoRoot, "log", "--format=%H%x1f%cI%x1f%an <%ae>",
		"--diff-filter=AM", "--follow", "--", relativePath,
	)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	records := make([]gitHistoryRecord, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.SplitN(line, "\x1f", 3)
		if len(fields) != 3 {
			return nil, fmt.Errorf("read page history: invalid git log record")
		}
		data, err := gitCommandOutput("-C", repoRoot, "show", fields[0]+":"+relativePath)
		if err != nil {
			return nil, err
		}
		records = append(records, gitHistoryRecord{
			commit: fields[0], timestamp: fields[1], actor: fields[2], data: []byte(data),
		})
	}
	return records, nil
}

func recordIndex(records []gitHistoryRecord, commit string) int {
	for index, record := range records {
		if record.commit == commit {
			return index
		}
	}
	return -1
}

func historyLifecycle(before, after []byte) ChangeClassification {
	beforeFields := historyFields(before)
	afterFields := historyFields(after)
	beforeStatus := strings.TrimSpace(fmt.Sprint(beforeFields["status"]))
	afterStatus := strings.TrimSpace(fmt.Sprint(afterFields["status"]))
	if beforeStatus != afterStatus && (afterStatus == "archived" || afterStatus == "retired") {
		return ChangeArchived
	}
	beforeSuperseded := strings.TrimSpace(fmt.Sprint(beforeFields["superseded_by"]))
	afterSuperseded := strings.TrimSpace(fmt.Sprint(afterFields["superseded_by"]))
	if beforeSuperseded != afterSuperseded && afterSuperseded != "" {
		return ChangeSuperseded
	}
	return ChangeUpdated
}

func historyFields(data []byte) frontmatterFields {
	parsed, err := parsePage(data)
	if err != nil {
		return frontmatterFields{}
	}
	return parsed.fields
}

func encodeCursor(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeCursor(cursor string, target any) error {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("cursor contains trailing data")
	}
	return nil
}

func digestString(value string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(value)))
}

type sourceCursorDescriptor struct {
	source vaultSource
	key    string
	repo   gitRepository
	prefix string
	kind   OriginKind
}

func cursorSources(vault *effectiveVault) ([]changeCursorSource, string, error) {
	descriptors, composition, err := sourceCursorDescriptors(vault)
	if err != nil {
		return nil, "", err
	}
	result := make([]changeCursorSource, 0, len(descriptors))
	for _, descriptor := range descriptors {
		head, err := gitCommandOutput("-C", descriptor.repo.root, "rev-parse", "HEAD")
		if err != nil {
			return nil, "", err
		}
		result = append(result, changeCursorSource{
			Key: descriptor.key, Repository: descriptor.repo.fingerprint,
			Commit: strings.TrimSpace(head),
		})
	}
	return result, composition, nil
}

func sourceCursorDescriptors(vault *effectiveVault) ([]sourceCursorDescriptor, string, error) {
	descriptors := []sourceCursorDescriptor{}
	compositionParts := []string{"effective-vault-v1"}
	for precedence, source := range vault.sources {
		repo, ok := repositoryAt(source.path)
		if !ok {
			compositionParts = append(compositionParts,
				fmt.Sprintf("%d\x00%s\x00non-git\x00%s", precedence, source.config.Vault.Name, source.path),
			)
			continue
		}
		prefix, err := filepath.Rel(repo.root, source.path)
		if err != nil {
			return nil, "", err
		}
		prefix = filepath.ToSlash(prefix)
		if prefix == "." {
			prefix = ""
		}
		key := digestString(fmt.Sprintf(
			"source\x00%d\x00%s\x00%s\x00%s",
			precedence, source.config.Vault.Name, repo.fingerprint, prefix,
		))
		descriptors = append(descriptors, sourceCursorDescriptor{
			source: source, key: key, repo: repo, prefix: prefix,
			kind: func() OriginKind {
				if source.vaultRoot == vault.root {
					return OriginLocal
				}
				return OriginImport
			}(),
		})
		compositionParts = append(compositionParts, key)
	}
	return descriptors, digestString(strings.Join(compositionParts, "\n")), nil
}

func sameCursorRepositories(left, right []changeCursorSource) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Key != right[index].Key ||
			left[index].Repository != right[index].Repository {
			return false
		}
	}
	return true
}

func validateCursorRange(
	vault *effectiveVault,
	base, target []changeCursorSource,
) error {
	if len(base) != len(target) {
		return ErrCursorExpired
	}
	descriptors, _, err := sourceCursorDescriptors(vault)
	if err != nil {
		return err
	}
	repositories := make(map[string]string, len(descriptors))
	for _, descriptor := range descriptors {
		repositories[descriptor.repo.fingerprint] = descriptor.repo.root
	}
	for index := range base {
		if base[index].Key != target[index].Key ||
			base[index].Repository != target[index].Repository {
			return ErrCursorExpired
		}
		repoRoot, ok := repositories[base[index].Repository]
		if !ok {
			return ErrCursorExpired
		}
		if err := validateCommit(repoRoot, base[index].Commit); err != nil {
			return err
		}
		if err := validateCommit(repoRoot, target[index].Commit); err != nil {
			return err
		}
		if _, err := gitCommandOutput(
			"-C", repoRoot, "merge-base", "--is-ancestor",
			base[index].Commit, target[index].Commit,
		); err != nil {
			return ErrCursorExpired
		}
	}
	return nil
}

type snapshotPage struct {
	relative string
	uri      string
	data     []byte
	revision string
	origin   Origin
}

func committedSnapshot(vault *effectiveVault, cursor []changeCursorSource) (map[string]snapshotPage, error) {
	descriptors, _, err := sourceCursorDescriptors(vault)
	if err != nil {
		return nil, err
	}
	commits := make(map[string]string, len(cursor))
	for _, source := range cursor {
		commits[source.Key] = source.Commit
	}
	result := map[string]snapshotPage{}
	for precedence, source := range vault.sources {
		var descriptor *sourceCursorDescriptor
		for index := range descriptors {
			if descriptors[index].source.path == source.path {
				descriptor = &descriptors[index]
				break
			}
		}
		if descriptor == nil {
			kind := OriginImport
			if source.vaultRoot == vault.root {
				kind = OriginLocal
			}
			if err := appendWorkingSnapshot(result, source, precedence, kind); err != nil {
				return nil, err
			}
			continue
		}
		commit, ok := commits[descriptor.key]
		if !ok {
			return nil, ErrCursorExpired
		}
		if err := validateCommit(descriptor.repo.root, commit); err != nil {
			return nil, err
		}
		if err := appendGitSnapshot(result, *descriptor, commit, precedence); err != nil {
			return nil, err
		}
	}
	if err := appendBundleSnapshot(result, len(vault.sources)); err != nil {
		return nil, err
	}
	return result, nil
}

func validateCommit(repoRoot, commit string) error {
	if strings.TrimSpace(commit) == "" {
		return ErrInvalidHistoryCursor
	}
	if _, err := gitCommandOutput("-C", repoRoot, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return ErrCursorExpired
	}
	return nil
}

func appendGitSnapshot(
	result map[string]snapshotPage,
	descriptor sourceCursorDescriptor,
	commit string,
	precedence int,
) error {
	args := []string{"-C", descriptor.repo.root, "ls-tree", "-r", "--name-only", commit}
	if descriptor.prefix != "" {
		args = append(args, "--", descriptor.prefix)
	}
	output, err := gitCommandOutput(args...)
	if err != nil {
		return err
	}
	for _, repoPath := range strings.Split(strings.TrimSpace(output), "\n") {
		if repoPath == "" || filepath.Ext(repoPath) != ".md" || reservedPageName(filepath.Base(repoPath)) {
			continue
		}
		relative := repoPath
		if descriptor.prefix != "" {
			var found bool
			relative, found = strings.CutPrefix(repoPath, descriptor.prefix+"/")
			if !found {
				continue
			}
		}
		if _, exists := result[relative]; exists {
			continue
		}
		data, err := gitCommandOutput("-C", descriptor.repo.root, "show", commit+":"+repoPath)
		if err != nil {
			return err
		}
		origin := Origin{
			Vault: descriptor.source.config.Vault.Name, Kind: descriptor.kind,
			Root:       descriptor.source.path,
			Path:       filepath.Join(descriptor.source.path, filepath.FromSlash(relative)),
			Precedence: precedence,
		}
		result[relative] = snapshotPage{
			relative: relative,
			uri:      documentURI(origin.Vault, relative),
			data:     []byte(data), revision: documentRevision([]byte(data)), origin: origin,
		}
	}
	return nil
}

func appendWorkingSnapshot(
	result map[string]snapshotPage,
	source vaultSource,
	precedence int,
	kind OriginKind,
) error {
	return filepath.WalkDir(source.path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != source.path && ignoredVaultDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".md" || reservedPageName(entry.Name()) {
			return nil
		}
		relative, err := filepath.Rel(source.path, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, exists := result[relative]; exists {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		origin := Origin{
			Vault: source.config.Vault.Name, Kind: kind, Root: source.path,
			Path: path, Precedence: precedence,
		}
		result[relative] = snapshotPage{
			relative: relative, uri: documentURI(origin.Vault, relative), data: data,
			revision: documentRevision(data), origin: origin,
		}
		return nil
	})
}

func appendBundleSnapshot(result map[string]snapshotPage, precedence int) error {
	documents, err := bundledDocuments()
	if err != nil {
		return err
	}
	for _, document := range documents {
		relative := filepath.ToSlash(filepath.Clean(document.Path))
		if _, exists := result[relative]; exists {
			continue
		}
		origin := Origin{
			Vault: "core", Kind: OriginBundle, Path: document.Path, Precedence: precedence,
		}
		result[relative] = snapshotPage{
			relative: relative, uri: documentURI(origin.Vault, relative), data: document.Data,
			revision: documentRevision(document.Data), origin: origin,
		}
	}
	return nil
}

func compareSnapshots(before, after map[string]snapshotPage) []VaultChange {
	paths := make(map[string]struct{}, len(before)+len(after))
	for path := range before {
		paths[path] = struct{}{}
	}
	for path := range after {
		paths[path] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	changes := []VaultChange{}
	for _, path := range ordered {
		old, hadOld := before[path]
		current, hasCurrent := after[path]
		switch {
		case !hadOld:
			origin := current.origin
			changes = append(changes, VaultChange{
				URI: current.uri, Classification: ChangeAdded,
				AfterRevision: current.revision, AfterOrigin: &origin,
			})
		case !hasCurrent:
			origin := old.origin
			changes = append(changes, VaultChange{
				URI: old.uri, Classification: ChangeEffectiveRemoved,
				BeforeRevision: old.revision, BeforeOrigin: &origin,
			})
		case old.origin.Vault != current.origin.Vault ||
			old.origin.Root != current.origin.Root:
			beforeOrigin, afterOrigin := old.origin, current.origin
			changes = append(changes, VaultChange{
				URI: current.uri, PreviousURI: old.uri, Classification: ChangeOriginReplaced,
				BeforeRevision: old.revision, AfterRevision: current.revision,
				BeforeOrigin: &beforeOrigin, AfterOrigin: &afterOrigin,
			})
		case old.revision != current.revision:
			beforeOrigin, afterOrigin := old.origin, current.origin
			changes = append(changes, VaultChange{
				URI: current.uri, Classification: historyLifecycle(old.data, current.data),
				BeforeRevision: old.revision, AfterRevision: current.revision,
				BeforeOrigin: &beforeOrigin, AfterOrigin: &afterOrigin,
			})
		}
	}
	return changes
}
