---
type: Directive
title: Fix integration verifier and remove dead API
description: Align the coding-agent integration verifier with the axi CLI, update its fixture to the directive contract, and delete the unused ListConcepts and Read APIs.
status: done
---

# Goal

Make the Harbor coding-agent integration verifier consistent with the current axi CLI and the directive contract, and completely remove the obsolete `vault.ListConcepts` and `vault.Read` APIs per the repository rule to prefer complete removal of obsolete behavior.

# Scope

- `integration/coding-agent/tests/test.sh`: expect the axi call forms the shipped skills actually instruct.
- `integration/coding-agent/environment/workspace/docs/directives/add-numbers.md`: satisfy the directive contract (checkbox tasks).
- `integration/coding-agent/environment/workspace/gnosis.toml`, `integration/coding-agent.sh`, `integration/coding-agent/environment/Dockerfile`: remove the obsolete `.gnosis-core` import — the embedded bundle already supplies the `core` vault, and the import hard-fails vault loading because `.gnosis-core` carries no `gnosis.toml` (blocking finding proven during execution with the old and new binaries).
- `internal/vault/concepts.go`: delete `ListConcepts` and `writeConceptTypePreviews`; `internal/vault/concepts_test.go`: delete their test.
- `internal/vault/search.go`: delete `Read`; `internal/vault/search_test.go`: delete its only call.
- No replacement APIs: `vault.Concepts` and `vault.ReadPage` already cover every real caller.

# Global constraints

- Complete removal, no compatibility shims (repository rule).
- Verify with `grep` that no references remain anywhere (code, tests, docs, skills).
- The integration environment cannot run here (Harbor + Docker); verification is static consistency between the skill text, the shim, and the verifier, plus the full Go gate.

# Dependencies

None.

# Implementation plan

### Task 1: Align the integration verifier and fixture with the axi CLI

**Load:** `integration/coding-agent/tests/test.sh`, `integration/coding-agent/environment/gnosis` (the arg-logging shim), `plugins/gnosis/skills/using-gnosis-for-development/SKILL.md`, `integration/coding-agent/environment/workspace/docs/directives/add-numbers.md`.
**Files:** modify `integration/coding-agent/tests/test.sh`, modify `integration/coding-agent/environment/workspace/docs/directives/add-numbers.md`.
**Interfaces:** consumes the skill's exact commands; produces matching verifier expectations.

- [x] In `integration/coding-agent/tests/test.sh`, replace the line `	&& [ "$(grep -Fxc 'procedure discovery --tags gnosis,development' .gnosis-calls)" -eq 1 ] \` with `	&& [ "$(grep -Fxc 'get procedures --tags gnosis,development' .gnosis-calls)" -eq 1 ] \` and replace the line `	&& grep -Fxq 'read gnosis://core/procedures/development/implementing-directive.md' .gnosis-calls` with `	&& grep -Fxq 'get procedures gnosis://core/procedures/development/implementing-directive.md --full' .gnosis-calls`. Rationale: the shim logs only `"$*"`, and the shipped skill instructs exactly `gnosis get procedures --tags gnosis,development` once and `gnosis get procedures '<URI>' --full` per contract read; procedures resolve from the `core` vault in the fixture.
- [x] Replace `integration/coding-agent/environment/workspace/docs/directives/add-numbers.md` with exactly:

````markdown
---
type: Directive
title: Add numbers
description: Add integer addition to the calculator module.
status: open
---

# Goal

Allow callers to add two integers.

# Scope

Add `calculator.add(left, right)` and its regression test. Do not change other behavior.

# Implementation plan

### Task 1: Add integer addition

**Load:** `calculator.py`, `test_calculator.py`.
**Files:** modify `calculator.py`, modify `test_calculator.py`.
**Interfaces:** produces `add(left: int, right: int) -> int`.

- [ ] Add a failing unittest for integer addition; run `python3 -m unittest`; expect failure.
- [ ] Implement the smallest function that passes it; run `python3 -m unittest`; expect pass.

# Acceptance criteria

- `calculator.add(2, 3)` returns `5` — run `python3 -c 'import calculator; assert calculator.add(2, 3) == 5'`; expect exit 0.
- `python3 -m unittest` passes.
- Work in this checkout; do not create a worktree.
- Use Python and its standard library only.
- Keep the current branch when the directive is complete; delivery is outside scope.
````

- [x] Static check: `grep -rn 'procedure discovery\|gnosis read ' integration/ plugins/`; expect no matches.
- [x] Remove the obsolete `.gnosis-core` import (root cause proven during execution: the import target carries no `gnosis.toml`, so every vault command hard-fails): delete the `[[vaults]]` block from `integration/coding-agent/environment/workspace/gnosis.toml`, delete `cp -R "$repo/docs" "$task/environment/.gnosis-core"` from `integration/coding-agent.sh`, and delete `COPY .gnosis-core .gnosis-core` from `integration/coding-agent/environment/Dockerfile`. Procedures still resolve as `gnosis://core/...` from the embedded bundle.
- [x] Host-side fixture check: `tmp=$(mktemp -d) && cp -r integration/coding-agent/environment/workspace/. "$tmp/" && ./dist/gnosis --vault "$tmp" validate vault`; expect `status: valid`, `warnings: 0`; `./dist/gnosis --vault "$tmp" get procedures --tags gnosis,development`; expect 5 `gnosis://core/procedures/development/` URIs.
- [x] Commit: `fix: align integration verifier with axi cli`.

