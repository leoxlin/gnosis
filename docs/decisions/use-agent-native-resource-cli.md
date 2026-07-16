---
type: Decision
title: Use an agent-native resource CLI
description: Keep ordinary gnosis commands resource-oriented and render their output as compact TOON.
---

# Decision

Expose the ordinary gnosis command surface as `gnosis [verb] [resource] [name] [flags]`, using focused subcommands only where the operation is itself a stable verb. Render ordinary success, empty-state, help, and error responses as compact TOON.

Keep protocol outputs in their native formats: MCP frames, HTTP JSON, server diagnostics, and shell completion scripts do not pass through the ordinary CLI output boundary.

# Why

A consistent resource grammar and one compact structured-output boundary make commands predictable and self-correcting for agents. The previous mixed grammar and command-specific output formats forced callers to learn unrelated invocation and parsing rules.

# Constraints

- Do not retain compatibility aliases for removed command forms.
- Keep one persistent `--vault` data-context flag and operation-specific flags local to their commands.
- Preserve explicit counts, bounded detail previews with an escape hatch, actionable structured usage errors, and protocol-channel isolation.
