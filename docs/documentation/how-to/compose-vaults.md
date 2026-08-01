# Compose vaults

Create a workspace that presents several local vaults as one deterministic
knowledge view.

## Import local vaults

```bash
mkdir workspace
cd workspace
gnosis apply workspace \
  --import /path/to/team-vault \
  --import /path/to/project-vault
```

Each repeated `--import` adds a `[[vaults]]` entry to
`workspace/gnosis.toml`. Register that composition in the user configuration:

```toml
[[vaults]]
vault_name = "workspace"
vault_root = "/absolute/path/to/workspace"
```

Inspect the effective order:

```bash
gnosis --vault workspace get vaults \
  --fields vault,kind,root,precedence
```

The primary vault has highest precedence, followed by imports in declaration
order and then the embedded core bundle. The first page at a given
vault-relative path wins.

Read an imported page by its own vault name:

```bash
gnosis --vault workspace get pages \
  gnosis://team-vault/references/policy.md --full
```

Use `gnosis://_/path/to/page.md` when a reference should resolve the first
matching page regardless of its vault authority.

## Use a GitHub wiki as the primary vault

```bash
mkdir team-wiki
cd team-wiki
gnosis apply workspace \
  --github-wiki owner/repository \
  --name team-wiki
```

gnosis clones the wiki into a local cache, fast-forwards it on load, and
commits and pushes successful mutations. Coordinate concurrent writers;
non-fast-forward conflicts are returned as errors.

After changing composition, run:

```bash
gnosis validate vault
```

Validation reports import cycles and structural errors.
