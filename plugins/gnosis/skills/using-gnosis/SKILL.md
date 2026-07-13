---
name: using-gnosis
description: Select and apply canonical Procedure records for work in a gnosis vault or repository.
---

# Using gnosis

Use this skill as the procedure gateway. Procedure records are the source of workflow instructions.

1. Resolve the current vault and repository, then read their agent instructions.
2. Run `gnosis procedure discovery` exactly once for the task. Do not delegate discovery or selection to a sub-agent.
3. From that output, directly select the smallest applicable dependency-ordered set of `Procedure` records using `description`. The command lists only model-invocable procedures; do not select another concept type.
4. Verify that every selection matches the author's request and current instructions. Proceed without asking the user to confirm the selected procedure series.
5. Read each selected procedure in dependency order with `gnosis read 'gnosis://....'`, using the exact URI returned by discovery.
6. Follow a single-step procedure's `Knowledge inputs`, `Process`, and `Completion` sections, or follow a multi-step procedure's numbered `STEP` sections according to their routing and branch instructions. Use gnosis page reads, knowledge queries, and link tracing for referenced records. Select another procedure only when the active contract requires it, and bind that selection to its exact URI.
7. Stop only when every selected procedure reaches its completion gate. Preserve current vault configuration and support repository completion claims with fresh evidence.
