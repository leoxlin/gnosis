---
type: Reference
title: OKF v0.1
description: Concise reference for the Open Knowledge Format v0.1 specification.
resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md
tags: [okf, version, v0.1, specification]
timestamp: 2026-07-09T02:50:53Z
---

# OKF v0.1

OKF v0.1 is the first published version of the [Open Knowledge Format (OKF)](./okf.md). It defines a minimal, markdown-based format for representing knowledge as a directory of markdown files with YAML frontmatter.

## Motivation

OKF standardizes the small set of structural conventions needed to make a knowledge corpus self-describing. It is designed to be:

* **Readable** by humans without tooling.
* **Parseable** by agents without bespoke SDKs.
* **Diffable** in version control.
* **Portable** across tools, organizations, and time.

Goals: give enrichment agents a universal write target, inform how consumption agents read and traverse bundles, facilitate exchange across systems, and standardize only the required fields needed for meaningful consumption.

Non-goals: defining a fixed taxonomy of concept types, prescribing storage or query infrastructure, or replacing domain-specific schemas.

## Terminology

| Term | Definition |
|---|---|
| **Knowledge Bundle** | A self-contained, hierarchical collection of knowledge documents; the unit of distribution. |
| **Concept** | A single unit of knowledge, represented as one markdown document. |
| **Concept ID** | The file path within the bundle with the `.md` suffix removed. |
| **Frontmatter** | YAML metadata block at the top of a markdown file, delimited by `---`. |
| **Body** | Everything in the file after the frontmatter. |
| **Link** | Standard markdown link from one concept to another. |
| **Citation** | Link from a concept to an external source supporting a claim. |

## Bundle structure

A bundle is a directory tree of markdown files. The directory structure is domain-independent. Reserved filenames have defined meaning at any level and MUST NOT be used for concept documents:

| Filename | Purpose |
|---|---|
| `index.md` | Directory listing for progressive disclosure. |
| `log.md` | Chronological history of updates. |

Bundles may be distributed as git repositories, archives, or subdirectories within larger repositories.

## Concept documents

Every concept is a UTF-8 markdown file with two parts: YAML frontmatter and a markdown body.

### Frontmatter

```yaml
---
type: <Type name>                  # REQUIRED
title: <Optional display name>
description: <Optional one-line summary>
resource: <Optional canonical URI for the underlying asset>
tags: [<tag>, <tag>, …]            # Optional
timestamp: <ISO 8601 datetime>     # Optional last-modified time
# … other producer-defined key/value pairs
---
```

* `type` is required and identifies the kind of concept. Type values are not centrally registered; consumers MUST tolerate unknown types gracefully.
* `title`, `description`, `resource`, `tags`, and `timestamp` are recommended, in that priority order.
* Producers MAY include additional keys. Consumers SHOULD preserve unknown keys and SHOULD NOT reject documents with unrecognized fields.

### Body

The body is standard markdown. Producers SHOULD favor structural markdown over freeform prose. Conventional section headings:

| Heading | Purpose |
|---|---|
| `# Schema` | Structured description of an asset's columns/fields. |
| `# Examples` | Concrete usage examples, often as fenced code blocks. |
| `# Citations` | External sources backing claims in the body. |

### Examples

A resource-bound concept:

```yaml
---
type: BigQuery Table
title: Customer Orders
description: One row per completed customer order across all channels.
resource: https://console.cloud.google.com/bigquery?p=acme&d=sales&t=orders
tags: [sales, orders, revenue]
timestamp: 2026-05-28T14:30:00Z
---

# Schema

| Column        | Type      | Description                              |
|---------------|-----------|------------------------------------------|
| `order_id`    | STRING    | Globally unique order identifier.        |
| `customer_id` | STRING    | Foreign key into `customers`.            |
| `total_usd`   | NUMERIC   | Order total in US dollars.               |
| `placed_at`   | TIMESTAMP | When the customer submitted the order.   |

# Citations

[1] [BigQuery table schema](https://console.cloud.google.com/bigquery?p=acme&d=sales&t=orders)
```

A concept not bound to a resource:

```yaml
---
type: Playbook
title: Incident response — data freshness alert
description: Steps to triage a freshness alert on the orders pipeline.
tags: [oncall, incident]
timestamp: 2026-04-12T09:00:00Z
---

# Trigger

A freshness alert fires when `orders` lags more than 30 minutes behind its expected SLA.

# Steps

1. Check the ingestion job dashboard.
2. …
```

## Cross-linking

Concepts MAY link to other concepts using standard markdown links:

* **Absolute (bundle-relative)** links begin with `/` and are the recommended form.
* **Relative** links use standard markdown relative paths.

A link asserts a relationship; the surrounding prose conveys the relationship type. Consumers MUST tolerate broken links.

## Index files

An `index.md` MAY appear in any directory, including the bundle root. It enumerates directory contents to support progressive disclosure. Index files contain no frontmatter. Example:

```markdown
# Section / Group Heading

* [Title 1](relative-url-1) - short description of item 1
* [Title 2](relative-url-2) - short description of item 2
```

## Log files

A `log.md` MAY appear at any level to record history. Format is a flat list of date-grouped entries, newest first:

```markdown
# Directory Update Log

## 2026-05-22
* **Update**: Added new BigQuery table reference for Customer Metrics.
* **Creation**: Established the Dataplex Playbook.
```

Date headings MUST use ISO 8601 `YYYY-MM-DD` form.

## Citations

External sources SHOULD be listed under a `# Citations` heading at the bottom of the document, numbered:

```markdown
# Citations

[1] [BigQuery public dataset announcement](https://cloud.google.com/blog/products/data-analytics/...)
```

Citation links MAY be absolute URLs, bundle-relative paths, or paths into a `references/` subdirectory.

## Conformance

A bundle is conformant with OKF v0.1 if:

1. Every non-reserved `.md` file contains parseable YAML frontmatter.
2. Every frontmatter block contains a non-empty `type` field.
3. Reserved filenames (`index.md`, `log.md`) follow their defined structures when present.

Consumers MUST NOT reject a bundle because of missing optional fields, unknown type values, unknown frontmatter keys, broken cross-links, or missing `index.md` files.

## Relationship to other formats

OKF is close to LLM wiki repositories, personal knowledge tools such as Obsidian, and metadata-as-code approaches. It differs in being explicitly specified — pinning down interoperability rules without dictating tooling.

## Versioning

OKF is versioned as `<major>.<minor>`. Minor revisions are backward-compatible; major revisions may introduce breaking changes. Bundles MAY declare the targeted version by including `okf_version: "0.1"` in the bundle-root `index.md` frontmatter block; this is the only place frontmatter is permitted in an `index.md`.

# Citations

[1] [OKF v0.1 specification](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
