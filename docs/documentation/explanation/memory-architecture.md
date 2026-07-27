# Memory architecture

gnosis treats memory as explicit, scoped knowledge rather than hidden agent
state. The same CLI and MCP operations use exactly one backend for each
invocation: a writable vault or Mem0.

## Identity before retrieval

Every operation requires a user ID and an agent ID. The pair defines the
retrieval boundary before ranking occurs. This prevents a semantically similar
memory belonging to another identity from leaking into results.

Vault-backed memory stores one statement per typed page. The page includes the
identity pair, lifecycle state, UTC timestamps, and a hash of the statement.
An exact active duplicate returns `NOOP`, making repeated writes safe without
creating redundant pages.

## Fixed backend selection

With only the identity variables configured, gnosis uses the effective writable
vault. Supplying external memory configuration selects Mem0. A partial
configuration is an error, and an external failure never falls back to a local
write.

This strict choice matters because silent fallback would split one logical
memory across stores and make later recall depend on which backend happened to
be available.

## Memory and durable knowledge

Memory records preserve facts, preferences, and observations relevant to one
scope. They are not a substitute for the rest of the knowledge model. A fact
that becomes generally authoritative belongs in a Concept; a learned lesson
belongs in a Reflection; a governing rule belongs in a Policy.

The vault backend gains reviewability and git history from plain Markdown.
Mem0 provides a dedicated shared service when that operational model is more
appropriate. Both present the same compact add-and-search interface.
