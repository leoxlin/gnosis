# Configure semantic search

Lexical (BM25F) search works everywhere with no services. Vector search is an optional derived index in PostgreSQL with pgvector.

## Prerequisites

- PostgreSQL with the `pgvector` extension.
- An OpenAI-compatible embeddings endpoint.

## Environment

Credentials come from the process environment, never from vault configuration:

    export GNOSIS_DATABASE_URL="postgres://user:pass@host:5432/dbname"
    export GNOSIS_EMBEDDING_URL="https://api.example.com/v1/embeddings"
    export GNOSIS_EMBEDDING_MODEL="text-embedding-3-small"
    export GNOSIS_EMBEDDING_API_KEY="..."

## Sync and query

    gnosis index knowledge      # replace this workspace's derived chunks atomically
    gnosis search knowledge "conceptual question" --backend vector

## Notes

- Markdown stays authoritative; the database is disposable derived state. Re-run `gnosis index knowledge` after edits — stale indexes are detected by content fingerprint and reported.
- Without these variables the default vector backend fails fast; pass `--backend lexical` or configure the environment.
- Details and rationale: [knowledge model](../explanation/knowledge-model.md).
