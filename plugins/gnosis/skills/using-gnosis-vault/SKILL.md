---
name: using-gnosis-vault
description: Select and apply the canonical Vault Process for work in a gnosis vault.
---

# Using gnosis vault

1. Resolve the current vault and read its agent instructions.
2. Read the Vault Process concept definition:

   ```sh
   gnosis read -type 'Concept Type' -title 'Vault Process'
   ```

3. Discover the available Vault Process records:

   ```sh
   gnosis concepts -type 'Vault Process'
   ```

4. Select only the process or process chain that governs the author's request. Read each selected record with:

   ```sh
   gnosis read -type 'Vault Process' -title '<process title>'
   ```

5. Follow the selected record through its completion gate. Preserve source provenance, repository rules, and current vault configuration.
