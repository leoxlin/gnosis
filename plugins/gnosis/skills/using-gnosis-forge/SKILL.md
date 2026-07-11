---
name: using-gnosis-forge
description: Select and apply the canonical Repository Process for knowledge-driven repository work.
---

# Using gnosis forge

1. Confirm the author explicitly invoked this skill, then resolve the repository vault and read its agent instructions.
2. Read the Repository Process concept definition:

   ```sh
   gnosis read -type 'Concept Type' -title 'Repository Process'
   ```

3. Discover the available Repository Process records:

   ```sh
   gnosis concepts -type 'Repository Process'
   ```

4. Select only the process or process chain that governs the author's request. Read each selected record with:

   ```sh
   gnosis read -type 'Repository Process' -title '<process title>'
   ```

5. Follow the selected record through its completion gate. Ground work in the relevant purpose, decisions, directives, concepts, implementation, and tests named by that record.
