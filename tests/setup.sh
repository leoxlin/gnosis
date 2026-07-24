#!/bin/sh

TEST_TMP=$(mktemp -d "${TMPDIR:-/tmp}/gnosis-integration.XXXXXX")
GNOSIS_BIN="$TEST_TMP/gnosis"

cleanup() {
	rm -rf -- "$TEST_TMP"
}
trap cleanup 0
trap 'exit 1' 1 2 15

(cd "$REPO_ROOT" && go build -o "$GNOSIS_BIN" ./cmd/gnosis)
