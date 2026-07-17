# Intent lifecycle

Intent is what governs work: one Purpose, many Decisions, and executable Directives.

## States

- **Purpose** is a singleton holding enduring outcomes and boundaries; it changes rarely and only with author confirmation.
- **Decisions** are append-only in effect: a changed choice creates a new record linked by `supersedes`, preserving the history of why.
- **Directives** move `draft → open → blocked|done`, and each transition is owned by a procedure — planning finalizes, implementing completes, replanning reopens. Status is never assigned from assertion.

## OpenSpec strategies

Directives behave like spec deltas rather than prose tasks:

1. **Delta semantics** — `# Purpose/Decision Changes` declares `## Added`/`## Modified`/`## Removed` effects on living intent records.
2. **Scenarios** — behavior acceptance criteria use `#### Scenario:` blocks with `**WHEN**`/`**THEN**` bullets, making done-ness unambiguous.
3. **Derived progress** — checkbox tasks let `gnosis get directives` compute progress instead of trusting status text.
4. **Strict validation** — the validator enforces the directive contract like code.
5. **Archive as merge** — `maintain-intents` folds a done directive's deltas into the living records, compacts remaining durable choices into Decisions, and removes the directive. The living records are the audit trail.

## Why procedures own transitions

Lifecycle rules live in Concept Type definitions and procedures — data, not code — so any agent that can read the vault can govern work the same way, and the rules evolve as knowledge instead of releases.
