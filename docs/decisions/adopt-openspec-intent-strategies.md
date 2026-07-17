---
type: Decision
title: Adopt OpenSpec strategies for intent records
description: Directives gain delta semantics, scenario-based acceptance criteria, computed task progress, strict structural validation, and archive-as-merge completion.
---

# Decision

Adopt five strategies from OpenSpec (Fission-AI/OpenSpec) into the gnosis intent system:

1. **Delta semantics.** A Directive's `# Purpose/Decision Changes` section declares its effect on living intent records as explicit deltas: `## Added`, `## Modified`, and `## Removed` subsections naming the exact target records and their effects. Finishing a directive merges those deltas into the living records (OpenSpec's archive step) before the directive becomes historical.
2. **Scenario-based acceptance criteria.** Acceptance criteria may use `#### Scenario: <name>` blocks with bold `**WHEN**` / `**THEN**` (and optional `**GIVEN**` / `**AND**`) bullets. When any scenario is present, every scenario must follow the grammar; malformed scenarios are validation errors, not conventions.
3. **Computed task progress.** Implementation tasks keep the `- [ ]` checkbox format, and the CLI derives per-directive progress (done/total) from the checkboxes instead of agents restating status.
4. **Strict structural validation.** The vault validator enforces the Directive contract — valid `status`, required `# Goal` / `# Scope` / `# Acceptance criteria` sections, checkbox tasks inside `# Implementation plan`, and scenario grammar — the same way it already enforces the Procedure contract.
5. **Archive as merge.** `maintain-intents` folds a `done` directive's declared deltas into the living Purpose/Decision records it named, then removes the directive, leaving the merged records as the audit trail.

# Why

OpenSpec's state-versus-diff separation makes each directive a precise, machine-checkable patch against living intent records instead of prose that drifts from what it changed. Scenario grammar turns acceptance criteria into unambiguous done-ness evidence. Computed progress and strict validation make directive state derivable rather than agent-remembered, which is the difference between a reliable handoff and a hopeful one.

Scenarios stay optional (progressive rigor) because many gnosis directives are exact-validation tasks where a command-and-expected-result criterion is already unambiguous; forcing scenarios there would be ceremony.

# Constraints

- The Directive schema stays plain Markdown with YAML frontmatter; deltas and scenarios are heading and bullet conventions, not new file formats.
- Delta declarations name exact record URIs; `## Modified` requires the full replacement text or exact section edits in the implementing procedure's contract.
- Checkbox progress is derived state; no status field is added for it.
- Validation treats structural violations as errors and keeps style guidance as warnings, matching the existing validator split.
- `maintain-intents` remains explicitly triggered; archival never runs implicitly.
