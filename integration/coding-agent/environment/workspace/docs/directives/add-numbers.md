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

1. Add a failing unittest for integer addition.
2. Implement the smallest function that passes it.
3. Run the complete unittest suite.

# Acceptance criteria

- `calculator.add(2, 3)` returns `5`.
- `python3 -m unittest` passes.
