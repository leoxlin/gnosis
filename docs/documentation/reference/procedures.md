# Procedures

A `Procedure` is an invocable Markdown execution contract. List procedures with
`gnosis get procedures`, filter them with `--tags`, and load one complete
contract with:

```bash
gnosis get procedures <gnosis-uri> --full
```

## Bundled procedures

| Procedure | Select when |
|---|---|
| `create-concept-type` | A vault needs a new or refined category |
| `ingest-knowledge` | Supplied evidence should create or update pages |
| `link-pages` | Existing pages mention known records without linking them |
| `maintain-vault` | Vault integrity needs auditing or repair |
| `query-vault` | A question must be answered from recorded knowledge |
| `recall` | An answer must come from scoped Memory records |
| `refining-procedure` | The author explicitly asks to refine an execution contract |
| `remember` | An episode or statement should become durable Memory records |

The bundled contracts use canonical URIs under `gnosis://core/procedures/`.
The portable `_` authority can select the highest-precedence version at the
same path.

## Contract rules

- The controlling agent discovers and selects the smallest applicable set.
- A selected contract is read in full before execution.
- Single-step contracts define `Inputs`, `Process`, and `Completion`.
- Multi-step contracts define numbered `STEP` sections and routing.
- A procedure that writes completes only after its stated validation and
  reporting gates pass.

Repository development planning is handled by OpenSpec, not by a bundled
development-procedure family.
