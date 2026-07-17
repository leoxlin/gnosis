# Procedures

Procedures are executable contracts an agent loads with `gnosis get procedures <uri> --full`. Discovery: `gnosis get procedures --tags gnosis,<family>`.

## Vault family

| Procedure | Use when |
|---|---|
| `query-vault` | Answering a question from recorded knowledge (tiered cost ladder). |
| `ingest-knowledge` | Supplied evidence should create or update concept pages. |
| `remember` | An episode or statement should become durable scoped memories. |
| `recall` | Answering from scoped Memory records. |
| `link-pages` | Pages mention known knowledge without linking it. |
| `maintain-vault` | Auditing or repairing vault integrity (consolidation pass). |
| `create-concept-type` | A vault needs a new or refined ontological category. |
| `refining-procedure` | An existing procedure must be rewritten for reliable execution (explicit request). |

## Development family

| Procedure | Use when |
|---|---|
| `planning-directives` | Turning a request into validated directive handoffs. |
| `implementing-directive` | Exactly one open directive must be implemented and verified. |
| `managing-intents` | A Purpose, Decision, or Directive record must be created, read, updated, or deleted. |
| `maintain-intents` | Explicitly requested archival of completed directives (merge deltas, compact decisions). |
| `debugging-methodically` | Diagnosing any bug or unexpected behavior before fixing. |

## Rules

- Procedures are selected and executed by the controlling agent; never spawn a selector subagent.
- Follow the `Inputs`/`Process`/`Completion` sections, or numbered `STEP` sections for multi-step procedures.
- Every procedure run ends with `gnosis validate vault` when it writes.
