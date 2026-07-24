#!/bin/sh

set -eu

TESTS_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$TESTS_DIR/.." && pwd)
. "$TESTS_DIR/setup.sh"
export TESTS_DIR REPO_ROOT TEST_TMP GNOSIS_BIN

for test in "$TESTS_DIR"/*/run.sh; do
	"$test"
done
