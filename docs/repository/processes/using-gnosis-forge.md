---
type: Repository Process
title: using-gnosis-forge
description: Use when an author explicitly asks gnosis to apply repository knowledge and processes to repository work.
---

# using-gnosis-forge

`using-gnosis-forge` is the explicit entry point for knowledge-driven repository work. It selects relevant Repository Process records, grounds them in durable repository knowledge, and then checks their instructions against implementation truth.

## Use when

- The author manually invokes `using-gnosis-forge` for a repository task.
- A resumed manual run needs to reconstruct its governing purpose, decisions, process, or directive.

Do not invoke this process implicitly. A dispatched subagent follows the process and knowledge named in its brief instead of restarting process selection.

## Knowledge inputs

- `gnosis.toml` and repository agent instructions.
- The [repository purpose](../purpose.md).
- Active decisions, open directives, concepts, and Repository Process pages relevant to the request.
- Current implementation, tests, and verification output.
- Path-scoped git history only when the preceding sources do not explain a choice.

## Process

1. Resolve the vault and read repository instructions.
2. Read repository purpose, then classify the request and retrieve only the active decisions, directives, concepts, and processes that govern it.
3. Inspect current implementation and tests for behavioral truth. Use scoped history only to close a specific rationale gap.
4. State the selected process or process chain, then follow it in dependency order. Process instructions govern the method; direct author instructions and repository rules retain precedence.
5. When a process produces durable knowledge, follow its concept definition and record shape rather than inventing one.
6. Keep knowledge and implementation aligned and run each selected process's completion gate.

## Completion

The selected process chain has reached its observable terminal state, every required durable record is valid, and no completion claim exceeds fresh evidence.

Adapted from [`using-superpowers`](https://github.com/obra/Superpowers/blob/d884ae04edebef577e82ff7c4e143debd0bbec99/skills/using-superpowers/SKILL.md), analyzed in [Superpowers (obra)](../../references/obra-superpowers.md).
