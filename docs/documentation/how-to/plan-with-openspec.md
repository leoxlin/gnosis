# Plan repository work with OpenSpec

Use OpenSpec for gnosis requirements, design decisions, implementation tasks,
and completed change history. The repository's `openspec/config.yaml`
registers the external `trium` store. Run commands from the gnosis repository
root so OpenSpec resolves that store.

Create a kebab-case change and inspect its required artifacts:

```bash
openspec new change <change-name>
openspec status --change <change-name>
```

Complete the generated proposal, delta specifications, design, and task list in
the store before implementation. Keep task checkboxes aligned with delivered
work.

Inspect current changes and specifications:

```bash
openspec list
openspec list --specs
openspec show <change-or-spec>
```

Validate all artifacts before finishing:

```bash
openspec validate --all --strict --no-interactive
```

After implementation and project checks pass, sync the change's delta
specifications into the main specifications and archive the change with the
OpenSpec workflow. Do not edit OpenSpec artifacts through `gnosis apply page`;
gnosis projects them for reading and search but leaves authoring to OpenSpec.
