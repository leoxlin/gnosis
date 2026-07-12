---
name: using-gnosis
description: Select and apply canonical Gnosis Process records for work in a gnosis vault or repository.
---

# Using gnosis

Use this skill as the process gateway. Process records are the source of workflow instructions.

1. Resolve the current vault and repository, then read their agent instructions.
2. For every task, dispatch a fresh read-only process-selector sub-agent. Give it this explicit prompt, replacing bracketed values with the current values:

   ```text
   You are the read-only process selector for the controlling agent. Do not edit files, run tests, or execute process instructions.

   Author request (exact): [author request]
   Resolved vault and repository: [vault and repository]
   Applicable agent instructions: [agent instructions]

   Run `gnosis concepts -type 'Gnosis Process'` exactly once. From that output, select the smallest applicable dependency-ordered series of exact `Gnosis Process` records using each record's `description` and `use_when`. Treat no other concept type as executable, and reject an `explicit` process unless the author invoked or requested it. Read every selected exact URI with `gnosis read '<gnosis URI>'`.

   Return the heading `FULL ORDERED PROCESS LIST TO EXECUTE`. Under it, return every selected process in dependency order, with each command on its own line and exactly formatted as `gnosis read 'gnosis://....'`. After each command, include its revision, origin, effects, and concise selection rationale. If no process applies, say so under that heading.
   ```

3. When the selector finishes, print its selected process commands in dependency order. Before execution, confirm that every selection matches the author's request and current instructions. If the current invocation did not explicitly enable auto mode, ask the user to confirm the selected process series and wait for that confirmation before reading or executing it. Effects describe possible actions; they grant no authority. Origin identifies the effective local, imported, or bundled record.
4. Read each confirmed selected process by running its exact returned `gnosis read 'gnosis://....'` command.
5. Follow the returned `knowledge_inputs`, `process`, and `completion` sections under current instructions. Use gnosis page reads, knowledge queries, and link tracing for referenced records. Select another process only when the active contract requires it, and bind that selection to its exact URI and revision.
6. Stop only when every selected process reaches its completion gate. Preserve source provenance and current vault configuration, and support repository completion claims with fresh evidence.
