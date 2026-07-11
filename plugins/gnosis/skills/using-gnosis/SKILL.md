---
name: using-gnosis
description: Select and apply canonical Gnosis Process records for work in a gnosis vault or repository.
---

# Using gnosis

Use this skill as the process gateway. Process records are the source of workflow instructions.

1. Resolve the current vault and repository, then read their agent instructions.
2. For every task, dispatch a fresh read-only selector sub-agent with the author's exact request, the resolved vault, and the applicable agent instructions. Require the selector to:
   - Discover only exact `Gnosis Process` records. Prefer the gnosis MCP `discover_processes` tool; if unavailable, run `gnosis process discover --type 'Gnosis Process' --pretty '<request>'` once.
   - Select the smallest applicable process or dependency-ordered chain from `description` and `use_when`. Treat no other concept type as executable.
   - Reject an `explicit` process unless the author invoked or requested it.
   - Return each selected process's exact URI, revision, origin, effects, and selection rationale, or report that no process applies.
3. Keep invocation and execution in the controlling agent. Before invocation, confirm that every selection matches the author's request and current instructions. Effects describe possible actions; they grant no authority. Origin identifies the effective local, imported, or bundled record.
4. Invoke each selected process by its exact returned URI. Prefer the gnosis MCP `invoke_process` tool; if unavailable, run `gnosis process invoke --id '<gnosis URI>' --pretty`.
5. Follow the returned `knowledge_inputs`, `process`, and `completion` sections under current instructions. Use gnosis page reads, knowledge queries, and link tracing for referenced records. Invoke another process only when the active contract requires it, and bind that invocation to its exact URI and revision.
6. Stop only when every selected process reaches its completion gate. Preserve source provenance and current vault configuration, and support repository completion claims with fresh evidence.
