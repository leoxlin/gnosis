---
type: ConceptType
title: Resource
description: A tool, repository, API, service, schema, or dashboard an agent can use or cite.
path: resources
---

# Resource

A **Resource** preserves where something is and how to reach it: repositories, APIs, services, MCP tools, schemas, and dashboards.

By convention, Resource records live at `gnosis://<vault>/resources/`.

## Use this for

- Locations, endpoints, capabilities, and usage constraints of external or internal systems — knowledge that answers "where is something?" or "what can the agent use?".

Do not use it for people or teams (Entity), policies about usage (Policy), or observations of behavior (Event).

## Minimum record

- `kind` frontmatter, `resource` locator, and `# Resource` describing what it provides.
- Optional `# Usage` with exact commands, endpoints, or schemas, and typed `relationships` (`depends_on`, `owned_by`, `deployed_to`).

## Lifecycle

- Identity is the addressed system. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `active` → `deprecated` → `archived`; archived pages are retained, not deleted.
- Update locators and usage in place when they drift, preserving unknown metadata; stale locators are validation-relevant facts, fix them when observed.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Resource
title: <name>
description: <what it provides>
kind: <repository | api | service | tool | dashboard | schema>
resource: <URL or address>
status: <active | deprecated | archived>
---
```
