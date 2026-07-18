# Knowledge model

gnosis separates three ideas that are often conflated:

- **Knowledge types** — facts, experiences, procedures, relationships, policies, preferences, observations, and reflections. These are what knowledge *means*.
- **Representations** — documents, vectors, graphs, tables, events, code. These are how knowledge is *stored*.
- **Access mechanisms** — lexical search, vector similarity, graph traversal, tool calls. These are how knowledge is *reached*.

A vector database or a wiki is never a knowledge type; it is a representation with access mechanisms. This is why gnosis keeps Markdown authoritative and treats pgvector as a disposable derived index.

## Type coverage

The bundled concept types cover the full taxonomy of agent knowledge: semantic/factual (Concept), episodic and perceptual (Event), procedural (Procedure), causal (typed `causes`/`depends_on` relationships), conditional and normative (Policy), social (Entity), resource and tool (Resource), preference/persona (scoped Memory), metacognitive (`status`/`confidence` metadata plus `^[inferred]`/`^[ambiguous]` markers), and reflective (Reflection). Working and short-term knowledge deliberately never enters the vault — it belongs to the agent's context.

## Authority and provenance

Every page carries provenance (`origin`, content `revision`, optional `source`/`observed_at`/`valid_from`) so retrieval can rank curated and observed knowledge above inference. Supersession (`superseded_by`) and archived records preserve negative knowledge: what was checked and no longer holds.

## Temporal roles

Events and memories carry absolute dates; consolidation turns episodes into Concepts and Reflections through explicit procedures; forgetting is archival with retained audit, never silent deletion.
