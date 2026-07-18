# Ingest knowledge

Turn supplied evidence — documents, transcripts, research — into durable concept pages.

## Steps

1. Load the procedure: `gnosis get procedures gnosis://_/procedures/vault/ingest-knowledge.md --full`, and follow it.
2. List exact types with `gnosis get concepts` and read the Concept Type definitions that apply (`gnosis get pages gnosis://_/concepts/<type>.md --full`).
3. Check identity before creating: `gnosis search knowledge --backend lexical "<the concept>"`. Update the matching page instead of duplicating it.
4. Write the record from its type's schema. Tag claims inline: unmarked for extracted facts, `^[inferred]` for your generalizations, `^[ambiguous]` for unresolved source disagreement.
5. Persist with `gnosis apply page '<record URI>' --filename <draft-file>`. When `vault_log` is enabled, add one newest-first entry to the nearest `log.md`.
6. When `vault_index` is enabled, run `gnosis index vault`. Always finish with `gnosis validate vault`.

## Rules of thumb

- One request about one concept changes exactly one page.
- Keep claims traceable: cite sources in the page body or `source` frontmatter.
- When no existing type fits, do not shoehorn — run the `create-concept-type` procedure.
