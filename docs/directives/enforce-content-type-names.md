---
type: Directive
title: Enforce Content Type Names
description: Require TypeName-form names for every Concept Type definition and migrate built-in types.
status: done
---

# Goal

Make every Concept Type definition use the `TypeName` (UpperCamelCase) convention, and reject definitions that do not.

# Scope

Change `internal/vault/validate.go` and its tests to validate the `title` of documents whose `type` is `Concept Type`. Change the four built-in Concept Type titles and every built-in document `type` value that refers to them to their UpperCamelCase form. Update exact-type process detection and affected Go test fixtures and expectations. Do not modify README.md or change non-Concept-Type human-facing titles.

# Implementation plan

1. Add focused validation tests that show a Concept Type title containing a space is accepted before implementation and that a TypeName title remains valid.
2. Add the smallest validation helper and validation error to reject Concept Type titles outside UpperCamelCase.
3. Replace built-in Concept Type names and the type references that resolve to them: `Purpose`→`Purpose`, `Procedure`→`Procedure`, `Directive`→`Directive`, and `Decision`→`Decision`; update the corresponding runtime constant and tests.
4. Run focused validation tests, the vault package suite, and the full Go suite; run `gnosis validate --vault .`; inspect the diff.

# Acceptance criteria

- A Concept Type document titled `Procedure` produces a validation error identifying the title and TypeName convention.
- A Concept Type document titled `Procedure` validates successfully.
- All built-in Concept Type definitions and their documents use exact UpperCamelCase type names.
- Process discovery and validation continue to recognize `Procedure` records.
- Focused tests, `go test ./internal/vault`, `go test ./...`, and `gnosis validate --vault .` complete successfully.
