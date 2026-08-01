# Configure semantic search

Configure the optional vector backend when lexical matching is not sufficient
for conceptual queries.

Vector storage defaults to PostgreSQL with the `pgvector` extension. For a
local index with no database server, select SQLite instead. Both use an
OpenAI-compatible embeddings endpoint.

Configure the default pgvector backend:

```bash
export GNOSIS_DATABASE_URL="postgres://user:pass@host:5432/database"
export GNOSIS_EMBEDDING_URL="https://api.example.com/v1/embeddings"
export GNOSIS_EMBEDDING_MODEL="text-embedding-3-small"
export GNOSIS_EMBEDDING_API_KEY="<secret>"
```

Or select SQLite:

```bash
export GNOSIS_VECTOR_BACKEND="sqlite"
export GNOSIS_EMBEDDING_URL="https://api.example.com/v1/embeddings"
export GNOSIS_EMBEDDING_MODEL="text-embedding-3-small"
export GNOSIS_EMBEDDING_API_KEY="<secret>"
```

SQLite defaults to `<user-cache>/gnosis/<vault-name>/semantic.db`. Set
`GNOSIS_SQLITE_VECTOR_PATH` to an absolute path to override it. Do not set
`GNOSIS_DATABASE_URL` when SQLite is selected.

Build or replace the derived index for the effective workspace:

```bash
gnosis index knowledge
```

Then query it:

```bash
gnosis search knowledge "<conceptual question>" --backend vector
```

Re-run `gnosis index knowledge` after Markdown changes or after changing the
embedding model. gnosis compares content fingerprints and reports stale or
incompatible derived data. Credentials do not belong in `gnosis.toml`;
Markdown remains authoritative.

The SQLite database is disposable derived state. Delete it when it is no
longer needed or when gnosis reports an incompatible schema, then run
`gnosis index knowledge` to rebuild it. gnosis never falls back to pgvector or
lexical search when the selected vector backend fails.
