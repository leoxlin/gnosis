# Query the vault

Answer a question from recorded knowledge without scanning the vault.

## Steps

Follow `query-vault` (`gnosis get procedures gnosis://_/procedures/vault/query-vault.md --full`), which applies this cost ladder:

1. **Catalog** — read the root `index.md` when `vault_index` is enabled.
2. **Lexical** — `gnosis search knowledge --backend lexical "<question>"`. An `index_only` answer needs no page reads; a `path` result answers relationship questions from link structure.
3. **Vector** — only when semantic retrieval is configured: `gnosis search knowledge --backend vector "<question>"`, merged by URI.
4. **Read** — open at most the top three `should_read` pages with `gnosis get pages '<URI>' --full`.
5. **Multi-hop** — `gnosis graph neighbors '<URI>'` or `gnosis graph path '<FROM>' '<TO>'` for relationship questions.

## Tips

- Questions about preferences or agent memories belong to `recall`, not the general ladder.
- No candidates means a knowledge gap — report it, or ingest the answer once you establish it.
- Always cite the page paths that support your answer; label synthesis `^[inferred]`.
