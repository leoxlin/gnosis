package vault

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageHistoryAndDiff(t *testing.T) {
	root := newHistoryRepository(t)
	path := filepath.Join(root, "docs", "note.md")
	first := historyMarkdown("active", "", "first")
	second := historyMarkdown("archived", "", "second")
	writeTestFile(t, path, first)
	commitHistoryFixture(t, root, "add note")
	writeTestFile(t, path, second)
	commitHistoryFixture(t, root, "archive note")
	working := historyMarkdown("archived", "", "working")
	writeTestFile(t, path, working)

	history, err := ReadPageHistory(root, "gnosis://local/note.md", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if history.Current != "present" || len(history.Entries) != 2 ||
		history.Entries[0].Classification != ChangeWorking ||
		history.Entries[1].Classification != ChangeArchived ||
		history.NextCursor == "" || !history.Bound.Truncated {
		t.Fatalf("history = %+v", history)
	}
	continued, err := ReadPageHistory(root, "gnosis://local/note.md", history.NextCursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(continued.Entries) != 1 ||
		continued.Entries[0].Classification != ChangeAdded ||
		continued.NextCursor != "" {
		t.Fatalf("continued history = %+v", continued)
	}

	diff, err := DiffPage(
		root,
		"gnosis://local/note.md",
		documentRevision([]byte(first)),
		documentRevision([]byte(working)),
		80,
	)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Classification != ChangeArchived ||
		!strings.Contains(diff.Diff, "-status: active") ||
		!diff.Bound.Truncated {
		t.Fatalf("diff = %+v", diff)
	}
	_, err = DiffPage(root, "gnosis://local/note.md", "sha256:missing", diff.ToRevision, 0)
	if !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("unknown revision error = %v", err)
	}
}

func TestPageHistoryReportsUnavailableAndAbsent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	writeConfig(t, root, "[vault]\nvault_name = \"plain\"\nvault_root = \"docs\"\n")
	writeTestFile(t, filepath.Join(root, "docs", "note.md"), historyMarkdown("active", "", "plain"))

	history, err := ReadPageHistory(root, "gnosis://plain/note.md", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if history.Current != "history_unavailable" || len(history.Entries) != 1 ||
		history.Entries[0].Classification != ChangeHistoryUnavailable {
		t.Fatalf("history = %+v", history)
	}

	root = newHistoryRepository(t)
	path := filepath.Join(root, "docs", "gone.md")
	writeTestFile(t, path, historyMarkdown("active", "", "gone"))
	commitHistoryFixture(t, root, "add gone")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	commitHistoryFixture(t, root, "remove gone")
	history, err = ReadPageHistory(root, "gnosis://local/gone.md", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if history.Current != "absent" || len(history.Entries) != 1 {
		t.Fatalf("absent history = %+v", history)
	}
}

func TestPageHistoryUsesRefreshedRemoteCache(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://history.example.test/team/vault.git")
	configureTestRemoteVault(t, fixture)
	writeTestFile(t, filepath.Join(fixture.seed, "note.md"), historyMarkdown("active", "", "remote update"))
	commitHistoryFixture(t, fixture.seed, "update remote")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")

	history, err := ReadPageHistory(fixture.url, "gnosis://remote/note.md", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Entries) < 2 || history.Entries[0].Commit == "" {
		t.Fatalf("remote history = %+v", history)
	}
}

func TestChangesSinceClassifiesCommittedEffectiveView(t *testing.T) {
	root := newHistoryRepository(t)
	baseline, err := ChangesSince(root, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "docs", "note.md")
	writeTestFile(t, path, historyMarkdown("active", "", "first"))
	commitHistoryFixture(t, root, "add note")
	added, err := ChangesSince(root, baseline.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, added, ChangeAdded)

	writeTestFile(t, path, historyMarkdown("active", "", "second"))
	commitHistoryFixture(t, root, "update note")
	updated, err := ChangesSince(root, added.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, updated, ChangeUpdated)

	writeTestFile(t, path, historyMarkdown("archived", "", "second"))
	commitHistoryFixture(t, root, "archive note")
	archived, err := ChangesSince(root, updated.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, archived, ChangeArchived)

	writeTestFile(t, path, historyMarkdown(
		"active",
		"gnosis://local/replacement.md",
		"second",
	))
	commitHistoryFixture(t, root, "supersede note")
	superseded, err := ChangesSince(root, archived.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, superseded, ChangeSuperseded)

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	commitHistoryFixture(t, root, "remove note")
	removed, err := ChangesSince(root, superseded.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, removed, ChangeEffectiveRemoved)
}

func TestChangesSinceClassifiesEffectiveOriginReplacement(t *testing.T) {
	requireGit(t)
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	imported := filepath.Join(root, "imported")
	runGit(t, "init", "--initial-branch=main", root)
	runGit(t, "-C", root, "config", "user.name", "gnosis test")
	runGit(t, "-C", root, "config", "user.email", "gnosis@example.test")
	runGit(t, "init", "--initial-branch=main", imported)
	runGit(t, "-C", imported, "config", "user.name", "gnosis test")
	runGit(t, "-C", imported, "config", "user.email", "gnosis@example.test")
	writeConfig(t, root, `[vault]
vault_name = "local"
vault_root = "docs"

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "imported"
vault_root = "docs"
`)
	writeTestFile(
		t,
		filepath.Join(imported, "docs", "shared.md"),
		historyMarkdown("active", "", "imported"),
	)
	commitHistoryFixture(t, imported, "add imported page")
	writeTestFile(
		t,
		filepath.Join(root, "docs", "shared.md"),
		historyMarkdown("active", "", "local"),
	)
	commitHistoryFixture(t, root, "add local page")

	baseline, err := ChangesSince(root, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "docs", "shared.md")); err != nil {
		t.Fatal(err)
	}
	commitHistoryFixture(t, root, "reveal imported page")
	result, err := ChangesSince(root, baseline.NextCursor, 0)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleHistoryChange(t, result, ChangeOriginReplaced)
	if result.Changes[0].PreviousURI != "gnosis://local/shared.md" ||
		result.Changes[0].URI != "gnosis://imported/shared.md" {
		t.Fatalf("origin replacement = %+v", result.Changes[0])
	}
}

func TestChangesSincePaginatesAndExpiresRewrittenHistory(t *testing.T) {
	root := newHistoryRepository(t)
	baseline, err := ChangesSince(root, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one", "two"} {
		writeTestFile(
			t,
			filepath.Join(root, "docs", name+".md"),
			historyMarkdown("active", "", name),
		)
	}
	commitHistoryFixture(t, root, "add two")
	first, err := ChangesSince(root, baseline.NextCursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Changes) != 1 || !first.Bound.Truncated {
		t.Fatalf("first page = %+v", first)
	}
	second, err := ChangesSince(root, first.NextCursor, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Changes) != 1 || second.Bound.Truncated {
		t.Fatalf("second page = %+v", second)
	}

	rewrittenBaseline, err := ChangesSince(root, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", root, "reset", "--hard", "HEAD~1")
	writeTestFile(t, filepath.Join(root, "docs", "replacement.md"), historyMarkdown("active", "", "replacement"))
	commitHistoryFixture(t, root, "replace history")
	_, err = ChangesSince(root, rewrittenBaseline.NextCursor, 0)
	if !errors.Is(err, ErrCursorExpired) {
		t.Fatalf("rewritten cursor error = %v", err)
	}
}

func newHistoryRepository(t *testing.T) string {
	t.Helper()
	requireGit(t)
	t.Setenv("HOME", t.TempDir())
	root := t.TempDir()
	runGit(t, "init", "--initial-branch=main", root)
	runGit(t, "-C", root, "config", "user.name", "gnosis test")
	runGit(t, "-C", root, "config", "user.email", "gnosis@example.test")
	writeConfig(t, root, "[vault]\nvault_name = \"local\"\nvault_root = \"docs\"\n")
	writeTestFile(t, filepath.Join(root, "docs", ".keep"), "")
	runGit(t, "-C", root, "add", ".")
	runGit(t, "-C", root, "commit", "--allow-empty", "-m", "initial")
	return root
}

func commitHistoryFixture(t *testing.T, root, message string) {
	t.Helper()
	runGit(t, "-C", root, "add", "--all")
	runGit(t, "-C", root, "commit", "-m", message)
}

func historyMarkdown(status, supersededBy, body string) string {
	supersession := ""
	if supersededBy != "" {
		supersession = "superseded_by: " + supersededBy + "\n"
	}
	return "---\ntype: Note\ntitle: History\nstatus: " + status + "\n" +
		supersession + "---\n\n# History\n\n" + body + "\n"
}

func assertSingleHistoryChange(
	t *testing.T,
	result ChangeFeedResult,
	classification ChangeClassification,
) {
	t.Helper()
	if len(result.Changes) != 1 || result.Changes[0].Classification != classification {
		t.Fatalf("changes = %+v, want one %s", result.Changes, classification)
	}
}
