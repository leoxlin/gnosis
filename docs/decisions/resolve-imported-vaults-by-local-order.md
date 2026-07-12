---
type: Decision
title: Resolve imported vaults by local order
description: Compose local vault directories and recursive imports in declared order, with earlier vault-relative pages taking precedence.
---

# Decision

Resolve a gnosis workspace from its local vault directories first, then its imported vaults in declaration order. Resolve each imported vault's own imports recursively in the same order. When two pages share a vault-relative path, retain the first page encountered.

# Why

Vault composition must preserve each vault as an independently usable OKF bundle while giving the workspace author an explicit and stable way to select a preferred version of overlapping knowledge. A depth-first traversal follows the local configuration that introduced an import and makes the effective knowledge view explainable from configuration order.

# Constraints

- Every local import target must be a vault with its own `gnosis.toml`.
- Import resolution detects cycles and de-duplicates a vault reached by more than one path.
- Duplicate titles remain distinct pages; only equal vault-relative paths are resolved by precedence.
- Remote import URLs are represented in configuration but are not resolved in this iteration.

# Rejected alternatives

- **Unordered merging** — rejected because page selection would be unstable and impossible for authors to control.
- **Title-based de-duplication** — rejected because different concepts can legitimately share a title.
- **Flattening imported vault contents into the local vault** — rejected because it breaks independent ownership and reuse of OKF bundles.

# Related decisions

- [`gnosis` purpose](../purpose.md)
- [Keep search sources and retrieval backends replaceable](keep-search-sources-and-retrieval-backends-replaceable.md)
