#!/bin/sh

set -eu

TESTS_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
REPO_ROOT=$(CDPATH= cd -- "$TESTS_DIR/.." && pwd)
if [ -z "${GNOSIS_BIN:-}" ]; then
	. "$TESTS_DIR/setup.sh"
fi
VAULT_DIR="$TEST_TMP/local-vault"
mkdir "$VAULT_DIR"

assert_contains() {
	case "$1" in
		*"$2"*) ;;
		*)
			printf 'expected output to contain %s:\n%s\n' "$2" "$1" >&2
			exit 1
			;;
	esac
}

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" create vault --name local --concepts)
assert_contains "$output" "status: created"
test -f "$VAULT_DIR/gnosis.toml"
test -f "$VAULT_DIR/concepts/concept.md"

cat >"$TEST_TMP/okf.md" <<'EOF'
---
type: Concept
title: OKF
description: The Open Knowledge Format used by gnosis vaults.
---

# OKF

OKF is a portable Markdown knowledge format.
EOF

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" apply page \
	"gnosis://local/concepts/okf.md" --filename "$TEST_TMP/okf.md")
assert_contains "$output" "status: applied"
assert_contains "$output" "changed: true"

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" apply page \
	"gnosis://local/concepts/okf.md" --filename "$TEST_TMP/okf.md")
assert_contains "$output" "status: no-op"
assert_contains "$output" "changed: false"

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" get pages \
	"gnosis://local/concepts/okf.md" --full)
assert_contains "$output" "title: OKF"
assert_contains "$output" "OKF is a portable Markdown knowledge format."

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" search knowledge \
	"portable Markdown" --backend lexical)
assert_contains "$output" "gnosis://local/concepts/okf.md"

output=$("$GNOSIS_BIN" --vault "$VAULT_DIR" validate vault)
assert_contains "$output" "status: valid"

printf 'local-vault: ok\n'
