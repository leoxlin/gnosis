---
type: Repository Process
title: requesting-code-review
description: Use after an implementation task or major feature and before integration to verify requirements and code quality independently.
---

# requesting-code-review

Code review is an independent gate between implementation and acceptance. The reviewer receives the work product and governing knowledge, not the controller's accumulated conversation or conclusions.

## Use when

- A task completes during subagent-driven development.
- A major feature or complex fix is implemented.
- A branch is approaching merge or pull-request creation.
- A fresh technical perspective would help resolve uncertainty.

## Knowledge inputs

- The governing directive or exact task brief and acceptance criteria.
- Relevant active decisions and repository constraints.
- The base and head revisions, a focused diff package, and implementation report.
- Fresh test evidence for the reviewed state.

## Process

1. Define the review range before dispatch: record the exact base and head revisions and package the commit list, diff stat, and full contextual diff in a file.
2. Give a fresh reviewer a self-contained brief containing paths to the requirements, relevant decisions, implementation report, and diff package. Keep the checkout read-only for the reviewer.
3. Require two explicit judgments:
   - **Requirement compliance:** missing, extra, or misunderstood behavior against the directive.
   - **Implementation quality:** correctness, boundaries, error handling, security, compatibility, tests, maintainability, and production readiness.
4. Require every finding to name evidence and actual severity: Critical, Important, or Minor. The report ends with an unambiguous approval or needs-fixes verdict.
5. Verify the reviewer actually inspected the diff. Resolve Critical and Important findings before proceeding; re-review the corrected range. Evaluate questionable findings through [receiving-code-review](receiving-code-review.md).
6. At the end of a branch, review the complete merge-base-to-head range even if every individual task was reviewed.

## Completion

An independent reviewer has judged both requirement compliance and code quality for the exact range, every blocking finding has been resolved and re-reviewed, and the resulting verdict is supported by file-level evidence.

Adapted from `requesting-code-review`, analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
