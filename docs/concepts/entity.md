---
type: ConceptType
title: Entity
description: A person, team, agent, or organization that knowledge is about.
path: entities
---

# Entity

An **Entity** preserves who is involved: people, teams, agents, and organizations, with their roles and relationships.

By convention, Entity records live at `gnosis://<vault>/entities/`.

## Use this for

- Ownership, authorship, trust relationships, team structure, and agent identity — knowledge that answers "who is involved?".

Do not use it for tools or services (Resource), preferences (Memory with `scope: user`), or events (Event).

## Minimum record

- `kind` frontmatter and `# Entity` describing who this is.
- Optional `# Roles` and typed `relationships` (`owns`, `maintains`, `member_of`, `trusts`).

## Lifecycle

- Identity is the real-world entity. Query for an existing page before creating one; reject duplicate identity.
- `status` is `active` while the entity is relevant and `archived` when it leaves scope; archived pages are retained, not deleted.
- Update roles and relationships in place as they change, preserving unknown metadata.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Entity
title: <name>
description: <who this is>
kind: <person | team | agent | organization>
status: <active | archived>
source: <optional origin of the claim>
---
```
