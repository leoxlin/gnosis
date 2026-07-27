# Maintain a vault

Run a maintenance pass after a large import or when search results become
duplicated, stale, or poorly connected.

1. Load the maintenance contract:

   ```bash
   gnosis get procedures gnosis://_/procedures/maintain-vault.md --full
   ```

2. Record the baseline:

   ```bash
   gnosis validate vault
   ```

3. Audit the findings required by the procedure: broken links, orphans,
   duplicate identities, contradictions, stale pages, and fragmented tags.

4. Apply high-confidence repairs through `gnosis apply page`. Merge duplicate
   content into the stronger page and archive the superseded record with a
   `superseded_by` link instead of silently deleting history.

5. If pages mention known records without links, load and follow:

   ```bash
   gnosis get procedures gnosis://_/procedures/link-pages.md --full
   ```

6. Rebuild generated indexes and validate the final state:

   ```bash
   gnosis index vault
   gnosis validate vault
   ```

Report each finding and whether it was repaired, deferred, or left unchanged.
