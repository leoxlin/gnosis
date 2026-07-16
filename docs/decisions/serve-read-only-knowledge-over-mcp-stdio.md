---
type: Decision
title: Serve read-only knowledge over MCP stdio
description: Expose gnosis retrieval through a small read-only MCP stdio surface implemented with the official Go SDK.
---

# Decision

Serve gnosis knowledge over MCP stdio through thin read-only tools that list vault knowledge, read exact pages, and search through the existing lexical or semantic retrieval paths. Use the official Go MCP SDK instead of implementing JSON-RPC or transport framing locally.

# Why

Agents need stable exact and retrieval-based access to authored knowledge, while gnosis Markdown remains human-authored durable state. Reusing the existing vault APIs keeps MCP behavior aligned with the CLI and avoids a parallel knowledge model or protocol implementation.

# Constraints

- MCP exposes no mutation, shell, or arbitrary-path tools.
- Stdout contains protocol frames only; diagnostics use stderr.
- Semantic configuration is resolved only when semantic search is requested, so exact and lexical retrieval remain hermetic.
- Additional transports or hosted-service concerns require a separate decision.
