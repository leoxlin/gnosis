# Query a vault

Use the bundled query procedure when an answer must come from recorded vault
knowledge.

1. Load the execution contract:

   ```bash
   gnosis get procedures gnosis://_/procedures/query-vault.md --full
   ```

2. Resolve a bounded evidence packet without external services:

   ```bash
   gnosis context knowledge "<question>"
   ```

   Use cited excerpts directly when sufficient. Read cited pages when an excerpt
   is truncated.

3. If evidence context is unavailable or reports a gap, use the lower-level
   lexical search:

   ```bash
   gnosis search knowledge "<question>" --backend lexical
   ```

4. Read only the candidates needed for the answer:

   ```bash
   gnosis get pages <gnosis-uri> --full
   ```

5. For a relationship question, inspect exact links:

   ```bash
   gnosis graph neighbors <gnosis-uri>
   gnosis graph path <from-uri> <to-uri>
   ```

6. Answer from the retrieved pages and cite their canonical URIs. Distinguish
   recorded claims from your own inference.

Use `--max-evidence`, `--max-chars`, and `--depth` to bound evidence context.
Use `--top`, `--max-read`, and `--depth` to bound lower-level retrieval. Use
vector or hybrid strategies only after completing
[Configure semantic search](configure-semantic-search.md).
