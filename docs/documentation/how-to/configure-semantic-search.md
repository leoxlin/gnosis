# Configure semantic search

Configure the optional vector backend when lexical matching is not sufficient
for conceptual queries.

You need PostgreSQL with the `pgvector` extension and an OpenAI-compatible
embeddings endpoint.

Set the connection and embedding credentials in the process environment:

```bash
export GNOSIS_DATABASE_URL="postgres://user:pass@host:5432/database"
export GNOSIS_EMBEDDING_URL="https://api.example.com/v1/embeddings"
export GNOSIS_EMBEDDING_MODEL="text-embedding-3-small"
export GNOSIS_EMBEDDING_API_KEY="<secret>"
```

Build or replace the derived index for the effective workspace:

```bash
gnosis index knowledge
```

Then query it:

```bash
gnosis search knowledge "<conceptual question>" --backend vector
```

Re-run `gnosis index knowledge` after Markdown changes. gnosis compares content
fingerprints and reports stale derived data. Credentials do not belong in
`gnosis.toml`; Markdown remains authoritative, and the vector index can be
discarded and rebuilt.
