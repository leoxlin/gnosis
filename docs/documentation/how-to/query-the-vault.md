# Query a vault

Use the bundled query procedure when an answer must come from recorded vault
knowledge.

1. Load the execution contract:

   ```bash
   gnosis get procedures gnosis://_/procedures/query-vault.md --full
   ```

2. Search the current Markdown without external services:

   ```bash
   gnosis search knowledge "<question>" --backend lexical
   ```

3. Read only the candidates needed for the answer:

   ```bash
   gnosis get pages <gnosis-uri> --full
   ```

4. For a relationship question, inspect exact links:

   ```bash
   gnosis graph neighbors <gnosis-uri>
   gnosis graph path <from-uri> <to-uri>
   ```

5. Answer from the retrieved pages and cite their canonical URIs. Distinguish
   recorded claims from your own inference.

Use `--top`, `--max-read`, and `--depth` to bound retrieval. Use the vector
backend only after completing
[Configure semantic search](configure-semantic-search.md).
