# Ingest knowledge

Use the bundled ingestion procedure to turn supplied evidence into typed,
traceable vault pages.

1. Discover and load the procedure:

   ```bash
   gnosis get procedures --tags gnosis,vault
   gnosis get procedures gnosis://_/procedures/ingest-knowledge.md --full
   ```

2. List the available types, then read the definition for the type you need:

   ```bash
   gnosis get concepts
   gnosis get pages gnosis://_/concepts/concept.md --full
   ```

3. Search for the subject before creating a page:

   ```bash
   gnosis search knowledge "<subject>" --backend lexical
   ```

   Update the matching page instead of creating a duplicate.

4. Draft one page that follows the selected type. Preserve source attribution
   in `source` frontmatter or citations in the body. Mark generalizations with
   `^[inferred]` and unresolved disagreement with `^[ambiguous]`.

5. Apply the draft:

   ```bash
   gnosis apply page <gnosis-uri> --filename <draft.md>
   ```

   Add `--update` only when the page intentionally shadows a lower-precedence
   page.

6. Regenerate enabled indexes and validate:

   ```bash
   gnosis index vault
   gnosis validate vault
   ```

If no type fits the evidence, load
`gnosis://_/procedures/create-concept-type.md` rather than forcing the page
into an unrelated schema.
