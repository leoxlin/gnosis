# Directory Update Log

## 2026-07-10
* **Update**: Renamed `record-purpose` to `refine-purpose` and required one-question-at-a-time author interrogation before purpose edits; see the [workflow decision](repository/decisions/refine-repository-purpose-through-author-interrogation.md) and [Refine purpose skill](repository/deltas/refine-purpose-skill.md) delta.
* **Update**: Renamed the knowledge-driven development bundle from `gnosis-bootstrap` to `gnosis-forge`; see the [naming decision](repository/decisions/name-knowledge-driven-development-bundle-gnosis-forge.md) and [`gnosis-forge` rename](repository/deltas/gnosis-forge-rename.md).
* **Update**: Shortened plugin skill names, added single-concept ingest, and made repository purpose one high-level file with sub-purposes; see [Skill names and purpose refinement](repository/deltas/skill-names-and-purpose.md).

## 2026-07-09
* **Creation**: Added focused `gnosis-vault` management skills and concise `gnosis-bootstrap` repository recorders, captured in the [Plugin skill bootstrap](repository/deltas/plugin-skill-bootstrap.md) delta.
* **Update**: Completed [Harden vault reliability](repository/directives/harden-vault-reliability.md), recorded the [Vault reliability hardening](repository/deltas/vault-reliability-hardening.md) delta, and documented strict configuration, CLI diagnostics, and shared quality checks.
* **Creation**: Added the [Harden vault reliability](repository/directives/harden-vault-reliability.md) directive for strict parsing, safe generated writes, predictable CLI behavior, and automated quality checks.
* **Update**: Moved README setup and quick-start command documentation into [Basic Usage](documentation/basic-usage.md).
* **Creation**: Added the [Documentation](concepts/documentation.md) ontological category for guides, references, runbooks, and contributor or maintainer material.
* **Update**: Rewrote the README and [`gnosis` purpose](repository/purpose.md) around the project name, singular knowledge access, self-bootstrapping repository knowledge, and the relationship to `praxis`.
* **Update**: Trimmed the repository concept types to their reusable essentials and added them as optional scaffold material for other projects.
* **Creation**: Collapsed repository bootstrap decisions into a single [Repository Decision](repository/decisions/bootstrap-knowledge-first.md) documenting the knowledge-first, OKF-based bootstrap and SDLC ontology.
* **Creation**: Added the [Repository Decision](concepts/repository-decision.md) ontological category for durable, triage-level choices.
* **Creation**: Added the [Repository Directive](concepts/repository-directive.md) ontological category for minimal implementation handoffs consumed by agentic loops.
* **Creation**: Added the [Repository Delta](concepts/repository-delta.md) ontological category as the durable trace created when a directive is implemented.
* **Creation**: Added the [Repository Purpose](concepts/repository-purpose.md) ontological category and the [`gnosis`](repository/purpose.md) instance, refined the `gnosis` purpose into a unified agentic-memory interface across LLM wikis, vector RAG, knowledge graphs, structured stores, episodic memory, and future memory backends, and added ingest/query and ontology support.
* **Creation**: Established OKF-compatible `docs/` bundle with root [index](index.md), [OKF concept](references/okf.md), and [OKF v0.1 concept](references/okf-v-0-1.md).
* **Update**: Moved OKF concepts into a dedicated `references/` subdirectory and expanded [OKF v0.1](references/okf-v-0-1.md) to a concise full-spec summary.
* **Update**: Added [LLM Wiki (Karpathy)](references/karpathy-llm-wiki.md) reference summarizing the persistent, LLM-maintained wiki pattern and citing the original gist.
* **Update**: Added [OntoGPT (SPIRES)](references/ontogpt-spires.md) reference summarizing the zero-shot, schema-guided knowledge extraction method and its open-source implementation, with links to the GitHub repository and the SPIRES paper.
