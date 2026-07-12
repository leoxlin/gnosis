---
type: Procedure
title: test-driven-development
description: Use when implementing a feature, bug fix, refactor, or behavior change before changing production code.
tags: [gnosis-execution]
invocation: model
---

# test-driven-development

Test-driven development proceeds red, green, then refactor. Seeing the intended test fail for the intended reason is the evidence that the test can detect missing behavior.

## Knowledge inputs

- The directive task and acceptance criteria for the behavior.
- Relevant decisions, test conventions, and supported-platform constraints.
- Existing public interfaces, implementation boundaries, tests, and test commands.

## Process

1. **Red:** Write one minimal test that expresses the next required behavior through the real interface. Use mocks only where a real dependency cannot reasonably be exercised.
2. Run the focused test and observe a behavioral failure caused by the missing behavior. A passing test or setup error is not red; correct the test until it fails for the intended reason.
3. **Green:** Write only the production code needed to satisfy that test. Keep unrelated refactors and future options out of this step.
4. Run the focused test, then the relevant surrounding tests. Fix production code until output is clean; do not weaken the requirement to obtain green.
5. **Refactor:** Improve names, boundaries, and duplication only while tests remain green. Add no new behavior during refactoring.
6. Repeat for the next behavior or edge case.

A production change written for the task before its failing test is reverted and reimplemented from the red state; retaining it as a template turns the cycle into tests-after.

## Completion

Every changed behavior has a test that was observed failing for the expected reason before implementation, minimal code made it pass, refactoring preserved green, edge and error cases required by the directive are covered, and the relevant suite has clean output.
