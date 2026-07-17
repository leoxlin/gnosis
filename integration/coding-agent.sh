#!/bin/sh

set -eu

repo=$(CDPATH='' cd -- "$(dirname "$0")/.." && pwd)
harbor=${HARBOR_BIN:-harbor}
gnosis=${GNOSIS_BIN:-"$repo/dist/gnosis"}
model=${HARBOR_MODEL:-openai/gpt-5.4}

command -v "$harbor" >/dev/null 2>&1 || {
	printf 'Harbor not found: %s (run uv tool install harbor)\n' "$harbor" >&2
	exit 1
}
[ -x "$gnosis" ] || {
	printf 'gnosis binary not found: %s (run mise run build)\n' "$gnosis" >&2
	exit 1
}

tmp=$(mktemp -d "${TMPDIR:-/tmp}/gnosis-harbor-integration.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
task=$tmp/task
cp -R "$repo/integration/coding-agent/." "$task"
cp "$gnosis" "$task/environment/gnosis-real"

if [ -z "${OPENAI_API_KEY:-}" ] && [ -z "${CODEX_AUTH_JSON_PATH:-}" ]; then
	export CODEX_FORCE_AUTH_JSON="${CODEX_FORCE_AUTH_JSON:-1}"
fi

"$harbor" run \
	--path "$task" \
	--agent codex \
	--model "$model" \
	--skill "$repo/plugins/gnosis/skills/using-gnosis-for-development" \
	--job-name coding-agent-integration \
	--jobs-dir "$tmp/jobs"

python3 -c 'import json, sys; result = json.load(open(sys.argv[1])); trials = result["trial_results"]; assert len(trials) == 1; trial = trials[0]; assert trial["exception_info"] is None; assert trial["verifier_result"]["rewards"]["reward"] == 1' \
	"$tmp/jobs/coding-agent-integration/result.json"
