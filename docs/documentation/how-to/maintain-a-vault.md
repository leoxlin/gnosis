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

3. Run the bounded read-only audit:

   ```bash
   gnosis audit knowledge
   ```

   Follow `next_cursor` until the result is complete. Audit staleness with an
   explicit policy, for example:

   ```bash
   gnosis audit knowledge --class stale --tier core --tier supporting \
     --stale-after 2160h
   ```

   If the command is unavailable, inspect the same broken supersession,
   orphan, duplicate-identity, contradiction, stale, archived-reference, and
   fragmented-tag signals manually.

4. Review the evidence and classification on every finding. Apply
   high-confidence structural repairs through `gnosis apply page`. Merge duplicate
   content into the stronger page and archive the superseded record with a
   `superseded_by` link instead of silently deleting history. Treat candidates
   as author decisions, not proven identities or meaning conflicts.

   Authored maintenance annotations appear with the `authored` classification:

   ```yaml
   maintenance:
     - kind: stale
       reason: Upstream policy changed.
       observed_at: 2026-07-29T09:00:00Z
       author: agent-id
     - kind: duplicate
       reason: The canonical page contains the same claim.
       observed_at: 2026-07-29T09:01:00Z
       target: gnosis://vault/policies/canonical.md
   ```

   Valid kinds are `stale`, `incorrect`, and `duplicate`. Every annotation
   requires `reason` and an RFC3339 `observed_at`; only `duplicate` accepts and
   requires a distinct, resolvable canonical `target` when written. Audits
   report these judgments and broken targets without changing annotations.

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
