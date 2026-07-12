---
name: using-gnosis
description: Select and apply canonical Procedure records for work in a gnosis vault or repository.
---

# Using gnosis

Use this skill as the procedure gateway. Procedure records are the source of workflow instructions.

1. Resolve the current vault and repository, then read their agent instructions.
2. For every task, dispatch a fresh read-only procedure-selector sub-agent. Give it this explicit prompt, replacing bracketed values with the current values:

   ```text
   You are the read-only procedure selector for the controlling agent. Do not edit files, run tests, or execute procedure instructions.

   Author request (exact): [author request]
   Resolved vault and repository: [vault and repository]
   Applicable agent instructions: [agent instructions]

   - Run `gnosis procedure discovery` exactly once.
   - From that output, select the smallest applicable dependency-ordered `Procedure` using `description` and `use_when`.
   - The command lists only model-invocable procedures; do not select another concept type.

   Return the heading `FULL ORDERED PROCEDURE LIST TO EXECUTE`. Under it, return every selected procedure in dependency order, with each command on its own line and exactly formatted as `gnosis read 'gnosis://....'`. After each command, include concise selection rationale. If no procedure applies, say so under that heading.
   ```

3. When the selector finishes, print its selected procedure commands in dependency order. Before execution, confirm that every selection matches the author's request and current instructions. If the current invocation did not explicitly enable auto mode, ask the user to confirm the selected procedure series and wait for that confirmation before reading or executing it.
4. Read each confirmed selected procedure by running its exact returned `gnosis read 'gnosis://....'` command.
5. Follow the returned `knowledge_inputs`, `process`, and `completion` sections under current instructions. Use gnosis page reads, knowledge queries, and link tracing for referenced records. Select another procedure only when the active contract requires it, and bind that selection to its exact URI.
6. Stop only when every selected procedure reaches its completion gate. Preserve current vault configuration and support repository completion claims with fresh evidence.
