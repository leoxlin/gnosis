---
type: Gnosis Process
title: refining-requirements
description: Use when requested planning must turn a prompt, issue, or bug report into directive-ready requirements.
invocation: model
effects: [read, vault-write, workspace-write, external]
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# refining-requirements

## Use when

- The author requests a plan or directive from any prompt, issue, or bug report.
- The author requests a functional project change, including source-code or non-CI configuration changes.

## Knowledge inputs

- The author's request, repository rules, and current purpose.
- Query-selected decisions, implementation, tests, issues, and authoritative sources.

## Process

1. Query gnosis first. Read only returned records and affected paths. Research only unresolved planning claims; retain each claim's source and version. Ask one author-owned question at a time and recommend a default.
2. For a bug without current root-cause evidence, invoke [systematic-debugging](../debugging/systematic-debugging.md) in diagnosis-only mode. Reuse current debugging evidence instead of reinvoking it. If diagnosis is the requested outcome, return the evidence and stop.
3. Produce one exact requirements packet: outcome, in/out scope, constraints, acceptance evidence, governing knowledge URI/revisions, and resolved choices. Stop with a blocker if any unknown could change the plan; confirm inferred author-owned choices.
4. Pass the packet unchanged to [creating-simple-directive](creating-simple-directive.md) only for one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency. Otherwise invoke [creating-complex-directives](creating-complex-directives.md).

## Completion

Diagnosis is complete, or evidence-backed requirements are routed by exact process URI.
