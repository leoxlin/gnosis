# Ingest GitHub evidence

Configure one repository under the named vault as described in the
[configuration reference](../reference/configuration.md), export the configured
token environment variable, then run a bounded backfill:

```sh
gnosis --vault team ingest github owner/repository \
  --since 2026-07-01T00:00:00Z \
  --max-items 1000
```

The result reports created, unchanged, tombstoned, rejected, rate-limit, and
cursor values. Re-run the same command after interruption or rate limiting.
Durable batches are safe to replay, and the cursor advances only after their
records are stored.

Run a complete deletion reconciliation without backfill bounds:

```sh
gnosis --vault team ingest github owner/repository --reconcile
```

To receive webhooks, configure `webhook_secret_env`, export that secret, and
start the opt-in route:

```sh
gnosis --vault team serve http --github-webhooks
```

Set the GitHub webhook URL to
`https://<host>/api/v1/github/webhooks/owner/repository` and select pull request,
issue, review, comment, and push events. Invalid signatures, oversized bodies,
unsupported events, and conflicting delivery IDs do not create evidence
records.

Treat the configured evidence store as authoritative private source material:

- back it up independently from the Markdown vault;
- restrict filesystem or S3 access to the gnosis operator identity;
- do not edit immutable record files;
- after a crash, rerun the same ingestion command;
- preserve cursor state with its records when restoring.

For ephemeral hosts, select `evidence_backend = "s3"` with `s3_bucket`,
`s3_region`, and an optional `s3_prefix`. The ingestion command and outcomes
are unchanged; immutable writes and tombstones precede each conditional cursor
advance.
