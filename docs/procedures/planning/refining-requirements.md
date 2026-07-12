---
type: Procedure
title: refining-requirements
description: Use when requested planning must turn a prompt, issue, or bug report into directive-ready requirements.
tags: [gnosis-planning]
invocation: model
---

# refining-requirements

## Knowledge inputs

- The author's request, repository rules, and current purpose.
- Query-selected decisions, implementation, tests, issues, and authoritative sources.

## Process

1. Query gnosis first. Read only returned records and affected paths. Research only unresolved planning claims; retain each claim's source and version. Ask one author-owned question at a time and recommend a default.
2. For a bug without current root-cause evidence, invoke [systematic-debugging](../debugging/systematic-debugging.md) in diagnosis-only mode. Reuse current debugging evidence instead of reinvoking it. If diagnosis is the requested outcome, return the evidence and stop.
3. Produce one exact requirements packet: outcome, in/out scope, constraints, acceptance evidence, governing knowledge URI/revisions, and resolved choices. Stop with a blocker if any unknown could change the plan; confirm inferred author-owned choices.
4. Pass the packet unchanged to [creating-directives](creating-directives.md), identifying whether it is one bounded delivery with no material architecture, purpose, decision, research, or cross-directive dependency, or complex work.

## Completion

Diagnosis is complete, or evidence-backed requirements are routed by exact process URI.
