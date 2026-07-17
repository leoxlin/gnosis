---
type: Decision
title: Adopt the knowledge-type ontology
description: Cover every knowledge use case in the knowledge-research taxonomy with seven new bundled concept types and a shared provenance metadata standard.
---

# Decision

Adopt seven new bundled Concept Type records — **Concept**, **Entity**, **Resource**, **Event**, **Memory**, **Reflection**, and **Policy** — and one shared provenance and epistemic metadata standard for content pages.

Map the knowledge-research taxonomy (knowledge-research.pdf, repository root) onto gnosis as follows:

- Semantic / factual knowledge → **Concept** pages.
- Episodic / experiential and perceptual / observed knowledge → **Event** pages.
- Procedural knowledge → existing **Procedure** records.
- Causal knowledge → typed `relationships` edges (`causes`, `depends_on`) between pages, not a new type.
- Conditional / situational and normative / policy knowledge → **Policy** pages.
- Social / relational knowledge → **Entity** pages (people, teams, agents, organizations).
- Resource and tool knowledge → **Resource** pages (repositories, APIs, services, dashboards, MCP tools).
- Preference / persona knowledge → **Memory** pages with `scope: user`.
- Metacognitive knowledge → the epistemic metadata standard (`status`, `confidence`, unknowns recorded in page bodies).
- Reflective / learned knowledge → **Reflection** pages.
- Working and short-term knowledge → the agent's context window and scratch state, never the vault.

The shared metadata standard adds optional frontmatter fields for content types: `status` (lifecycle state defined per type), `confidence` (0.0–1.0), `source` (where the claim came from), `observed_at`, `valid_from`, `superseded_by` (link), `tier` (`core | supporting | peripheral`), and `entities` (named entities mentioned, for retrieval boosts).

# Why

The author's knowledge-research taxonomy requires one system to cover facts, experiences, procedures, relationships, policies, preferences, observations, and reflections with provenance, temporal validity, and authority ranking. gnosis already covers procedures, purposes, decisions, and directives; the seven new types close the remaining gaps without new storage subsystems, because every type is a plain OKF page family. The README already promises five foundational types including Concept, which had no bundled definition.

Per-type status fields, rather than one global lifecycle, keep each type's policy exact while the shared field names keep retrieval and validation uniform. Causal and conditional knowledge stay as relationships and policies instead of new page families, which keeps the ontology minimal.

# Constraints

- Each new type is one Concept Type record in `docs/concepts/` with a `path` frontmatter field naming its instance directory: `concepts`, `entities`, `resources`, `events`, `memories`, `reflections`, `policies` — Concept instances share `concepts/` with type definitions, matching the existing flat convention.
- Metadata fields are optional unless a type's lifecycle section requires them; unknown frontmatter is preserved verbatim.
- No new Go storage, index, or retrieval subsystem is introduced; existing page, search, and graph paths must serve the new types unchanged.
- The bundle embeds the new records automatically through the existing `concepts/*.md` glob in `docs/embed.go`.
- The knowledge-research taxonomy document remains a repository artifact; vault records cite it by name instead of linking outside the vault root.
