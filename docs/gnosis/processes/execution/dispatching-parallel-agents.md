---
type: GnosisProcess
title: dispatching-parallel-agents
description: Use when two or more problem domains can be investigated or changed without shared state, overlapping files, or sequential dependencies.
tags: [gnosis-execution]
invocation: model
effects: [vault-write, workspace-write]
use_when:
  - Failures belong to distinct tests or subsystems with unrelated root causes.
  - Several investigations can reach conclusions without consuming one another's output.
  - Agents have separate workspaces or read-only scopes that prevent interference.
  - Related failures, exploratory diagnosis, shared-state changes, and sequential dependencies remain under one coordinating process.
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# dispatching-parallel-agents

Parallel dispatch assigns one independent problem domain to each agent with isolated, deliberately constructed context. Concurrency is earned by independence, not merely by task count.

## Knowledge inputs

- The governing directive or bounded problem statement.
- Relevant active decisions and Gnosis Process pages for each domain.
- Current failures, implementation boundaries, ownership of files and workspaces, and integration tests.
- Dependencies between domains, including any shared mutable resource.

## Process

1. Partition work by causal and ownership boundaries. For each candidate pair, verify that neither needs the other's result and that their edits or resources cannot conflict.
2. Create one focused, self-contained brief per domain. Include the exact goal, scope, evidence, constraints, relevant knowledge paths, allowed files or resources, required verification, report contract, and every governing process's exact URI for invocation with `gnosis process invoke --id '<process URI>' --pretty`.
3. Dispatch all independent briefs concurrently. Use separate workspaces for modifying agents; read-only investigations may share a checkout.
4. Require each agent to report root cause or implementation, files changed, commands and results, unresolved concerns, and the evidence supporting its conclusion.
5. Review every returned artifact and diff independently. Check for overlap or incompatible assumptions before integrating.
6. Run the combined verification suite and any cross-domain checks after integration. A set of individually passing changes is not evidence that the combined state passes.

## Completion

Every domain has an evidence-backed result, concurrent work has no unresolved overlap, and fresh combined verification confirms the integrated state. The controller, not an individual parallel agent, updates the governing directive.
