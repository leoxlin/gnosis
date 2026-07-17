# Maintain a vault

Keep the wiki linked, deduplicated, and fresh.

## Audit and repair

Run `maintain-vault` (`gnosis get procedures gnosis://_/procedures/vault/maintain-vault.md --full`):

1. Baseline: `gnosis validate vault`.
2. Audit orphans, near-duplicates, stale pages, contradictions, tag fragmentation, and broken typed relationships.
3. Apply high-confidence repairs in place through `gnosis apply page`; merge duplicates into the richer page and mark the loser `status: archived` with `superseded_by`.
4. Regenerate indexes (`gnosis index vault`) and log repairs when those options are enabled.
5. Re-validate and report every finding with its disposition.

## Cross-link pages

Run `link-pages` to convert high-confidence unlinked mentions of known titles and aliases into real links in the vault's configured format, adding typed `relationships` only where the text states them explicitly. It links at most the first mention per target and five new links per page — restraint keeps the graph readable.

## Cadence

Cross-link after large ingests; run the full consolidation pass on a schedule or when query results feel noisy.
