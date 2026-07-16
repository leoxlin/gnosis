package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	toon "github.com/toon-format/toon-go"
)

func TestTOONOutputIsOrderedAndConforming(t *testing.T) {
	rows := []toon.Object{
		toon.NewObject(
			toon.Field{Key: "uri", Value: "gnosis://local/one.md"},
			toon.Field{Key: "title", Value: "One, quoted"},
			toon.Field{Key: "type", Value: "Reference"},
		),
		toon.NewObject(
			toon.Field{Key: "uri", Value: "gnosis://local/two.md"},
			toon.Field{Key: "title", Value: "Two\nlines"},
			toon.Field{Key: "type", Value: "Reference"},
		),
	}
	value := listOutput("pages", len(rows), rows, "0 pages found", []string{"Run `gnosis get pages <uri>`"})

	var output bytes.Buffer
	if err := writeTOON(&output, value); err != nil {
		t.Fatal(err)
	}
	want := "count: 2\npages[2]{uri,title,type}:\n" +
		"  \"gnosis://local/one.md\",\"One, quoted\",Reference\n" +
		"  \"gnosis://local/two.md\",\"Two\\nlines\",Reference\n" +
		"help[1]: Run `gnosis get pages <uri>`"
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
	if strings.HasSuffix(output.String(), "\n") {
		t.Fatalf("output has trailing newline: %q", output.String())
	}
	for _, line := range strings.Split(output.String(), "\n") {
		if strings.HasSuffix(line, " ") || strings.HasPrefix(line, "\t") {
			t.Fatalf("non-conforming whitespace in %q", line)
		}
	}
	if _, err := toon.Decode(output.Bytes()); err != nil {
		t.Fatalf("decode output: %v", err)
	}
}

func TestTOONOutputMakesEmptyStateDefinitive(t *testing.T) {
	var output bytes.Buffer
	if err := writeTOON(
		&output,
		listOutput("pages", 0, []toon.Object{}, "0 pages found in the current vault", nil),
	); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"count: 0", "pages[0]:", "message: 0 pages found in the current vault"} {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("output = %q, missing %q", output.String(), value)
		}
	}
}

func TestFieldsPreserveRequestedOrder(t *testing.T) {
	selector, err := parseFields("title, uri", []string{"uri"}, []string{"uri", "title", "type"})
	if err != nil {
		t.Fatal(err)
	}
	row := selector.object(func(name string) (any, bool) {
		values := map[string]string{"uri": "u", "title": "t", "type": "x"}
		value, ok := values[name]
		return value, ok
	})
	if got := []string{row.Fields[0].Key, row.Fields[1].Key}; got[0] != "title" || got[1] != "uri" {
		t.Fatalf("fields = %v", got)
	}
}

func TestFieldsRejectUnknownEmptyAndDuplicateNames(t *testing.T) {
	for _, raw := range []string{"unknown", "uri,,title", "uri,uri"} {
		if _, err := parseFields(raw, []string{"uri"}, []string{"uri", "title"}); err == nil {
			t.Fatalf("parseFields(%q) succeeded", raw)
		}
	}
}

func TestTruncateUsesCharactersAndFullEscapeHatch(t *testing.T) {
	content := strings.Repeat("界", detailPreviewLimit+1)
	preview, total, isTruncated := truncate(content, false)
	if total != detailPreviewLimit+1 || !isTruncated {
		t.Fatalf("total = %d, truncated = %t", total, isTruncated)
	}
	if utf8.RuneCountInString(preview) != detailPreviewLimit {
		t.Fatalf("preview characters = %d", utf8.RuneCountInString(preview))
	}
	full, total, isTruncated := truncate(content, true)
	if full != content || total != detailPreviewLimit+1 || isTruncated {
		t.Fatalf("full result length = %d, total = %d, truncated = %t", len(full), total, isTruncated)
	}
}