### Task 2: Remove ListConcepts and Read

**Load:** `internal/vault/concepts.go:87-160` (`ListConcepts`, `writeConceptTypePreviews`), `internal/vault/concepts_test.go:51-90`, `internal/vault/search.go:216-245` (`Read`), `internal/vault/search_test.go:169-178`.
**Files:** modify `internal/vault/concepts.go`, `internal/vault/concepts_test.go`, `internal/vault/search.go`, `internal/vault/search_test.go`.
**Interfaces:** removes `ListConcepts(root, conceptType string, output io.Writer) error` and `Read(root, conceptType, title string) ([]byte, error)`.

- [x] Red: run `go test ./internal/vault/ -run 'TestListConcepts|TestSearchSourceIncludesBundled'`; expect ok now (baseline), then delete the code and watch the compiler flag every remaining reference — the reference set must be exactly the four files above.
- [x] Delete `ListConcepts` and `writeConceptTypePreviews` from `internal/vault/concepts.go`; drop the now-unused `io` import if nothing else uses it (check with `go build ./internal/vault/`).
- [x] Delete `TestListConceptsWritesTypePreviewsAndTypedConcepts` from `internal/vault/concepts_test.go`; drop unused imports/identifiers the compiler flags.
- [x] Delete `Read` from `internal/vault/search.go`.
- [x] In `internal/vault/search_test.go`, delete the block `data, err := Read(root, "Procedure", "query-vault")` through its `t.Fatalf("read = %q", data)` assertion; keep every other assertion.
- [x] Run `go build ./... && go test ./...`; expect all ok.
- [x] Run `grep -rn 'ListConcepts\|vault\.Read(\|\bRead(root' --include='*.go' .`; expect no matches.
- [x] Commit: `refactor: remove dead listconcepts and read apis`.

# Acceptance criteria

- The verifier matches the shipped skill — inspect `integration/coding-agent/tests/test.sh` beside `plugins/gnosis/skills/using-gnosis-for-development/SKILL.md`; expect the logged forms `get procedures --tags gnosis,development` (exactly once) and `get procedures gnosis://core/procedures/development/implementing-directive.md --full`.
- The fixture satisfies the directive contract — inspect the fixture `add-numbers.md`; expect `### Task 1:` with checkbox steps; run `tmp=$(mktemp -d) && cp -r integration/coding-agent/environment/workspace/. "$tmp/" && ./dist/gnosis --vault "$tmp" validate vault`; expect `status: valid`, `warnings: 0`, with core procedures served from the embedded bundle.
- Dead API is gone — run `grep -rn 'ListConcepts\|writeConceptTypePreviews' --include='*.go' .`; expect no matches; run `go doc gnosis/internal/vault Read`; expect no symbol.
- No regressions — run `mise run checks`; expect all green.
