# Work with intents

Intents are the vault's governance records: one **Purpose**, durable **Decisions**, and executable **Directives**. All CRUD goes through the `managing-intents` procedure, which defers to each Concept Type's lifecycle.

## Read current intent

    gnosis get pages gnosis://local/purpose.md --full
    gnosis get concepts Decision
    gnosis get directives

`get directives` derives `tasks_done`/`tasks_total` from checkbox steps — progress is computed, never restated.

## Record a decision

1. Query first: `gnosis search knowledge --backend lexical "<the choice>"` — decisions reject duplicate identity.
2. Draft the record (`# Decision`, `# Why`, `# Constraints`; `supersedes` link when replacing).
3. Apply with `gnosis apply page gnosis://local/decisions/<name>.md --filename <file>`.
4. A changed choice is a new Decision whose `supersedes` links the old one — never rewrite history.

## Plan and run a directive

1. Invoke `planning-directives` (`gnosis get procedures gnosis://_/procedures/development/planning-directives.md --full`): refine requirements, draft, review, finalize. It owns the `draft` → `open` transition.
2. A complete directive has Goal, Scope, checkbox tasks under `### Task N:`, evidence-bearing acceptance criteria (`#### Scenario:` blocks for behavior), and `## Added`/`## Modified`/`## Removed` deltas when it changes Purpose or Decision records.
3. Invoke `implementing-directive` for exactly one open directive at a time; it owns `open` → `blocked|done` and flips task checkboxes as work lands.
4. Periodically, when the author says `maintain-intents`, completed directives are archived: their declared deltas merge into the living records, durable choices compact into Decisions, and the directive files are removed.

## Validate

`gnosis validate vault` enforces the directive contract: valid `status`, required sections, checkbox tasks, scenario grammar, and delta headers.
