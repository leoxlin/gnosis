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
