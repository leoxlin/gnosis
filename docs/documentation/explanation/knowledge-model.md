# Knowledge model

gnosis separates the meaning of knowledge from its storage and retrieval.
Keeping these concerns distinct prevents an implementation choice—such as a
vector database—from becoming the conceptual model.

## Meaning

Concept Types say what a record means. A Concept captures factual knowledge; an
Event captures something that happened; a Procedure tells an agent how to act;
a Policy governs action; a Memory preserves scoped experience or preference.
Typed relationships add causal, dependency, ownership, and other semantic
connections.

This vocabulary can evolve inside the vault because type definitions are
themselves Markdown records.

## Representation

The authoritative representation is typed Markdown. It is inspectable without
gnosis, works with ordinary git history, and can be edited with general-purpose
tools. Canonical URIs separate page identity from a particular filesystem
location.

Indexes, graphs, rendered HTML, and vectors are alternate views or derived
representations. None replaces the source page.

## Access

Different questions need different access mechanisms:

- lexical ranking finds explicit terms in live Markdown;
- vector similarity finds conceptually related passages;
- exact URI reads retrieve a known record;
- graph traversal answers relationship questions;
- MCP exposes the same operations to an agent.

Search therefore returns bounded candidates rather than pretending ranking is
the answer. Exact page reads provide the evidence used to respond.

## Provenance and time

Origin and revision identify where a page came from and which content was
read. Optional source, observation, validity, confidence, and inference markers
explain how strongly a claim is supported. Supersession and archival retain
negative knowledge—what used to be believed or has ceased to apply—instead of
silently erasing it.
