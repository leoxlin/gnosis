---
type: Repository Decision
title: Make vault processes knowledge
description: Store vault workflows as typed vault knowledge and use packaged skills as runtime adapters.
---

# Decision

Represent each reusable `gnosis-vault` workflow as a [Vault Process](../../concepts/vault-process.md) record under `docs/vault/processes/`. Make these records the source of truth. Keep packaged vault skills as small runtime adapters that point to and execute their corresponding canonical process.

# Why

Vault workflows need to remain readable, linkable, and adaptable by the vaults they govern. Keeping their complete instructions only in plugin packaging would make runtime artifacts the source of truth and invite them to drift from durable knowledge. A single process record per capability makes its selection conditions, knowledge inputs, ordered behavior, and completion evidence visible without abandoning runtime skill discovery.

Rejected alternatives:

- **Keep complete workflows only in packaged skills** — rejected because a vault cannot inspect or evolve its own governing processes as knowledge.
- **Add one generic vault process for every capability** — rejected because ingestion, ontology, maintenance, and query workflows have different inputs, mutation rules, and completion states.
- **Remove packaged skills entirely** — rejected because runtime discovery and invocation still require a portable adapter.

# Constraints

- Each core vault capability has exactly one `Vault Process` record with `Use when`, `Knowledge inputs`, `Process`, and `Completion` sections.
- Each packaged vault skill links to its matching process and does not duplicate its workflow.
- Processes retain current behavior for configured indexes and logs, validation, source traceability, author-owned semantic conflicts, and read-only queries.

# Related decisions

- [Make repository processes knowledge](make-repository-processes-knowledge.md)
- [Bootstrap `gnosis` knowledge first on OKF](bootstrap-knowledge-first.md)
